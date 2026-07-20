package replication

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stevenstank/bolt/internal/record"
)

// Snapshotter exposes a consistent copy of data for replica bootstrapping.
type Snapshotter interface {
	Snapshot() map[string]record.Entry
}

// SetApplier applies replicated writes.
type SetApplier interface {
	ApplySet(key, value string) error
	ApplySetWithExpiry(key, value string, expiresAt time.Time) error
}

// ReplicaConfig configures a background replica connection.
type ReplicaConfig struct {
	PrimaryAddr string
	Store       SetApplier
	Dial        func(addr string) (net.Conn, error)
	RetryDelay  time.Duration
}

// Primary manages replica connections for a primary node.
type Primary struct {
	store Snapshotter
	logger *log.Logger

	mu                 sync.Mutex
	replicas           map[*replicaConn]struct{}
	heartbeatInterval time.Duration
}

type replicaConn struct {
	conn net.Conn
	mu   sync.Mutex
}

// NewPrimary creates a primary replication manager.
func NewPrimary(store Snapshotter, logger *log.Logger) *Primary {
	return &Primary{
		store:             store,
		logger:            logger,
		replicas:          make(map[*replicaConn]struct{}),
		heartbeatInterval: time.Second,
	}
}

// AcceptReplica registers a new replica connection and sends its initial snapshot.
func (p *Primary) AcceptReplica(conn net.Conn) {
	replica := &replicaConn{conn: conn}

	p.mu.Lock()
	if err := p.writeSnapshotLocked(replica); err != nil {
		p.mu.Unlock()
		_ = conn.Close()
		return
	}
	p.replicas[replica] = struct{}{}
	p.mu.Unlock()

	go p.readReplica(replica)
	go p.sendHeartbeats(replica)
}

func (p *Primary) ConnectedReplicas() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.replicas)
}

func (p *Primary) ReplicationStatus() string {
	if p.ConnectedReplicas() > 0 {
		return "connected"
	}
	return "waiting"
}

func (r *Replica) ConnectedReplicas() int {
	return 0
}

func (r *Replica) ReplicationStatus() string {
	return "connected"
}

// OnSet broadcasts a replicated SET to connected replicas.
func (p *Primary) OnSet(key, value string) {
	p.OnSetWithExpiry(key, value, time.Time{})
}

// OnSetWithExpiry broadcasts a replicated SET with TTL to connected replicas.
func (p *Primary) OnSetWithExpiry(key, value string, expiresAt time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for replica := range p.replicas {
		if err := replica.writeLine(formatSetRecordWithExpiry(key, value, expiresAt)); err != nil {
			delete(p.replicas, replica)
			_ = replica.conn.Close()
		}
	}
}

// ApplySet is a no-op for primary (it doesn't apply writes).
func (p *Primary) ApplySet(key, value string) error {
	return nil
}

// ApplySetWithExpiry is a no-op for primary (it doesn't apply writes).
func (p *Primary) ApplySetWithExpiry(key, value string, expiresAt time.Time) error {
	return nil
}

func (p *Primary) writeSnapshotLocked(replica *replicaConn) error {
	if err := replica.writeLine("SNAPSHOT BEGIN"); err != nil {
		return err
	}

	snapshot := p.store.Snapshot()
	keys := make([]string, 0, len(snapshot))
	for key := range snapshot {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entry := snapshot[key]
		if err := replica.writeLine(formatSetRecordWithExpiry(key, entry.Value, entry.ExpiresAt)); err != nil {
			return err
		}
	}

	return replica.writeLine("SNAPSHOT END")
}

func (p *Primary) readReplica(replica *replicaConn) {
	scanner := bufio.NewScanner(replica.conn)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "PONG" {
			continue
		}
	}
	p.removeReplica(replica)
}

