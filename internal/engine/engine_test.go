package engine

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestEngineStoresAndLoadsValues(t *testing.T) {
	store := newMemoryStore()
	engine := New(store)

	if err := engine.Set("name", "saksham"); err != nil {
		t.Fatalf("set value: %v", err)
	}

	got, ok := engine.Get("name")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "saksham" {
		t.Fatalf("expected value %q, got %q", "saksham", got)
	}
}

func TestEngineNotifiesObserversAfterSuccessfulSet(t *testing.T) {
	store := newMemoryStore()
	observer := &trackingObserver{}
	engine := New(store, observer)

	if err := engine.Set("name", "saksham"); err != nil {
		t.Fatalf("set value: %v", err)
	}

	observer.mu.Lock()
	defer observer.mu.Unlock()

	if len(observer.sets) != 1 {
		t.Fatalf("expected one observer notification, got %d", len(observer.sets))
	}
	if observer.sets[0] != "name=saksham" {
		t.Fatalf("expected observer notification %q, got %q", "name=saksham", observer.sets[0])
	}
}

func TestEngineDoesNotNotifyObserversWhenSetFails(t *testing.T) {
	store := newMemoryStore()
	store.setErr = errors.New("disk full")
	observer := &trackingObserver{}
	engine := New(store, observer)

	if err := engine.Set("name", "saksham"); err == nil {
		t.Fatal("expected set to fail")
	}

	observer.mu.Lock()
	defer observer.mu.Unlock()

	if len(observer.sets) != 0 {
		t.Fatalf("expected no observer notifications, got %v", observer.sets)
	}
}

func TestEngineRejectsSetWhenReadOnly(t *testing.T) {
	store := newMemoryStore()
	engine := New(store)
	engine.SetReadOnly(true)

	if err := engine.Set("name", "saksham"); err == nil || err.Error() != "replica is read-only" {
		t.Fatalf("expected read-only error, got %v", err)
	}

	if _, ok := engine.Get("name"); ok {
		t.Fatal("expected read-only set not to persist data")
	}
}

func TestEngineAppliesReplicatedSetWhenReadOnly(t *testing.T) {
	store := newMemoryStore()
	engine := New(store)
	engine.SetReadOnly(true)

	if err := engine.ApplySet("name", "saksham"); err != nil {
		t.Fatalf("apply replicated set: %v", err)
	}

	got, ok := engine.Get("name")
	if !ok {
		t.Fatal("expected key to exist after replicated write")
	}
	if got != "saksham" {
		t.Fatalf("expected value %q, got %q", "saksham", got)
	}
}

type memoryStore struct {
	mu     sync.Mutex
	values map[string]string
	setErr error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}}
}

func (s *memoryStore) Set(key, value string) error {
	return s.set(key, value)
}

func (s *memoryStore) SetWithExpiry(key, value string, expiresAt time.Time) error {
	return s.set(key, value)
}

func (s *memoryStore) set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = value
	return nil
}

func (s *memoryStore) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	return value, ok
}

func (s *memoryStore) ApplySet(key, value string) error {
	return s.set(key, value)
}

func (s *memoryStore) ApplySetWithExpiry(key, value string, expiresAt time.Time) error {
	return s.set(key, value)
}

type trackingObserver struct {
	mu   sync.Mutex
	sets []string
}

func (o *trackingObserver) OnSet(key, value string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.sets = append(o.sets, key+"="+value)
}
