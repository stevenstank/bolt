package command

import (
	"strings"
	"testing"

	"github.com/stevenstank/bolt/internal/storage"
)

type mockInfoProvider struct {
	nodeID              string
	role                string
	uptime              int64
	connectedClients    int
	replicationStatus   string
	connectedReplicas   int
}

func (m *mockInfoProvider) NodeID() string {
	return m.nodeID
}

func (m *mockInfoProvider) Role() string {
	return m.role
}

func (m *mockInfoProvider) Uptime() int64 {
	return m.uptime
}

func (m *mockInfoProvider) ConnectedClients() int {
	return m.connectedClients
}

func (m *mockInfoProvider) ReplicationStatus() string {
	return m.replicationStatus
}

func (m *mockInfoProvider) ConnectedReplicas() int {
	return m.connectedReplicas
}

func TestInfoCommand(t *testing.T) {
	store := storage.NewStore()
	dispatcher := NewDispatcher(store)
	
	info := &mockInfoProvider{
		nodeID:            "test-node-123",
		role:              "primary",
		uptime:            100,
		connectedClients:  5,
		replicationStatus: "connected",
		connectedReplicas: 2,
	}
	dispatcher.SetInfo(info)

	// Set some test data
	store.Set("key1", "value1")
	store.Set("key2", "value2")

	processor := NewProcessor(dispatcher)
	response := processor.Process("INFO")

	lines := strings.Split(response, "\n")
	
	// Verify all expected fields are present
	fields := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			fields[parts[0]] = parts[1]
		}
	}

	if fields["node_id"] != "test-node-123" {
		t.Errorf("expected node_id test-node-123, got %s", fields["node_id"])
	}
	if fields["role"] != "primary" {
		t.Errorf("expected role primary, got %s", fields["role"])
	}
	if fields["uptime"] != "100" {
		t.Errorf("expected uptime 100, got %s", fields["uptime"])
	}
	if fields["connected_clients"] != "5" {
		t.Errorf("expected connected_clients 5, got %s", fields["connected_clients"])
	}
	if fields["replication_status"] != "connected" {
		t.Errorf("expected replication_status connected, got %s", fields["replication_status"])
	}
	if fields["connected_replicas"] != "2" {
		t.Errorf("expected connected_replicas 2, got %s", fields["connected_replicas"])
	}
	if fields["key_count"] != "2" {
		t.Errorf("expected key_count 2, got %s", fields["key_count"])
	}
}

func TestInfoCommandWithoutInfoProvider(t *testing.T) {
	store := storage.NewStore()
	dispatcher := NewDispatcher(store)
	
	// Don't set info provider - should use defaults

	processor := NewProcessor(dispatcher)
	response := processor.Process("INFO")

	lines := strings.Split(response, "\n")
	
	fields := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			fields[parts[0]] = parts[1]
		}
	}

	// Verify defaults
	if fields["node_id"] != "unknown" {
		t.Errorf("expected default node_id unknown, got %s", fields["node_id"])
	}
	if fields["role"] != "primary" {
		t.Errorf("expected default role primary, got %s", fields["role"])
	}
	if fields["uptime"] != "0" {
		t.Errorf("expected default uptime 0, got %s", fields["uptime"])
	}
	if fields["connected_clients"] != "0" {
		t.Errorf("expected default connected_clients 0, got %s", fields["connected_clients"])
	}
	if fields["replication_status"] != "disabled" {
		t.Errorf("expected default replication_status disabled, got %s", fields["replication_status"])
	}
	if fields["connected_replicas"] != "0" {
		t.Errorf("expected default connected_replicas 0, got %s", fields["connected_replicas"])
	}
}

func TestInfoCommandInTransaction(t *testing.T) {
	store := storage.NewStore()
	dispatcher := NewDispatcher(store)
	
	info := &mockInfoProvider{
		nodeID:            "test-node-456",
		role:              "replica",
		uptime:            200,
		connectedClients:  3,
		replicationStatus: "disabled",
		connectedReplicas: 0,
	}
	dispatcher.SetInfo(info)

	processor := NewProcessor(dispatcher)

	// Start transaction
	processor.Process("MULTI")
	
	// INFO should be queued in transaction
	response := processor.Process("INFO")
	if response != "QUEUED" {
		t.Errorf("expected QUEUED, got %s", response)
	}

	// Execute transaction
	response = processor.Process("EXEC")
	if !strings.Contains(response, "node_id: test-node-456") {
		t.Errorf("expected INFO result in EXEC output, got %s", response)
	}
}