func (p *Primary) sendHeartbeats(replica *replicaConn) {
	ticker := time.NewTicker(p.heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if _, ok := p.replicas[replica]; !ok {
			p.mu.Unlock()
			return
		}
		err := replica.writeLine("PING")
		if err != nil {
			delete(p.replicas, replica)
			_ = replica.conn.Close()
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()
	}
}

func (p *Primary) removeReplica(replica *replicaConn) {
	p.mu.Lock()
	delete(p.replicas, replica)
	p.mu.Unlock()
	_ = replica.conn.Close()
}

func (r *replicaConn) writeLine(line string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := fmt.Fprintln(r.conn, line)
	return err
}

// Replica connects to a primary and applies replicated updates.
type Replica struct {
	config      ReplicaConfig
	logger      *log.Logger
	retryDelay  time.Duration
	stop        chan struct{}
	once        sync.Once
}

// NewReplica creates a background replica client.
func NewReplica(config ReplicaConfig, logger *log.Logger) *Replica {
	if config.Dial == nil {
		config.Dial = func(addr string) (net.Conn, error) {
			return net.Dial("tcp", addr)
		}
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 250 * time.Millisecond
	}

	return &Replica{
		config:     config,
		logger:     logger,
		retryDelay: config.RetryDelay,
		stop:       make(chan struct{}),
	}
}

// Run keeps the replica connected and retries after disconnects.
func (r *Replica) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := r.config.Dial(r.config.PrimaryAddr)
		if err != nil {
			if !r.sleep(ctx) {
				return
			}
			continue
		}

		if _, err := fmt.Fprintln(conn, "SYNC"); err != nil {
			_ = conn.Close()
			if !r.sleep(ctx) {
				return
			}
			continue
		}

		if err := r.handleConnection(conn); err != nil && !isClosedError(err) {
			if r.logger != nil {
				r.logger.Printf("replica connection error: %v", err)
			}
		}
		_ = conn.Close()

		if !r.sleep(ctx) {
			return
		}
	}
}

func (r *Replica) handleConnection(conn net.Conn) error {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if err := r.processLine(scanner.Text(), conn); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (r *Replica) processLine(line string, conn net.Conn) error {
	line = strings.TrimSpace(line)
	switch {
	case line == "SNAPSHOT BEGIN":
		return nil
	case line == "SNAPSHOT END":
		return nil
	case line == "PING":
		_, err := fmt.Fprintln(conn, "PONG")
		return err
	case strings.HasPrefix(line, "SET\t"):
		key, expiresAt, value, err := parseSetRecordWithExpiry(line)
		if err != nil {
			return err
		}
		if expiresAt.IsZero() {
			if err := r.config.Store.ApplySet(key, value); err != nil {
				return err
			}
		} else {
			if err := r.config.Store.ApplySetWithExpiry(key, value, expiresAt); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown replication line: %q", line)
	}
}

func (r *Replica) sleep(ctx context.Context) bool {
	timer := time.NewTimer(r.retryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "closed network connection")
}

func formatSetRecord(key, value string) string {
	return formatSetRecordWithExpiry(key, value, time.Time{})
}

func formatSetRecordWithExpiry(key, value string, expiresAt time.Time) string {
	if expiresAt.IsZero() {
		return fmt.Sprintf("SET\t%d:%s\t%d:%s", len(key), key, len(value), value)
	}
	expiresAtText := strconv.FormatInt(expiresAt.UnixNano(), 10)
	return fmt.Sprintf("SET\t%d:%s\t%d:%s\t%d:%s", len(key), key, len(expiresAtText), expiresAtText, len(value), value)
}

func parseSetRecord(line string) (string, string, error) {
	key, _, value, err := parseSetRecordWithExpiry(line)
	return key, value, err
}

func parseSetRecordWithExpiry(line string) (string, time.Time, string, error) {
	parts := strings.Split(line, "\t")
	if parts[0] != "SET" || (len(parts) != 3 && len(parts) != 4) {
		return "", time.Time{}, "", fmt.Errorf("invalid replication record: %q", line)
	}

	key, err := parseLengthPrefixedField(parts[1])
	if err != nil {
		return "", time.Time{}, "", err
	}

	var expiresAt time.Time
	var valueField string
	if len(parts) == 3 {
		valueField = parts[2]
	} else {
		expiresAtText, err := parseLengthPrefixedField(parts[2])
		if err != nil {
			return "", time.Time{}, "", err
		}
		expiresAtUnixNano, err := strconv.ParseInt(expiresAtText, 10, 64)
		if err != nil {
			return "", time.Time{}, "", fmt.Errorf("invalid replication expiry %q: %w", expiresAtText, err)
		}
		expiresAt = time.Unix(0, expiresAtUnixNano)
		valueField = parts[3]
	}

	value, err := parseLengthPrefixedField(valueField)
	if err != nil {
		return "", time.Time{}, "", err
	}
	return key, expiresAt, value, nil
}

func parseLengthPrefixedField(field string) (string, error) {
	lengthText, value, ok := strings.Cut(field, ":")
	if !ok {
		return "", fmt.Errorf("invalid replication field: %q", field)
	}

	length, err := strconv.Atoi(lengthText)
	if err != nil {
		return "", fmt.Errorf("invalid replication field length %q: %w", lengthText, err)
	}
	if length != len(value) {
		return "", fmt.Errorf("invalid replication field length: expected %d bytes, got %d", length, len(value))
	}
	return value, nil
}