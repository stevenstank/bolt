package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stevenstank/bolt/internal/protocol"
	"github.com/stevenstank/bolt/internal/pubsub"
	"github.com/stevenstank/bolt/internal/transaction"
)

type storageEngine interface {
	Set(key, value string) error
	SetWithExpiry(key, value string, expiresAt time.Time) error
	Get(key string) (string, bool)
	KeyCount() int
	MemoryUsage() int64
}

type pubsubClient interface {
	Subscribe(channel string) error
	Unsubscribe(channel string) error
	Publish(channel, message string) (int, error)
}

type InfoProvider interface {
	NodeID() string
	Role() string
	Uptime() int64
	ConnectedClients() int
	ReplicationStatus() string
	ConnectedReplicas() int
}

// Dispatcher validates commands and routes them to storage.
type Dispatcher struct {
	engine       storageEngine
	pubsub       pubsubClient
	subscriber   *pubsub.Subscriber
	transaction  *transaction.Transaction
	info         InfoProvider
}

// NewDispatcher creates a command dispatcher.
func NewDispatcher(engine storageEngine) *Dispatcher {
	return &Dispatcher{
		engine:      engine,
		transaction: transaction.New(),
	}
}

// NewDispatcherWithPubsub creates a command dispatcher with pubsub support.
func NewDispatcherWithPubsub(engine storageEngine, pubsub pubsubClient, subscriber *pubsub.Subscriber) *Dispatcher {
	return &Dispatcher{
		engine:      engine,
		pubsub:      pubsub,
		subscriber:  subscriber,
		transaction: transaction.New(),
	}
}

// SetInfo sets the info provider for this dispatcher.
func (d *Dispatcher) SetInfo(info InfoProvider) {
	d.info = info
}

// SetTransaction sets the transaction for this dispatcher (used for per-client isolation).
func (d *Dispatcher) SetTransaction(tx *transaction.Transaction) {
	d.transaction = tx
}

// Dispatch executes a parsed command and returns a plain-text response.
func (d *Dispatcher) Dispatch(cmd protocol.Command) string {
	// Handle transaction commands
	switch cmd.Name {
	case "MULTI":
		if d.transaction.InMulti() {
			return "ERR already in MULTI mode"
		}
		d.transaction.StartMulti()
		return "OK"
	case "EXEC":
		if !d.transaction.InMulti() {
			return "ERR not in MULTI mode"
		}
		return d.executeTransaction()
	case "DISCARD":
		if !d.transaction.InMulti() {
			return "ERR not in MULTI mode"
		}
		d.transaction.Discard()
		return "OK"
	}

	// If in MULTI mode, queue the command
	if d.transaction.InMulti() {
		d.transaction.Queue(cmd)
		return "QUEUED"
	}

	// Execute command immediately
	return d.executeCommand(cmd)
}

func (d *Dispatcher) executeCommand(cmd protocol.Command) string {
	switch cmd.Name {
	case "SET":
		if len(cmd.Args) == 2 {
			if err := d.engine.Set(cmd.Args[0], cmd.Args[1]); err != nil {
				return fmt.Sprintf("ERR %v", err)
			}
			return "OK"
		}
		if len(cmd.Args) == 4 && strings.EqualFold(cmd.Args[2], "EX") {
			seconds, err := strconv.Atoi(cmd.Args[3])
			if err != nil {
				return fmt.Sprintf("ERR invalid EX seconds %q", cmd.Args[3])
			}
			if err := d.engine.SetWithExpiry(cmd.Args[0], cmd.Args[1], time.Now().Add(time.Duration(seconds)*time.Second)); err != nil {
				return fmt.Sprintf("ERR %v", err)
			}
			return "OK"
		}
		if len(cmd.Args) < 2 {
			return "ERR SET requires key and value"
		}
		return "ERR SET supports optional EX seconds"
	case "GET":
		if len(cmd.Args) != 1 {
			return "ERR GET requires key"
		}
		value, ok := d.engine.Get(cmd.Args[0])
		if !ok {
			return "(nil)"
		}
		return value
	case "INFO":
		return d.handleInfo()
	case "SUBSCRIBE":
		if d.pubsub == nil {
			return "ERR pubsub not enabled"
		}
		if len(cmd.Args) != 1 {
			return "ERR SUBSCRIBE requires channel"
		}
		if err := d.pubsub.Subscribe(cmd.Args[0]); err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		return fmt.Sprintf("OK subscribed to %s", cmd.Args[0])
	case "UNSUBSCRIBE":
		if d.pubsub == nil {
			return "ERR pubsub not enabled"
		}
		if len(cmd.Args) != 1 {
			return "ERR UNSUBSCRIBE requires channel"
		}
		if err := d.pubsub.Unsubscribe(cmd.Args[0]); err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		return fmt.Sprintf("OK unsubscribed from %s", cmd.Args[0])
	case "PUBLISH":
		if d.pubsub == nil {
			return "ERR pubsub not enabled"
		}
		if len(cmd.Args) != 2 {
			return "ERR PUBLISH requires channel and message"
		}
		count, err := d.pubsub.Publish(cmd.Args[0], cmd.Args[1])
		if err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		return fmt.Sprintf("%d", count)
	default:
		return fmt.Sprintf("ERR unknown command %q", cmd.Name)
	}
}

func (d *Dispatcher) executeTransaction() string {
	commands := d.transaction.Exec()
	if len(commands) == 0 {
		return "OK"
	}

	var results []string
	for _, cmd := range commands {
		result := d.executeCommand(cmd)
		results = append(results, result)
	}

	// Return results as a multi-line response
	return strings.Join(results, "\n")
}

func (d *Dispatcher) handleInfo() string {
	var lines []string

	// Node ID
	if d.info != nil {
		lines = append(lines, fmt.Sprintf("node_id: %s", d.info.NodeID()))
	} else {
		lines = append(lines, "node_id: unknown")
	}

	// Role
	if d.info != nil {
		lines = append(lines, fmt.Sprintf("role: %s", d.info.Role()))
	} else {
		lines = append(lines, "role: primary")
	}

	// Uptime
	if d.info != nil {
		lines = append(lines, fmt.Sprintf("uptime: %d", d.info.Uptime()))
	} else {
		lines = append(lines, "uptime: 0")
	}

	// Connected clients
	if d.info != nil {
		lines = append(lines, fmt.Sprintf("connected_clients: %d", d.info.ConnectedClients()))
	} else {
		lines = append(lines, "connected_clients: 0")
	}

	// Replication status
	if d.info != nil {
		lines = append(lines, fmt.Sprintf("replication_status: %s", d.info.ReplicationStatus()))
	} else {
		lines = append(lines, "replication_status: disabled")
	}

	// Connected replicas
	if d.info != nil {
		lines = append(lines, fmt.Sprintf("connected_replicas: %d", d.info.ConnectedReplicas()))
	} else {
		lines = append(lines, "connected_replicas: 0")
	}

	// Key count
	lines = append(lines, fmt.Sprintf("key_count: %d", d.engine.KeyCount()))

	// Memory usage
	lines = append(lines, fmt.Sprintf("memory_usage: %d", d.engine.MemoryUsage()))

	return strings.Join(lines, "\n")
}
