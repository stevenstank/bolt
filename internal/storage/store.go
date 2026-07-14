package storage

import (
	"sync"

	"github.com/stevenstank/bolt/internal/persistence"
)

type setAppender interface {
	AppendSet(key, value string) error
}

// Store is Bolt's in-memory key-value storage engine.
type Store struct {
	mu           sync.RWMutex
	data         map[string]string
	persister    setAppender
	snapshotPath string
}

// NewStore creates an empty, thread-safe in-memory store.
func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

// NewPersistentStore creates a store backed by an append-only file.
func NewPersistentStore(path string) (*Store, error) {
	aof := persistence.NewAOF(path)
	data, err := aof.Load()
	if err != nil {
		return nil, err
	}

	return &Store{
		data:      data,
		persister: aof,
	}, nil
}

// NewDurableStore creates a store backed by an AOF and snapshot file.
func NewDurableStore(aofPath, snapshotPath string) (*Store, error) {
	data, err := persistence.LoadSnapshot(snapshotPath)
	if err != nil {
		return nil, err
	}

	aof := persistence.NewAOF(aofPath)
	aofData, err := aof.Load()
	if err != nil {
		return nil, err
	}
	for key, value := range aofData {
		data[key] = value
	}

	return &Store{
		data:         data,
		persister:    aof,
		snapshotPath: snapshotPath,
	}, nil
}

// Set stores value at key, replacing any existing value.
func (s *Store) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.persister != nil {
		if err := s.persister.AppendSet(key, value); err != nil {
			return err
		}
	}

	s.data[key] = value
	return nil
}

// SaveSnapshot writes the current store contents to the configured snapshot path.
func (s *Store) SaveSnapshot() error {
	if s.snapshotPath == "" {
		return nil
	}

	s.mu.RLock()
	data := make(map[string]string, len(s.data))
	for key, value := range s.data {
		data[key] = value
	}
	s.mu.RUnlock()

	return persistence.SaveSnapshot(s.snapshotPath, data)
}

// Get returns the value stored at key.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.data[key]
	return value, ok
}
