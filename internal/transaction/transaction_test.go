package transaction

import (
	"testing"

	"github.com/stevenstank/bolt/internal/protocol"
)

func TestTransactionStartMulti(t *testing.T) {
	tx := New()
	if tx.InMulti() {
		t.Fatal("expected transaction to not be in MULTI mode initially")
	}

	tx.StartMulti()
	if !tx.InMulti() {
		t.Fatal("expected transaction to be in MULTI mode after StartMulti")
	}
}

func TestTransactionQueue(t *testing.T) {
	tx := New()
	tx.StartMulti()

	cmd := protocol.Command{Name: "SET", Args: []string{"key", "value"}}
	tx.Queue(cmd)

	if tx.QueuedCount() != 1 {
		t.Fatalf("expected 1 queued command, got %d", tx.QueuedCount())
	}
}

func TestTransactionExec(t *testing.T) {
	tx := New()
	tx.StartMulti()

	cmd1 := protocol.Command{Name: "SET", Args: []string{"key1", "value1"}}
	cmd2 := protocol.Command{Name: "SET", Args: []string{"key2", "value2"}}
	tx.Queue(cmd1)
	tx.Queue(cmd2)

	commands := tx.Exec()
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if tx.InMulti() {
		t.Fatal("expected transaction to exit MULTI mode after Exec")
	}
	if tx.QueuedCount() != 0 {
		t.Fatal("expected queue to be cleared after Exec")
	}
}

func TestTransactionDiscard(t *testing.T) {
	tx := New()
	tx.StartMulti()

	cmd := protocol.Command{Name: "SET", Args: []string{"key", "value"}}
	tx.Queue(cmd)

	tx.Discard()
	if tx.InMulti() {
		t.Fatal("expected transaction to exit MULTI mode after Discard")
	}
	if tx.QueuedCount() != 0 {
		t.Fatal("expected queue to be cleared after Discard")
	}
}

func TestTransactionDiscardWithoutMulti(t *testing.T) {
	tx := New()
	tx.Discard() // Should not panic
	if tx.InMulti() {
		t.Fatal("expected transaction to not be in MULTI mode")
	}
}

func TestTransactionExecWithoutMulti(t *testing.T) {
	tx := New()
	commands := tx.Exec()
	if len(commands) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(commands))
	}
}
