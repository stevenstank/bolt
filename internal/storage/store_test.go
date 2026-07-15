package storage

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestStoreSnapshotReturnsCopyOfCurrentData(t *testing.T) {
	store := NewStore()
	store.Set("name", "bolt")

	snapshot := store.Snapshot()
	snapshot["name"] = "mutated"

	got, ok := store.Get("name")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "bolt" {
		t.Fatalf("expected store value %q, got %q", "bolt", got)
	}
}

func TestPersistentStoreLoadsValuesFromAOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")

	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("create persistent store: %v", err)
	}
	if err := store.Set("name", "bolt"); err != nil {
		t.Fatalf("set value: %v", err)
	}

	restarted, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("restart persistent store: %v", err)
	}

	got, ok := restarted.Get("name")
	if !ok {
		t.Fatal("expected persisted key to exist after restart")
	}
	if got != "bolt" {
		t.Fatalf("expected persisted value %q, got %q", "bolt", got)
	}
}

func TestPersistentStoreReturnsErrorForCorruptAOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	if err := os.WriteFile(path, []byte("not a valid record\n"), 0o644); err != nil {
		t.Fatalf("write corrupt aof: %v", err)
	}

	if _, err := NewPersistentStore(path); err == nil {
		t.Fatal("expected corrupt AOF to return an error")
	}
}

func TestDurableStoreLoadsSnapshotThenAOF(t *testing.T) {
	dir := t.TempDir()
	aofPath := filepath.Join(dir, "bolt.aof")
	snapshotPath := filepath.Join(dir, "bolt.snapshot")

	store, err := NewDurableStore(aofPath, snapshotPath)
	if err != nil {
		t.Fatalf("create durable store: %v", err)
	}
	if err := store.Set("name", "from-aof"); err != nil {
		t.Fatalf("set value: %v", err)
	}
	if err := store.SaveSnapshot(); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	if err := store.Set("language", "Go"); err != nil {
		t.Fatalf("set value after snapshot: %v", err)
	}

	restarted, err := NewDurableStore(aofPath, snapshotPath)
	if err != nil {
		t.Fatalf("restart durable store: %v", err)
	}

	if got, ok := restarted.Get("name"); !ok || got != "from-aof" {
		t.Fatalf("expected snapshot value %q, got %q ok=%v", "from-aof", got, ok)
	}
	if got, ok := restarted.Get("language"); !ok || got != "Go" {
		t.Fatalf("expected AOF value %q, got %q ok=%v", "Go", got, ok)
	}
}

func TestStoreSaveSnapshotWritesCurrentData(t *testing.T) {
	dir := t.TempDir()
	store, err := NewDurableStore(filepath.Join(dir, "bolt.aof"), filepath.Join(dir, "bolt.snapshot"))
	if err != nil {
		t.Fatalf("create durable store: %v", err)
	}

	if err := store.Set("name", "saksham"); err != nil {
		t.Fatalf("set value: %v", err)
	}
	if err := store.SaveSnapshot(); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	restarted, err := NewDurableStore(filepath.Join(dir, "empty.aof"), filepath.Join(dir, "bolt.snapshot"))
	if err != nil {
		t.Fatalf("restart from snapshot: %v", err)
	}
	if got, ok := restarted.Get("name"); !ok || got != "saksham" {
		t.Fatalf("expected snapshot value %q, got %q ok=%v", "saksham", got, ok)
	}
}
