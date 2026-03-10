package repository_test

import (
	"bruce/app/entities"
	repository "bruce/app/repository/dummy"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) (*gorm.DB, repository.DummyRepository) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	err = db.AutoMigrate(&entities.Dummy{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	repo := repository.NewDummyRepository(db)
	return db, repo
}

func TestDummyRepository_Create(t *testing.T) {
	_, repo := setupDB(t)
	dummy := entities.Dummy{
		Text: "Hello, World!",
	}
	err := repo.Create(dummy)
	require.NoError(t, err)

	result, err := repo.GetDummyByID(1)
	require.NoError(t, err)
	require.Equal(t, "Hello, World!", result.Text)
}

func TestDummyRepository_GetDummyByID(t *testing.T) {
	_, repo := setupDB(t)
	dummy := entities.Dummy{
		Text: "Sample Text",
	}
	err := repo.Create(dummy)
	require.NoError(t, err)

	result, err := repo.GetDummyByID(1)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.ID)
	require.Equal(t, "Sample Text", result.Text)
}

func TestDummyRepository_UpdateDummy(t *testing.T) {
	_, repo := setupDB(t)
	dummy := entities.Dummy{
		Text: "Initial Text",
	}
	err := repo.Create(dummy)
	require.NoError(t, err)

	result, err := repo.GetDummyByID(1)
	require.NoError(t, err)

	result.Text = "Updated Text"
	err = repo.UpdateDummy(result)
	require.NoError(t, err)

	updatedResult, err := repo.GetDummyByID(1)
	require.NoError(t, err)
	require.Equal(t, "Updated Text", updatedResult.Text)
}
