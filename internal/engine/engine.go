package engine

type store interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}

// Engine owns database operations and coordinates access to storage.
type Engine struct {
	store store
}

// New creates an Engine backed by store.
func New(store store) *Engine {
	return &Engine{store: store}
}

// Set stores a value.
func (e *Engine) Set(key, value string) error {
	return e.store.Set(key, value)
}

// Get returns a value from storage.
func (e *Engine) Get(key string) (string, bool) {
	return e.store.Get(key)
}
