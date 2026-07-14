package engine

import "testing"

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

type memoryStore struct {
	values map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}}
}

func (s *memoryStore) Set(key, value string) error {
	s.values[key] = value
	return nil
}

func (s *memoryStore) Get(key string) (string, bool) {
	value, ok := s.values[key]
	return value, ok
}
