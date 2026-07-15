package replication

import (
	"bufio"
	"net"
	"testing"
	"time"
)

func TestPrimarySendsSnapshotBeforeLiveUpdates(t *testing.T) {
	primary := NewPrimary(&snapshotStore{data: map[string]string{"name": "bolt"}}, nil)
	primary.heartbeatInterval = time.Hour

	serverConn, replicaConn := net.Pipe()
	defer serverConn.Close()
	defer replicaConn.Close()

	lines := make(chan string, 4)
	go func() {
		reader := bufio.NewReader(replicaConn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			lines <- line[:len(line)-1]
		}
	}()

	go primary.AcceptReplica(serverConn)

	if line := readLineFromChannel(t, lines); line != "SNAPSHOT BEGIN" {
		t.Fatalf("expected snapshot begin, got %q", line)
	}
	if line := readLineFromChannel(t, lines); line != "SET\t4:name\t4:bolt" {
		t.Fatalf("expected snapshot record, got %q", line)
	}
	if line := readLineFromChannel(t, lines); line != "SNAPSHOT END" {
		t.Fatalf("expected snapshot end, got %q", line)
	}

	primary.OnSet("language", "Go")
	select {
	case line := <-lines:
		if line != "SET\t8:language\t2:Go" {
			t.Fatalf("expected live update, got %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live update")
	}
}

type snapshotStore struct {
	data map[string]string
}

func (s *snapshotStore) Snapshot() map[string]string {
	copy := make(map[string]string, len(s.data))
	for key, value := range s.data {
		copy[key] = value
	}
	return copy
}

func readLineFromChannel(t *testing.T, lines <-chan string) string {
	t.Helper()

	select {
	case line := <-lines:
		return line
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replication line")
		return ""
	}
}