package main

import "testing"

func TestParseConfigUsesDefaultAddress(t *testing.T) {
	config, err := parseConfig(nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if config.Addr != "127.0.0.1:6379" {
		t.Fatalf("expected default address %q, got %q", "127.0.0.1:6379", config.Addr)
	}
	if config.AOFPath != "bolt.aof" {
		t.Fatalf("expected default AOF path %q, got %q", "bolt.aof", config.AOFPath)
	}
	if config.SnapshotPath != "bolt.snapshot" {
		t.Fatalf("expected default snapshot path %q, got %q", "bolt.snapshot", config.SnapshotPath)
	}
}

func TestParseConfigUsesAddrFlag(t *testing.T) {
	config, err := parseConfig([]string{"-addr", "127.0.0.1:6380"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if config.Addr != "127.0.0.1:6380" {
		t.Fatalf("expected configured address %q, got %q", "127.0.0.1:6380", config.Addr)
	}
}

func TestParseConfigUsesPersistenceFlags(t *testing.T) {
	config, err := parseConfig([]string{
		"-aof", "/tmp/bolt.aof",
		"-snapshot", "/tmp/bolt.snapshot",
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if config.AOFPath != "/tmp/bolt.aof" {
		t.Fatalf("expected configured AOF path %q, got %q", "/tmp/bolt.aof", config.AOFPath)
	}
	if config.SnapshotPath != "/tmp/bolt.snapshot" {
		t.Fatalf("expected configured snapshot path %q, got %q", "/tmp/bolt.snapshot", config.SnapshotPath)
	}
}

func TestParseConfigUsesReplicaOfFlag(t *testing.T) {
	config, err := parseConfig([]string{
		"-replicaof", "127.0.0.1:6380",
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if config.ReplicaOf != "127.0.0.1:6380" {
		t.Fatalf("expected configured replica source %q, got %q", "127.0.0.1:6380", config.ReplicaOf)
	}
}
