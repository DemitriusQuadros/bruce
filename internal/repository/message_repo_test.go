package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/domain"
)

func makeSession(t *testing.T, repo SessionRepository) *domain.Session {
	t.Helper()
	s, err := repo.FindOrCreate("whatsapp", uuid.New().String())
	require.NoError(t, err)
	return s
}

func TestInsert_Success(t *testing.T) {
	db := openTestDB(t)
	msgRepo := NewMessageRepository(db)
	sessRepo := NewSessionRepository(db)
	s := makeSession(t, sessRepo)

	msg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: s.ID,
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}
	err := msgRepo.Insert(msg)
	require.NoError(t, err)
}

func TestGetContextWindow_ReturnsChronologicalOrder(t *testing.T) {
	db := openTestDB(t)
	msgRepo := NewMessageRepository(db)
	sessRepo := NewSessionRepository(db)
	s := makeSession(t, sessRepo)

	base := time.Now().UTC().Truncate(time.Second)
	for i, content := range []string{"first", "second", "third"} {
		err := msgRepo.Insert(&domain.Message{
			ID:        uuid.New().String(),
			SessionID: s.ID,
			Role:      "user",
			Content:   content,
			Timestamp: base.Add(time.Duration(i) * time.Second),
		})
		require.NoError(t, err)
	}

	msgs, err := msgRepo.GetContextWindow(s.ID, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	assert.Equal(t, "first", msgs[0].Content)
	assert.Equal(t, "second", msgs[1].Content)
	assert.Equal(t, "third", msgs[2].Content)
}

func TestGetContextWindow_RespectsLimit(t *testing.T) {
	db := openTestDB(t)
	msgRepo := NewMessageRepository(db)
	sessRepo := NewSessionRepository(db)
	s := makeSession(t, sessRepo)

	base := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		require.NoError(t, msgRepo.Insert(&domain.Message{
			ID:        uuid.New().String(),
			SessionID: s.ID,
			Role:      "user",
			Content:   "msg",
			Timestamp: base.Add(time.Duration(i) * time.Second),
		}))
	}

	msgs, err := msgRepo.GetContextWindow(s.ID, 3)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestGetRecent_ReturnsNewestFirst(t *testing.T) {
	db := openTestDB(t)
	msgRepo := NewMessageRepository(db)
	sessRepo := NewSessionRepository(db)
	s := makeSession(t, sessRepo)

	base := time.Now().UTC().Truncate(time.Second)
	for i, content := range []string{"oldest", "middle", "newest"} {
		require.NoError(t, msgRepo.Insert(&domain.Message{
			ID:        uuid.New().String(),
			SessionID: s.ID,
			Role:      "user",
			Content:   content,
			Timestamp: base.Add(time.Duration(i) * time.Second),
		}))
	}

	msgs, err := msgRepo.GetRecent(s.ID, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	assert.Equal(t, "newest", msgs[0].Content)
	assert.Equal(t, "oldest", msgs[2].Content)
}
