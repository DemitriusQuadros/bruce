package domain

import "time"

// ConfigEntry represents a runtime configuration key-value pair stored in SQLite.
type ConfigEntry struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
