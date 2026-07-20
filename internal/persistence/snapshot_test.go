package persistence

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stevenstank/bolt/internal/record"
)

func TestSaveSnapshotWritesKeysInDeterministicOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.snapshot")
	data := map[string]record.Entry{
		"name":    {Value: "bolt"},
		"message": {Value: "hello bolt"},
	}

	if err := SaveSnapshot(path, data); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	want := "SET\t7:message\t10:hello bolt\nSET\t4:name\t4:bolt\n"
	if got := string(contents); got != want {
		t.Fatalf("expected snapshot contents %q, got %q", want, got)
	}
}

func TestLoadSnapshotReturnsSavedData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.snapshot")
	want := map[string]record.Entry{
		"name":    {Value: "bolt"},
		"message": {Value: "hello bolt"},
	}

	if err := SaveSnapshot(path, want); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	got, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected snapshot data %v, got %v", want, got)
	}
}

func TestLoadMissingSnapshotReturnsEmptyData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.snapshot")

	data, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("load missing snapshot: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected no data, got %v", data)
	}
}

func TestLoadSnapshotSkipsExpiredEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.snapshot")
	expiresAt := time.Unix(1, 0)
	data := map[string]record.Entry{
		"expired": {Value: "gone", ExpiresAt: expiresAt},
	}

	if err := SaveSnapshot(path, data); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	got, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected expired entry to be skipped, got %v", got)
	}
}
