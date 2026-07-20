package storage

import (
	"context"
	"sync"
	"time"

	"github.com/stevenstank/bolt/internal/persistence"
	"github.com/stevenstank/bolt/internal/record"
)

type setAppender interface {
	AppendSet(key, value string, expiresAt time.Time) error
}

// Store is Bolt's in-memory key-value storage engine.
type Store struct {
	mu           sync.RWMutex
	data         map[string]record.Entry
	persister    setAppender
	snapshotPath string
	cleanupCtx   context.Context
	cancelCleanup context.CancelFunc
}

// NewStore creates an empty, thread-safe in-memory store.
func NewStore() *Store {
	s := &Store{
		data: make(map[string]record.Entry),
	}
	s.startCleanup()
	return s
}

// NewPersistentStore creates a store backed by an append-only file.
func NewPersistentStore(path string) (*Store, error) {
	aof := persistence.NewAOF(path)
	data, err := aof.Load()
	if err != nil {
		return nil, err
	}

	s := &Store{data: data, persister: aof}
	s.startCleanup()
	return s, nil
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

	s := &Store{
		data:         data,
		persister:    aof,
		snapshotPath: snapshotPath,
	}
	s.startCleanup()
	return s, nil
}

// Set stores value at key, replacing any existing value.
func (s *Store) Set(key, value string) error {
	return s.set(key, value, time.Time{})
}

// SetWithExpiry stores a value with an expiration timestamp.
func (s *Store) SetWithExpiry(key, value string, expiresAt time.Time) error {
	return s.set(key, value, expiresAt)
}

func (s *Store) set(key, value string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.persister != nil {
		if err := s.persister.AppendSet(key, value, expiresAt); err != nil {
			return err
		}
	}

	s.data[key] = record.Entry{Value: value, ExpiresAt: expiresAt}
	return nil
}

// SaveSnapshot writes the current store contents to the configured snapshot path.
func (s *Store) SaveSnapshot() error {
	if s.snapshotPath == "" {
		return nil
	}

	s.mu.RLock()
	data := make(map[string]record.Entry, len(s.data))
	for key, value := range s.data {
		data[key] = value
	}
	s.mu.RUnlock()

	return persistence.SaveSnapshot(s.snapshotPath, data)
}

// Get returns the value stored at key.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()

	entry, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if entry.Expired(time.Now()) {
		s.purgeExpiredKey(key)
		return "", false
	}
	return entry.Value, true
}

// Snapshot returns a copy of the current key-value data.
func (s *Store) Snapshot() map[string]record.Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	snapshot := make(map[string]record.Entry, len(s.data))
	for key, value := range s.data {
		if value.Expired(now) {
			continue
		}
		snapshot[key] = value
	}
	return snapshot
}

// PurgeExpired removes expired keys from memory.
func (s *Store) PurgeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, value := range s.data {
		if value.Expired(now) {
			delete(s.data, key)
		}
	}
}

func (s *Store) purgeExpiredKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[key]
	if !ok || !entry.Expired(time.Now()) {
		return
	}
	delete(s.data, key)
}

// startCleanup begins a background goroutine to periodically purge expired keys.
func (s *Store) startCleanup() {
	s.cleanupCtx, s.cancelCleanup = context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.cleanupCtx.Done():
				return
			case <-ticker.C:
				s.PurgeExpired()
			}
		}
	}()
}

// Close stops the background cleanup goroutine.
func (s *Store) Close() {
	if s.cancelCleanup != nil {
		s.cancelCleanup()
	}
}

// KeyCount returns the number of keys in the store.
func (s *Store) KeyCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// MemoryUsage returns an estimate of memory usage in bytes.
func (s *Store) MemoryUsage() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total int64
	for _, entry := range s.data {
		total += int64(len(entry.Value))
	}
	return total
}
