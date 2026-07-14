package protocol

import "testing"

func TestParseCommandTrimsWhitespaceAndUppercasesName(t *testing.T) {
	cmd, err := Parse("  set name saksham  ")
	if err != nil {
		t.Fatalf("parse command: %v", err)
	}

	if cmd.Name != "SET" {
		t.Fatalf("expected command name %q, got %q", "SET", cmd.Name)
	}
	if got := cmd.Args; len(got) != 2 || got[0] != "name" || got[1] != "saksham" {
		t.Fatalf("expected args [name saksham], got %v", got)
	}
}

func TestParseSetTreatsEverythingAfterKeyAsValue(t *testing.T) {
	cmd, err := Parse("SET quote hello world from bolt")
	if err != nil {
		t.Fatalf("parse command: %v", err)
	}

	if cmd.Name != "SET" {
		t.Fatalf("expected command name %q, got %q", "SET", cmd.Name)
	}
	if got := cmd.Args; len(got) != 2 || got[0] != "quote" || got[1] != "hello world from bolt" {
		t.Fatalf("expected args [quote hello world from bolt], got %v", got)
	}
}

func TestParseEmptyCommandReturnsError(t *testing.T) {
	if _, err := Parse("   "); err == nil {
		t.Fatal("expected empty command to return an error")
	}
}

func TestParseDoesNotPanicOnMalformedInput(t *testing.T) {
	inputs := []string{
		"",
		"\t",
		"GET",
		"UNKNOWN key",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("parse panicked: %v", recovered)
				}
			}()

			_, _ = Parse(input)
		})
	}
}
