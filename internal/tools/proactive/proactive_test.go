package proactive

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
)

type mockRepo struct {
	tasks map[string]*domain.ProactiveTask
}

func newMockRepo() *mockRepo {
	return &mockRepo{tasks: make(map[string]*domain.ProactiveTask)}
}

func (m *mockRepo) Create(ctx context.Context, task *domain.ProactiveTask) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domain.ProactiveTask, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockRepo) ListBySession(ctx context.Context, sessionID string) ([]domain.ProactiveTask, error) {
	var list []domain.ProactiveTask
	for _, t := range m.tasks {
		if t.SessionID == sessionID {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *mockRepo) ListAll(ctx context.Context) ([]domain.ProactiveTask, error) {
	var list []domain.ProactiveTask
	for _, t := range m.tasks {
		list = append(list, *t)
	}
	return list, nil
}

func (m *mockRepo) GetDueTasks(ctx context.Context, now time.Time) ([]domain.ProactiveTask, error) {
	var list []domain.ProactiveTask
	for _, t := range m.tasks {
		if t.IsActive && (t.NextRunAt.Before(now) || t.NextRunAt.Equal(now)) {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *mockRepo) Update(ctx context.Context, task *domain.ProactiveTask) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockRepo) UpdateNextRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time) error {
	if t, ok := m.tasks[id]; ok {
		t.LastRunAt = &lastRunAt
		t.NextRunAt = nextRunAt
	}
	return nil
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	if t, ok := m.tasks[id]; ok {
		t.IsActive = isActive
	}
	return nil
}

func (m *mockRepo) UpdateLastResultHash(ctx context.Context, id string, hash string) error {
	if t, ok := m.tasks[id]; ok {
		t.LastResultHash = hash
	}
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	delete(m.tasks, id)
	return nil
}

func TestCreateTool_Watch(t *testing.T) {
	repo := newMockRepo()
	cfg := &config.Config{App: config.AppConfig{Timezone: "America/Sao_Paulo"}}
	tool := NewCreateTool(repo, nil, cfg)

	// Valid watch
	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"title":             "Inbox Watch",
		"type":              "watch",
		"schedule":          "15",
		"prompt_condition":  "urgent emails",
		"target_connector":  "discord",
		"target_channel_id": "chan-999",
	})
	require.NoError(t, err)
	assert.Contains(t, res.(string), "Inbox Watch")

	// Verify task in repo
	all, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "discord", all[0].TargetConnector)
	assert.Equal(t, "chan-999", all[0].TargetChannelID)

	// Invalid interval < 5
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"title":            "Too fast watch",
		"type":             "watch",
		"schedule":         "2",
		"prompt_condition": "fast check",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "less than 5 minutes")
}

func TestCreateTool_Cron(t *testing.T) {
	repo := newMockRepo()
	tool := NewCreateTool(repo, nil, &config.Config{})

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"title":            "Daily Standup",
		"type":             "cron",
		"schedule":         "0 9 * * 1-5",
		"prompt_condition": "summarize standup info",
		"timezone":         "America/Sao_Paulo",
	})
	require.NoError(t, err)
	assert.Contains(t, res.(string), "Daily Standup")

	// Invalid cron
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"title":            "Bad Cron",
		"type":             "cron",
		"schedule":         "invalid-cron",
		"prompt_condition": "bad",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestListToggleDeleteTools(t *testing.T) {
	repo := newMockRepo()
	createTool := NewCreateTool(repo, nil, &config.Config{})
	listTool := NewListTool(repo)
	toggleTool := NewToggleTool(repo)
	deleteTool := NewDeleteTool(repo)
	ctx := context.Background()

	// 1. Create a task
	_, err := createTool.Execute(ctx, map[string]interface{}{
		"title":            "Sprint Review",
		"type":             "cron",
		"schedule":         "0 14 * * 5",
		"prompt_condition": "review sprint cards",
	})
	require.NoError(t, err)

	// 2. List tasks
	listRes, err := listTool.Execute(ctx, map[string]interface{}{})
	require.NoError(t, err)
	assert.Contains(t, listRes.(string), "Sprint Review")

	// 3. Pause task
	toggleRes, err := toggleTool.Execute(ctx, map[string]interface{}{
		"task_id_or_title": "Sprint Review",
		"action":           "pause",
	})
	require.NoError(t, err)
	assert.Contains(t, toggleRes.(string), "paused")

	all, _ := repo.ListAll(ctx)
	assert.False(t, all[0].IsActive)

	// 4. Delete task
	delRes, err := deleteTool.Execute(ctx, map[string]interface{}{
		"task_id_or_title": "Sprint Review",
	})
	require.NoError(t, err)
	assert.Contains(t, delRes.(string), "deleted")

	allAfter, _ := repo.ListAll(ctx)
	assert.Empty(t, allAfter)
}

func TestCreateTool_ContextAutoDetection(t *testing.T) {
	repo := newMockRepo()
	tool := NewCreateTool(repo, nil, &config.Config{})

	// Context simulates user chatting via WhatsApp
	ctx := ai.WithSessionContext(context.Background(), "sess-wa-456", "whatsapp", "+5511988887777")

	res, err := tool.Execute(ctx, map[string]interface{}{
		"title":            "Daily Notion & Mail Summary",
		"type":             "cron",
		"schedule":         "0 9 * * 1-5",
		"prompt_condition": "Check Notion tasks and unread emails",
	})
	require.NoError(t, err)
	assert.Contains(t, res.(string), "Daily Notion & Mail Summary")

	all, err := repo.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	task := all[0]
	assert.Equal(t, "sess-wa-456", task.SessionID)
	assert.Equal(t, "whatsapp", task.ConnectorType)
	assert.Equal(t, "+5511988887777", task.ChannelID)
	assert.Equal(t, "whatsapp", task.TargetConnector)
	assert.Equal(t, "+5511988887777", task.TargetChannelID)
}
