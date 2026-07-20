package command

import (
	"testing"

	"github.com/stevenstank/bolt/internal/engine"
	"github.com/stevenstank/bolt/internal/protocol"
	"github.com/stevenstank/bolt/internal/storage"
)

func TestDispatcherMulti(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	resp := dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	if resp != "OK" {
		t.Fatalf("expected OK, got %s", resp)
	}

	if !dispatcher.transaction.InMulti() {
		t.Fatal("expected transaction to be in MULTI mode")
	}
}

func TestDispatcherMultiTwice(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	resp := dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	if resp != "ERR already in MULTI mode" {
		t.Fatalf("expected error, got %s", resp)
	}
}

func TestDispatcherQueueCommands(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	dispatcher.Dispatch(protocol.Command{Name: "MULTI"})

	resp := dispatcher.Dispatch(protocol.Command{Name: "SET", Args: []string{"key", "value"}})
	if resp != "QUEUED" {
		t.Fatalf("expected QUEUED, got %s", resp)
	}

	if dispatcher.transaction.QueuedCount() != 1 {
		t.Fatalf("expected 1 queued command, got %d", dispatcher.transaction.QueuedCount())
	}

	// Key should not be set yet
	_, ok := store.Get("key")
	if ok {
		t.Fatal("key should not be set during transaction")
	}
}

func TestDispatcherExec(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	dispatcher.Dispatch(protocol.Command{Name: "SET", Args: []string{"key1", "value1"}})
	dispatcher.Dispatch(protocol.Command{Name: "SET", Args: []string{"key2", "value2"}})

	resp := dispatcher.Dispatch(protocol.Command{Name: "EXEC"})
	if resp != "OK\nOK" {
		t.Fatalf("expected 'OK\\nOK', got %s", resp)
	}

	// Keys should now be set
	val, ok := store.Get("key1")
	if !ok || val != "value1" {
		t.Fatalf("key1 not set correctly")
	}

	val, ok = store.Get("key2")
	if !ok || val != "value2" {
		t.Fatalf("key2 not set correctly")
	}

	if dispatcher.transaction.InMulti() {
		t.Fatal("transaction should not be in MULTI mode after EXEC")
	}
}

func TestDispatcherExecWithoutMulti(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	resp := dispatcher.Dispatch(protocol.Command{Name: "EXEC"})
	if resp != "ERR not in MULTI mode" {
		t.Fatalf("expected error, got %s", resp)
	}
}

func TestDispatcherDiscard(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	dispatcher.Dispatch(protocol.Command{Name: "SET", Args: []string{"key", "value"}})

	resp := dispatcher.Dispatch(protocol.Command{Name: "DISCARD"})
	if resp != "OK" {
		t.Fatalf("expected OK, got %s", resp)
	}

	if dispatcher.transaction.InMulti() {
		t.Fatal("transaction should not be in MULTI mode after DISCARD")
	}

	if dispatcher.transaction.QueuedCount() != 0 {
		t.Fatal("queue should be empty after DISCARD")
	}

	// Key should not be set
	_, ok := store.Get("key")
	if ok {
		t.Fatal("key should not be set after DISCARD")
	}
}

func TestDispatcherDiscardWithoutMulti(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	resp := dispatcher.Dispatch(protocol.Command{Name: "DISCARD"})
	if resp != "ERR not in MULTI mode" {
		t.Fatalf("expected error, got %s", resp)
	}
}

func TestDispatcherExecWithGet(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	// Set a key first
	store.Set("key", "value")

	dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	dispatcher.Dispatch(protocol.Command{Name: "GET", Args: []string{"key"}})

	resp := dispatcher.Dispatch(protocol.Command{Name: "EXEC"})
	if resp != "value" {
		t.Fatalf("expected 'value', got %s", resp)
	}
}

func TestDispatcherExecEmptyTransaction(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	dispatcher := NewDispatcher(eng)

	dispatcher.Dispatch(protocol.Command{Name: "MULTI"})
	resp := dispatcher.Dispatch(protocol.Command{Name: "EXEC"})
	if resp != "OK" {
		t.Fatalf("expected OK, got %s", resp)
	}
}

func TestProcessorClone(t *testing.T) {
	store := storage.NewStore()
	eng := engine.New(store)
	processor := NewProcessorWithEngine(eng)

	clone := processor.Clone().(*Processor)
	if clone == nil {
		t.Fatal("clone should not be nil")
	}
	if clone.dispatcher == processor.dispatcher {
		t.Fatal("clone should have a different dispatcher")
	}
}
