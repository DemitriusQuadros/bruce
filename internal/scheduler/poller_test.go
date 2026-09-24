package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/domain"
)

func TestResolveTimezone(t *testing.T) {
	// 1. Explicit valid timezone
	loc := ResolveTimezone("America/New_York", "America/Sao_Paulo")
	assert.Equal(t, "America/New_York", loc.String())

	// 2. Empty task timezone with config fallback
	loc2 := ResolveTimezone("", "Europe/London")
	assert.Equal(t, "Europe/London", loc2.String())

	// 3. Fallback when both empty/invalid
	loc3 := ResolveTimezone("invalid/tz", "")
	assert.Equal(t, "America/Sao_Paulo", loc3.String())
}

func TestCalculateNextRun_Watch(t *testing.T) {
	poller := NewPoller(nil, nil, &config.Config{})
	now := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)

	// Valid 30-minute interval
	task := &domain.ProactiveTask{
		TaskType:     domain.TaskTypeWatch,
		ScheduleExpr: "30",
	}
	next, err := poller.CalculateNextRun(task, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(30*time.Minute), next)

	// Sub-5 minute interval gets clamped to 5 minutes
	taskClamped := &domain.ProactiveTask{
		TaskType:     domain.TaskTypeWatch,
		ScheduleExpr: "2",
	}
	nextClamped, err := poller.CalculateNextRun(taskClamped, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(5*time.Minute), nextClamped)
}

func TestCalculateNextRun_Cron(t *testing.T) {
	poller := NewPoller(nil, nil, &config.Config{
		App: config.AppConfig{Timezone: "America/Sao_Paulo"},
	})

	// Friday 2026-09-18 08:00:00 UTC
	// Sao Paulo is UTC-3, so it is 05:00:00 local time
	now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

	// Cron: 0 9 * * * (9 AM daily)
	task := &domain.ProactiveTask{
		TaskType:     domain.TaskTypeCron,
		ScheduleExpr: "0 9 * * *",
		Timezone:     "America/Sao_Paulo",
	}
	next, err := poller.CalculateNextRun(task, now)
	require.NoError(t, err)

	// 9 AM Sao Paulo = 12:00:00 UTC
	expected := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

type mockTaskRepo struct {
	tasks []domain.ProactiveTask
}

func (m *mockTaskRepo) Create(ctx context.Context, task *domain.ProactiveTask) error { return nil }
func (m *mockTaskRepo) GetByID(ctx context.Context, id string) (*domain.ProactiveTask, error) {
	return nil, nil
}
func (m *mockTaskRepo) ListBySession(ctx context.Context, sessionID string) ([]domain.ProactiveTask, error) {
	return nil, nil
}
func (m *mockTaskRepo) ListAll(ctx context.Context) ([]domain.ProactiveTask, error) { return nil, nil }
func (m *mockTaskRepo) GetDueTasks(ctx context.Context, now time.Time) ([]domain.ProactiveTask, error) {
	return m.tasks, nil
}
func (m *mockTaskRepo) Update(ctx context.Context, task *domain.ProactiveTask) error {
	return nil
}
func (m *mockTaskRepo) UpdateNextRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time) error {
	return nil
}
func (m *mockTaskRepo) UpdateStatus(ctx context.Context, id string, isActive bool) error { return nil }
func (m *mockTaskRepo) UpdateLastResultHash(ctx context.Context, id string, hash string) error {
	return nil
}
func (m *mockTaskRepo) Delete(ctx context.Context, id string) error { return nil }

func TestPoller_EvaluateDueTasks(t *testing.T) {
	now := time.Now()
	repo := &mockTaskRepo{
		tasks: []domain.ProactiveTask{
			{
				ID:              "task-1",
				Title:           "Watch Inbox",
				TaskType:        domain.TaskTypeWatch,
				ScheduleExpr:    "15",
				TargetConnector: "discord",
				TargetChannelID: "chan-123",
				NextRunAt:       now.Add(-1 * time.Minute),
			},
		},
	}

	poller := NewPoller(repo, nil, &config.Config{})
	// Should run without panic even with nil asynqClient (mocking dry run)
	poller.EvaluateDueTasks(context.Background())
}
