package engine

import "errors"

type store interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}

// Observer receives notifications after successful writes.
type Observer interface {
	OnSet(key, value string)
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

	return e.applySet(key, value, true)
}

// ApplySet stores a value received from replication.
func (e *Engine) ApplySet(key, value string) error {
	return e.applySet(key, value, false)
}

func (e *Engine) applySet(key, value string, notify bool) error {
	if err := e.store.Set(key, value); err != nil {
		return err
	}

	if notify {
		for _, observer := range e.observers {
			if observer != nil {
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
