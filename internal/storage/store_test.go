package storage

import "testing"

func TestStoreSetAndGet(t *testing.T) {
	store := NewStore()

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{
			name:  "simple value",
			key:   "name",
			value: "bolt",
		},
		{
			name:  "empty value",
			key:   "empty",
			value: "",
		},
		{
			name:  "value with spaces",
			key:   "message",
			value: "hello bolt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.Set(tt.key, tt.value)

			got, ok := store.Get(tt.key)
			if !ok {
				t.Fatalf("expected key %q to exist", tt.key)
			}
			if got != tt.value {
				t.Fatalf("expected value %q, got %q", tt.value, got)
			}
		})
	}
}

func TestStoreSetReplacesExistingValue(t *testing.T) {
	store := NewStore()

	store.Set("name", "old")
	store.Set("name", "new")

	got, ok := store.Get("name")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "new" {
		t.Fatalf("expected replacement value %q, got %q", "new", got)
	}
}

func TestStoreGetMissingKey(t *testing.T) {
	store := NewStore()

	got, ok := store.Get("missing")
	if ok {
		t.Fatal("expected missing key to return ok=false")
	}
	if got != "" {
		t.Fatalf("expected missing key to return empty value, got %q", got)
	}
}
