package record

import "time"

// Entry stores a value and optional expiration timestamp.
type Entry struct {
	Value     string
	ExpiresAt time.Time
}

// Expired reports whether the entry should no longer be served.
func (e Entry) Expired(now time.Time) bool {
	return !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt)
}
