package repository

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/database"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	require.NoError(t, err)
	_, err = db.Exec(database.Schema)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestFindOrCreate_CreatesOnFirstCall(t *testing.T) {
	repo := NewSessionRepository(openTestDB(t))
	s, err := repo.FindOrCreate("whatsapp", "channel-1")
	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, "whatsapp", s.ConnectorType)
	assert.Equal(t, "channel-1", s.ChannelID)
	assert.Equal(t, "", s.SystemPrompt)
	assert.True(t, s.IsActive)
}

func TestFindOrCreate_ReturnsExistingOnSecondCall(t *testing.T) {
	repo := NewSessionRepository(openTestDB(t))
	s1, err := repo.FindOrCreate("discord", "chan-abc")
	require.NoError(t, err)

	s2, err := repo.FindOrCreate("discord", "chan-abc")
	require.NoError(t, err)

	assert.Equal(t, s1.ID, s2.ID, "second call should return the same session")
}

func TestGetAll_ReturnsSessions(t *testing.T) {
	repo := NewSessionRepository(openTestDB(t))
	_, err := repo.FindOrCreate("whatsapp", "c1")
	require.NoError(t, err)
	_, err = repo.FindOrCreate("discord", "c2")
	require.NoError(t, err)

	sessions, err := repo.GetAll()
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := NewSessionRepository(openTestDB(t))
	_, err := repo.GetByID("nonexistent-id")
	assert.Error(t, err)
}

func TestUpdateSystemPrompt(t *testing.T) {
	repo := NewSessionRepository(openTestDB(t))
	s, err := repo.FindOrCreate("whatsapp", "chan-x")
	require.NoError(t, err)

	err = repo.UpdateSystemPrompt(s.ID, "You are a helpful assistant.")
	require.NoError(t, err)

	updated, err := repo.GetByID(s.ID)
	require.NoError(t, err)
	assert.Equal(t, "You are a helpful assistant.", updated.SystemPrompt)
}

func TestSetActive(t *testing.T) {
	repo := NewSessionRepository(openTestDB(t))
	s, err := repo.FindOrCreate("discord", "chan-y")
	require.NoError(t, err)
	assert.True(t, s.IsActive)

	err = repo.SetActive(s.ID, false)
	require.NoError(t, err)

	updated, err := repo.GetByID(s.ID)
	require.NoError(t, err)
	assert.False(t, updated.IsActive)
}
