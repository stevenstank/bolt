package storage

import "sync"

// Store is Bolt's in-memory key-value storage engine.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewStore creates an empty, thread-safe in-memory store.
func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

// Set stores value at key, replacing any existing value.
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
}

// Get returns the value stored at key.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.data[key]
	return value, ok
}
