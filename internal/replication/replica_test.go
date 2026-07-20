package replication

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stevenstank/bolt/internal/engine"
)

func TestReplicaAppliesSnapshotAndAcknowledgesHeartbeats(t *testing.T) {
	store := &memoryReplicaStore{values: map[string]string{}}
	eng := engine.New(store)
	eng.SetReadOnly(true)
	replica := NewReplica(ReplicaConfig{Store: eng}, nil)

	primaryConn, replicaConn := net.Pipe()
	defer primaryConn.Close()
	defer replicaConn.Close()

	go func() {
		_, _ = primaryConn.Write([]byte("SNAPSHOT BEGIN\nSET\t4:name\t4:bolt\nSNAPSHOT END\nPING\n"))
	}()

	go func() {
		_ = replica.handleConnection(replicaConn)
	}()

	waitForValue(t, time.Second, func() bool {
		got, ok := store.Get("name")
		return ok && got == "bolt"
	})

	if err := primaryConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	line, err := bufio.NewReader(primaryConn).ReadString('\n')
	if err != nil {
		t.Fatalf("read heartbeat response: %v", err)
	}
	if line != "PONG\n" {
		t.Fatalf("expected heartbeat response %q, got %q", "PONG\n", line)
	}
}

type memoryReplicaStore struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *memoryReplicaStore) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *memoryReplicaStore) SetWithExpiry(key, value string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *memoryReplicaStore) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	return value, ok
}

func (s *memoryReplicaStore) ApplySet(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *memoryReplicaStore) ApplySetWithExpiry(key, value string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func waitForValue(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}

func TestReplicaAppliesSetWithExpiry(t *testing.T) {
	store := &memoryReplicaStore{values: map[string]string{}}
	eng := engine.New(store)
	eng.SetReadOnly(true)
	replica := NewReplica(ReplicaConfig{Store: eng}, nil)

	primaryConn, replicaConn := net.Pipe()
	defer primaryConn.Close()
	defer replicaConn.Close()

	expiresAt := time.Now().Add(time.Hour)
	expiresAtText := strconv.FormatInt(expiresAt.UnixNano(), 10)

	go func() {
		_, _ = primaryConn.Write([]byte(fmt.Sprintf("SET\t4:name\t%d:%s\t%d:%s\n", len(expiresAtText), expiresAtText, 4, "bolt")))
	}()

	go func() {
		_ = replica.handleConnection(replicaConn)
	}()

	waitForValue(t, time.Second, func() bool {
		got, ok := store.Get("name")
		return ok && got == "bolt"
	})
}