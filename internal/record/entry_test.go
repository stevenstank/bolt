package record

import (
	"testing"
	"time"
)

func TestEntryExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		entry    Entry
		expected bool
	}{
		{
			name: "no expiration",
			entry: Entry{
				Value: "hello",
			},
			expected: false,
		},
		{
			name: "expiration in future",
			entry: Entry{
				Value:     "hello",
				ExpiresAt: now.Add(time.Minute),
			},
			expected: false,
		},
		{
			name: "expiration in past",
			entry: Entry{
				Value:     "hello",
				ExpiresAt: now.Add(-time.Minute),
			},
			expected: true,
		},
		{
			name: "expiration exactly now",
			entry: Entry{
				Value:     "hello",
				ExpiresAt: now,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.Expired(now)
			if got != tt.expected {
				t.Fatalf("Expired() = %v, want %v", got, tt.expected)
			}
		})
	}
}