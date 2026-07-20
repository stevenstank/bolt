package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAOFAppendSetWritesDeterministicRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	aof := NewAOF(path)

	if err := aof.AppendSet("name", "bolt", time.Time{}); err != nil {
		t.Fatalf("append set: %v", err)
	}
	if err := aof.AppendSet("message", "hello bolt", time.Time{}); err != nil {
		t.Fatalf("append set: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read aof: %v", err)
	}
	got := string(contents)

	want := "SET\t4:name\t4:bolt\nSET\t7:message\t10:hello bolt\n"
	if got != want {
		t.Fatalf("expected AOF contents %q, got %q", want, got)
	}
}

func TestAOFLoadReplaysLatestValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	aof := NewAOF(path)

	if err := aof.AppendSet("name", "old", time.Time{}); err != nil {
		t.Fatalf("append old value: %v", err)
	}
	if err := aof.AppendSet("name", "new", time.Time{}); err != nil {
		t.Fatalf("append new value: %v", err)
	}

	data, err := aof.Load()
	if err != nil {
		t.Fatalf("load aof: %v", err)
	}

	if got := data["name"]; got.Value != "new" || got.ExpiresAt != (time.Time{}) {
		t.Fatalf("expected replayed value %q with no expiry, got %+v", "new", got)
	}
}

func TestAOFLoadMissingFileReturnsEmptyData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.aof")
	aof := NewAOF(path)

	data, err := aof.Load()
	if err != nil {
		t.Fatalf("load missing aof: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected no data, got %v", data)
	}
}

func TestAOFLoadRecoversFromPartialTrailingRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	if err := os.WriteFile(path, []byte("SET\t4:name\t4:bolt\nSET\t7:message\t"), 0o644); err != nil {
		t.Fatalf("write partial aof: %v", err)
	}

	aof := NewAOF(path)
	data, err := aof.Load()
	if err != nil {
		t.Fatalf("load aof: %v", err)
	}

	if got := data["name"]; got.Value != "bolt" {
		t.Fatalf("expected recovered value %q, got %+v", "bolt", got)
	}
	if _, ok := data["message"]; ok {
		t.Fatal("expected partial trailing record to be ignored")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read recovered aof: %v", err)
	}
	if got, want := string(contents), "SET\t4:name\t4:bolt\n"; got != want {
		t.Fatalf("expected recovered AOF contents %q, got %q", want, got)
	}
}

func TestAOFLoadReturnsErrorForCompleteInvalidRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	if err := os.WriteFile(path, []byte("not a valid record\n"), 0o644); err != nil {
		t.Fatalf("write invalid aof: %v", err)
	}

	aof := NewAOF(path)
	if _, err := aof.Load(); err == nil {
		t.Fatal("expected invalid complete record to return an error")
	}
}

func TestAOFAppendSetWritesExpirationMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	aof := NewAOF(path)
	expiresAt := time.Unix(123, 0)

	if err := aof.AppendSet("name", "bolt", expiresAt); err != nil {
		t.Fatalf("append set: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read aof: %v", err)
	}
	if got := string(contents); got == "SET\t4:name\t4:bolt\n" {
		t.Fatal("expected expiration metadata to be written")
	}
}

func TestAOFLoadPreservesExpirationMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bolt.aof")
	aof := NewAOF(path)
	expiresAt := time.Now().Add(time.Hour)
	if err := aof.AppendSet("name", "bolt", expiresAt); err != nil {
		t.Fatalf("append set: %v", err)
	}

	data, err := aof.Load()
	if err != nil {
		t.Fatalf("load aof: %v", err)
	}
	if got := data["name"]; got.Value != "bolt" || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expiration metadata to survive replay, got %+v", got)
	}
}
