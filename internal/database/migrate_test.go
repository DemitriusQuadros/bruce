package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunMigrations_FreshDatabase(t *testing.T) {
	// Test on a fresh in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify sessions table exists and has provider_override column
	exists, err := columnExists(db, "sessions", "provider_override")
	require.NoError(t, err)
	assert.True(t, exists, "provider_override column should exist after migration")

	// Verify we can insert a session with provider_override
	_, err = db.Exec(
		`INSERT INTO sessions (id, connector_type, channel_id, system_prompt, is_active, provider_override, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		"test-id", "whatsapp", "123456", "You are helpful", 1, "gemini",
	)
	require.NoError(t, err)

	// Query it back
	var id, connectorType, channelID, systemPrompt, providerOverride string
	var isActive int
	err = db.QueryRow(
		`SELECT id, connector_type, channel_id, system_prompt, is_active, provider_override
		 FROM sessions WHERE id = ?`,
		"test-id",
	).Scan(&id, &connectorType, &channelID, &systemPrompt, &isActive, &providerOverride)
	require.NoError(t, err)
	assert.Equal(t, "test-id", id)
	assert.Equal(t, "whatsapp", connectorType)
	assert.Equal(t, "123456", channelID)
	assert.Equal(t, "gemini", providerOverride)
}

func TestRunMigrations_Idempotent(t *testing.T) {
	// Test that running migrations twice doesn't error out
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations first time
	err = RunMigrations(db)
	require.NoError(t, err)

	// Run migrations second time (should be idempotent)
	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify column still exists and works
	exists, err := columnExists(db, "sessions", "provider_override")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestColumnExists_ReturnsTrue(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create a simple table
	_, err = db.Exec(`CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	exists, err := columnExists(db, "test_table", "name")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestColumnExists_ReturnsFalse(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create a simple table
	_, err = db.Exec(`CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	exists, err := columnExists(db, "test_table", "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRunMigrations_AddsProviderOverrideToExistingTable(t *testing.T) {
	// Simulate an old database without provider_override
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create sessions table without provider_override (old schema)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id               TEXT    PRIMARY KEY,
			connector_type   TEXT    NOT NULL CHECK(connector_type IN ('whatsapp', 'discord')),
			channel_id       TEXT    NOT NULL,
			system_prompt    TEXT    NOT NULL DEFAULT '',
			is_active        INTEGER NOT NULL DEFAULT 1,
			created_at       DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at       DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(connector_type, channel_id)
		)
	`)
	require.NoError(t, err)

	// Verify provider_override doesn't exist yet
	exists, err := columnExists(db, "sessions", "provider_override")
	require.NoError(t, err)
	assert.False(t, exists, "provider_override should not exist before migration")

	// Run migrations (should add provider_override)
	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify provider_override now exists
	exists, err = columnExists(db, "sessions", "provider_override")
	require.NoError(t, err)
	assert.True(t, exists, "provider_override should exist after migration")

	// Verify existing data still works and defaults to empty string
	_, err = db.Exec(
		`INSERT INTO sessions (id, connector_type, channel_id, system_prompt, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		"old-session-id", "discord", "789", "system prompt", 1,
	)
	require.NoError(t, err)

	// Query it back
	var providerOverride string
	err = db.QueryRow(
		`SELECT provider_override FROM sessions WHERE id = ?`,
		"old-session-id",
	).Scan(&providerOverride)
	require.NoError(t, err)
	assert.Equal(t, "", providerOverride, "provider_override should default to empty string for existing records")
}
