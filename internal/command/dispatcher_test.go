package command

import (
	"errors"
	"sync"
	"testing"
	"time"

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

func TestDispatcherExecutesSetWithExpiry(t *testing.T) {
	store := newMemoryStore()
	dispatcher := NewDispatcher(store)

	response := dispatcher.Dispatch(protocol.Command{
		Name: "SET",
		Args: []string{"name", "saksham", "EX", "60"},
	})

	if response != "OK" {
		t.Fatalf("expected response %q, got %q", "OK", response)
	}
	if got := store.values["name"]; got != "saksham" {
		t.Fatalf("expected stored value %q, got %q", "saksham", got)
	}
	if _, ok := store.expires["name"]; !ok {
		t.Fatal("expected expiry to be set")
	}
}

func TestDispatcherRejectsInvalidExpiry(t *testing.T) {
	dispatcher := NewDispatcher(newMemoryStore())

	response := dispatcher.Dispatch(protocol.Command{
		Name: "SET",
		Args: []string{"name", "saksham", "EX", "invalid"},
	})

	if response != "ERR invalid EX seconds \"invalid\"" {
		t.Fatalf("expected response %q, got %q", "ERR invalid EX seconds \"invalid\"", response)
	}
}


type memoryStore struct {
	mu      sync.Mutex
	values  map[string]string
	expires map[string]time.Time
	setErr  error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}, expires: map[string]time.Time{}}
}

func (s *memoryStore) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = value
	return nil
}

func (s *memoryStore) SetWithExpiry(key, value string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = value
	s.expires[key] = expiresAt
	return nil
}

func (s *memoryStore) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	return value, ok
}

func (s *memoryStore) ApplySet(key, value string) error {
	return s.Set(key, value)
}

func (s *memoryStore) ApplySetWithExpiry(key, value string, expiresAt time.Time) error {
	return s.SetWithExpiry(key, value, expiresAt)
}

func (s *memoryStore) KeyCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.values)
}

func (s *memoryStore) MemoryUsage() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int64
	for _, value := range s.values {
		total += int64(len(value))
	}
	return total
}
