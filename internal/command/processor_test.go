package command

import (
	"testing"
)

func TestProcessorProcessesPlainTextCommands(t *testing.T) {
	store := newMemoryStore()
	processor := NewProcessor(NewDispatcher(store))

	if response := processor.Process("SET name saksham"); response != "OK" {
		t.Fatalf("expected SET response %q, got %q", "OK", response)
	}
	if response := processor.Process("GET name"); response != "saksham" {
		t.Fatalf("expected GET response %q, got %q", "saksham", response)
	}
}

func TestProcessorPreservesSetValuesWithSpaces(t *testing.T) {
	store := newMemoryStore()
	processor := NewProcessor(NewDispatcher(store))

	if response := processor.Process("SET quote hello world from bolt"); response != "OK" {
		t.Fatalf("expected SET response %q, got %q", "OK", response)
	}
	if response := processor.Process("GET quote"); response != "hello world from bolt" {
		t.Fatalf("expected GET response %q, got %q", "hello world from bolt", response)
	}
}

func TestProcessorReturnsErrorForMalformedInput(t *testing.T) {
	processor := NewProcessor(NewDispatcher(newMemoryStore()))

	response := processor.Process("   ")
	if response == "" || response[:3] != "ERR" {
		t.Fatalf("expected ERR response, got %q", response)
	}
}

