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
