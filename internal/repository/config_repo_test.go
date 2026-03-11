package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsert_InsertsNew(t *testing.T) {
	repo := NewConfigRepository(openTestDB(t))

	err := repo.Upsert("theme", "dark")
	require.NoError(t, err)

	val, err := repo.Get("theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestConfigUpsert_OverwritesExistingValue(t *testing.T) {
	repo := NewConfigRepository(openTestDB(t))

	require.NoError(t, repo.Upsert("key1", "original"))
	require.NoError(t, repo.Upsert("key1", "updated"))

	val, err := repo.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, "updated", val)
}

func TestGet_NotFound(t *testing.T) {
	repo := NewConfigRepository(openTestDB(t))

	_, err := repo.Get("missing-key")
	assert.Error(t, err)
}

func TestGetAll_ReturnsAll(t *testing.T) {
	repo := NewConfigRepository(openTestDB(t))

	require.NoError(t, repo.Upsert("a", "1"))
	require.NoError(t, repo.Upsert("b", "2"))
	require.NoError(t, repo.Upsert("c", "3"))

	entries, err := repo.GetAll()
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	// ordered by key ASC
	assert.Equal(t, "a", entries[0].Key)
	assert.Equal(t, "c", entries[2].Key)
}
