package replication

import (
	"bufio"
	"net"
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
	values map[string]string
}

func (s *memoryReplicaStore) Set(key, value string) error {
	s.values[key] = value
	return nil
}

func (s *memoryReplicaStore) Get(key string) (string, bool) {
	value, ok := s.values[key]
	return value, ok
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