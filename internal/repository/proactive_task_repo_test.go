package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/domain"
)

func TestProactiveTaskRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	sessionRepo := NewSessionRepository(db)
	repo := NewProactiveTaskRepository(db)
	ctx := context.Background()

	// Create a session first to satisfy foreign key.
	session, err := sessionRepo.FindOrCreate("discord", "chan-1")
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)
	task := &domain.ProactiveTask{
		SessionID:       session.ID,
		ConnectorType:   "discord",
		ChannelID:       "chan-1",
		TargetConnector: "whatsapp",
		TargetChannelID: "5511999999999",
		Title:           "Daily Morning Briefing",
		TaskType:        domain.TaskTypeCron,
		ScheduleExpr:    "0 9 * * 1-5",
		Timezone:        "America/Sao_Paulo",
		PromptCondition: "Summarize today's meetings and PRs",
		TargetTools:     []string{"calendar_read", "github_list_prs"},
		IsActive:        true,
		NextRunAt:       now.Add(1 * time.Hour),
	}

	// 1. Create
	err = repo.Create(ctx, task)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, task.ID, fetched.ID)
	assert.Equal(t, "Daily Morning Briefing", fetched.Title)
	assert.Equal(t, domain.TaskTypeCron, fetched.TaskType)
	assert.Equal(t, "whatsapp", fetched.TargetConnector)
	assert.Equal(t, "5511999999999", fetched.TargetChannelID)
	assert.Equal(t, []string{"calendar_read", "github_list_prs"}, fetched.TargetTools)
	assert.True(t, fetched.IsActive)

	// 3. ListBySession
	list, err := repo.ListBySession(ctx, session.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 4. UpdateNextRun
	lastRun := now
	nextRun := now.Add(24 * time.Hour)
	err = repo.UpdateNextRun(ctx, task.ID, lastRun, nextRun)
	require.NoError(t, err)

	fetchedAfterRun, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, fetchedAfterRun.LastRunAt)
	assert.WithinDuration(t, lastRun, *fetchedAfterRun.LastRunAt, 2*time.Second)
	assert.WithinDuration(t, nextRun, fetchedAfterRun.NextRunAt, 2*time.Second)

	// 5. UpdateStatus (toggle pause)
	err = repo.UpdateStatus(ctx, task.ID, false)
	require.NoError(t, err)

	fetchedPaused, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.False(t, fetchedPaused.IsActive)

	// 6. UpdateLastResultHash
	err = repo.UpdateLastResultHash(ctx, task.ID, "abc123hash")
	require.NoError(t, err)

	fetchedHashed, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "abc123hash", fetchedHashed.LastResultHash)

	// 6b. Update full task
	fetchedHashed.Title = "Updated Briefing Title"
	fetchedHashed.ScheduleExpr = "0 10 * * 1-5"
	err = repo.Update(ctx, fetchedHashed)
	require.NoError(t, err)

	fetchedUpdated, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Briefing Title", fetchedUpdated.Title)
	assert.Equal(t, "0 10 * * 1-5", fetchedUpdated.ScheduleExpr)

	// 7. Delete
	err = repo.Delete(ctx, task.ID)
	require.NoError(t, err)

	fetchedDeleted, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Nil(t, fetchedDeleted)
}

func TestProactiveTaskRepository_GetDueTasks(t *testing.T) {
	db := openTestDB(t)
	sessionRepo := NewSessionRepository(db)
	repo := NewProactiveTaskRepository(db)
	ctx := context.Background()

	session, err := sessionRepo.FindOrCreate("telegram", "chat-99")
	require.NoError(t, err)

	now := time.Now()

	// Task 1: Due now and active -> should be returned
	task1 := &domain.ProactiveTask{
		SessionID:       session.ID,
		ConnectorType:   "telegram",
		ChannelID:       "chat-99",
		TargetConnector: "telegram",
		TargetChannelID: "chat-99",
		Title:           "Due Task",
		TaskType:        domain.TaskTypeWatch,
		ScheduleExpr:    "15",
		PromptCondition: "Check unread emails",
		IsActive:        true,
		NextRunAt:       now.Add(-5 * time.Minute),
	}
	require.NoError(t, repo.Create(ctx, task1))

	// Task 2: Due in future -> should NOT be returned
	task2 := &domain.ProactiveTask{
		SessionID:       session.ID,
		ConnectorType:   "telegram",
		ChannelID:       "chat-99",
		TargetConnector: "telegram",
		TargetChannelID: "chat-99",
		Title:           "Future Task",
		TaskType:        domain.TaskTypeWatch,
		ScheduleExpr:    "15",
		PromptCondition: "Check unread emails",
		IsActive:        true,
		NextRunAt:       now.Add(30 * time.Minute),
	}
	require.NoError(t, repo.Create(ctx, task2))

	// Task 3: Due now but PAUSED -> should NOT be returned
	task3 := &domain.ProactiveTask{
		SessionID:       session.ID,
		ConnectorType:   "telegram",
		ChannelID:       "chat-99",
		TargetConnector: "telegram",
		TargetChannelID: "chat-99",
		Title:           "Paused Due Task",
		TaskType:        domain.TaskTypeCron,
		ScheduleExpr:    "0 9 * * *",
		PromptCondition: "Daily report",
		IsActive:        false,
		NextRunAt:       now.Add(-10 * time.Minute),
	}
	require.NoError(t, repo.Create(ctx, task3))

	dueTasks, err := repo.GetDueTasks(ctx, now)
	require.NoError(t, err)
	assert.Len(t, dueTasks, 1)
	assert.Equal(t, task1.ID, dueTasks[0].ID)
}

func TestProactiveTaskRepository_CascadeDeleteOnSession(t *testing.T) {
	db := openTestDB(t)
	sessionRepo := NewSessionRepository(db)
	repo := NewProactiveTaskRepository(db)
	ctx := context.Background()

	session, err := sessionRepo.FindOrCreate("web", "web-sess-1")
	require.NoError(t, err)

	task := &domain.ProactiveTask{
		SessionID:       session.ID,
		ConnectorType:   "web",
		ChannelID:       "web-sess-1",
		TargetConnector: "web",
		TargetChannelID: "web-sess-1",
		Title:           "Web Session Task",
		TaskType:        domain.TaskTypeCron,
		ScheduleExpr:    "0 12 * * *",
		PromptCondition: "Midday briefing",
		IsActive:        true,
		NextRunAt:       time.Now(),
	}
	require.NoError(t, repo.Create(ctx, task))

	// Delete parent session
	err = sessionRepo.Delete(session.ID)
	require.NoError(t, err)

	// Proactive task should be automatically deleted via CASCADE
	fetched, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched)
}
