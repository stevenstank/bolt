package command

import (
	"errors"
	"testing"

	eengine "github.com/stevenstank/bolt/internal/engine"
	"github.com/stevenstank/bolt/internal/protocol"
)

func TestDispatcherExecutesSet(t *testing.T) {
	store := newMemoryStore()
	dispatcher := NewDispatcher(store)

	response := dispatcher.Dispatch(protocol.Command{
		Name: "SET",
		Args: []string{"name", "saksham"},
	})

	if response != "OK" {
		t.Fatalf("expected response %q, got %q", "OK", response)
	}
	if got := store.values["name"]; got != "saksham" {
		t.Fatalf("expected stored value %q, got %q", "saksham", got)
	}
}

func TestDispatcherExecutesGet(t *testing.T) {
	store := newMemoryStore()
	store.values["name"] = "saksham"
	dispatcher := NewDispatcher(store)

	response := dispatcher.Dispatch(protocol.Command{
		Name: "GET",
		Args: []string{"name"},
	})

	if response != "saksham" {
		t.Fatalf("expected response %q, got %q", "saksham", response)
	}
}

func TestDispatcherReturnsNilForMissingGet(t *testing.T) {
	dispatcher := NewDispatcher(newMemoryStore())

	response := dispatcher.Dispatch(protocol.Command{
		Name: "GET",
		Args: []string{"missing"},
	})

	if response != "(nil)" {
		t.Fatalf("expected response %q, got %q", "(nil)", response)
	}
}

func TestDispatcherHandlesInvalidCommands(t *testing.T) {
	dispatcher := NewDispatcher(newMemoryStore())

	tests := []struct {
		name string
		cmd  protocol.Command
	}{
		{
			name: "set missing value",
			cmd:  protocol.Command{Name: "SET", Args: []string{"name"}},
		},
		{
			name: "get missing key",
			cmd:  protocol.Command{Name: "GET"},
		},
		{
			name: "unknown command",
			cmd:  protocol.Command{Name: "DEL", Args: []string{"name"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := dispatcher.Dispatch(tt.cmd)
			if response == "" || response[:3] != "ERR" {
				t.Fatalf("expected ERR response, got %q", response)
			}
		})
	}
}

func TestDispatcherReturnsErrorWhenSetCannotPersist(t *testing.T) {
	store := newMemoryStore()
	store.setErr = errors.New("disk full")
	dispatcher := NewDispatcher(store)

	response := dispatcher.Dispatch(protocol.Command{
		Name: "SET",
		Args: []string{"name", "saksham"},
	})

	if response != "ERR disk full" {
		t.Fatalf("expected response %q, got %q", "ERR disk full", response)
	}
}

func TestDispatcherRejectsSetWhenEngineIsReadOnly(t *testing.T) {
	store := newMemoryStore()
	eng := eengine.New(store)
	eng.SetReadOnly(true)
	dispatcher := NewDispatcher(eng)

	response := dispatcher.Dispatch(protocol.Command{
		Name: "SET",
		Args: []string{"name", "saksham"},
	})

	if response != "ERR replica is read-only" {
		t.Fatalf("expected response %q, got %q", "ERR replica is read-only", response)
	}
}

func TestDispatcherStillAllowsReadsWhenEngineIsReadOnly(t *testing.T) {
	store := newMemoryStore()
	eng := eengine.New(store)
	eng.SetReadOnly(true)
	if err := eng.ApplySet("name", "saksham"); err != nil {
		t.Fatalf("seed replicated value: %v", err)
	}
	dispatcher := NewDispatcher(eng)

	response := dispatcher.Dispatch(protocol.Command{
		Name: "GET",
		Args: []string{"name"},
	})

	if response != "saksham" {
		t.Fatalf("expected response %q, got %q", "saksham", response)
	}
}

type memoryStore struct {
	values map[string]string
	setErr error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}}
}

func (s *memoryStore) Set(key, value string) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = value
	return nil
}

func (s *memoryStore) Get(key string) (string, bool) {
	value, ok := s.values[key]
	return value, ok
}
