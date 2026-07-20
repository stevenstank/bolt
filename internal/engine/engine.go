package engine

import (
	"errors"
	"time"
)

type store interface {
	Set(key, value string) error
	SetWithExpiry(key, value string, expiresAt time.Time) error
	Get(key string) (string, bool)
}

// Observer receives notifications after successful writes.
type Observer interface {
	OnSet(key, value string)
}

type expiryObserver interface {
	OnSetWithExpiry(key, value string, expiresAt time.Time)
}

// Engine owns database operations and coordinates access to storage.
type Engine struct {
	store     store
	observers []Observer
	readOnly   bool
}

// New creates an Engine backed by store.
func New(store store, observers ...Observer) *Engine {
	return &Engine{store: store, observers: observers}
}

// SetReadOnly toggles whether normal writes are allowed.
func (e *Engine) SetReadOnly(readOnly bool) {
	e.readOnly = readOnly
}

// Set stores a value.
func (e *Engine) Set(key, value string) error {
	if e.readOnly {
		return errors.New("replica is read-only")
	}

	return e.applySet(key, value, time.Time{}, true)
}

// SetWithExpiry stores a value with an expiration timestamp.
func (e *Engine) SetWithExpiry(key, value string, expiresAt time.Time) error {
	if e.readOnly {
		return errors.New("replica is read-only")
	}

	return e.applySet(key, value, expiresAt, true)
}

// ApplySet stores a value received from replication.
func (e *Engine) ApplySet(key, value string) error {
	return e.applySet(key, value, time.Time{}, false)
}

// ApplySetWithExpiry stores a replicated value with expiration.
func (e *Engine) ApplySetWithExpiry(key, value string, expiresAt time.Time) error {
	return e.applySet(key, value, expiresAt, false)
}

func (e *Engine) applySet(key, value string, expiresAt time.Time, notify bool) error {
	var err error
	if expiresAt.IsZero() {
		err = e.store.Set(key, value)
	} else {
		err = e.store.SetWithExpiry(key, value, expiresAt)
	}
	if err != nil {
		return err
	}

	if notify {
		for _, observer := range e.observers {
			if observer != nil {
				if observerWithExpiry, ok := observer.(expiryObserver); ok {
					observerWithExpiry.OnSetWithExpiry(key, value, expiresAt)
					continue
				}
				observer.OnSet(key, value)
			}
		}
	}
	return nil
}

// Get returns a value from storage.
func (e *Engine) Get(key string) (string, bool) {
	return e.store.Get(key)
}

// KeyCount returns the number of keys in storage.
func (e *Engine) KeyCount() int {
	if kc, ok := e.store.(interface{ KeyCount() int }); ok {
		return kc.KeyCount()
	}
	return 0
}

// MemoryUsage returns the estimated memory usage in bytes.
func (e *Engine) MemoryUsage() int64 {
	if mu, ok := e.store.(interface{ MemoryUsage() int64 }); ok {
		return mu.MemoryUsage()
	}
	return 0
}
