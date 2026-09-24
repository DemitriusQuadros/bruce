package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/domain"
)

type mockProactiveRepo struct {
	tasks map[string]*domain.ProactiveTask
}

func newMockProactiveRepo() *mockProactiveRepo {
	return &mockProactiveRepo{tasks: make(map[string]*domain.ProactiveTask)}
}

func (m *mockProactiveRepo) Create(ctx context.Context, task *domain.ProactiveTask) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockProactiveRepo) GetByID(ctx context.Context, id string) (*domain.ProactiveTask, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockProactiveRepo) ListBySession(ctx context.Context, sessionID string) ([]domain.ProactiveTask, error) {
	var list []domain.ProactiveTask
	for _, t := range m.tasks {
		if t.SessionID == sessionID {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *mockProactiveRepo) ListAll(ctx context.Context) ([]domain.ProactiveTask, error) {
	var list []domain.ProactiveTask
	for _, t := range m.tasks {
		list = append(list, *t)
	}
	return list, nil
}

func (m *mockProactiveRepo) GetDueTasks(ctx context.Context, now time.Time) ([]domain.ProactiveTask, error) {
	return nil, nil
}

func (m *mockProactiveRepo) Update(ctx context.Context, task *domain.ProactiveTask) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockProactiveRepo) UpdateNextRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time) error {
	return nil
}

func (m *mockProactiveRepo) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	if t, ok := m.tasks[id]; ok {
		t.IsActive = isActive
	}
	return nil
}

func (m *mockProactiveRepo) UpdateLastResultHash(ctx context.Context, id string, hash string) error {
	return nil
}

func (m *mockProactiveRepo) Delete(ctx context.Context, id string) error {
	delete(m.tasks, id)
	return nil
}

func TestProactiveTasksHandler_CreateAndList(t *testing.T) {
	repo := newMockProactiveRepo()
	cfg := &config.Config{App: config.AppConfig{Timezone: "America/Sao_Paulo"}}
	handler := ProactiveTasksHandler(cfg)

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/proactive-tasks", handler).Methods("GET", "POST")
	r.HandleFunc("/api/v1/proactive-tasks/{id}", handler).Methods("GET", "PATCH", "DELETE")

	// 1. Create task
	reqBody := []byte(`{
		"session_id": "sess-1",
		"title": "Morning Briefing",
		"task_type": "cron",
		"schedule_expr": "0 9 * * 1-5",
		"prompt_condition": "Summarize my day",
		"timezone": "America/Sao_Paulo"
	}`)

	req := httptest.NewRequest("POST", "/api/v1/proactive-tasks", bytes.NewReader(reqBody))
	req = req.WithContext(context.WithValue(req.Context(), "proactiveTaskRepo", repo))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var created domain.ProactiveTask
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Morning Briefing", created.Title)

	// 2. List tasks
	reqList := httptest.NewRequest("GET", "/api/v1/proactive-tasks", nil)
	reqList = reqList.WithContext(context.WithValue(reqList.Context(), "proactiveTaskRepo", repo))
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)

	assert.Equal(t, http.StatusOK, wList.Code)
	var list []domain.ProactiveTask
	err = json.Unmarshal(wList.Body.Bytes(), &list)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Patch task (pause)
	patchBody := []byte(`{"is_active": false}`)
	reqPatch := httptest.NewRequest("PATCH", "/api/v1/proactive-tasks/"+created.ID, bytes.NewReader(patchBody))
	reqPatch = reqPatch.WithContext(context.WithValue(reqPatch.Context(), "proactiveTaskRepo", repo))
	wPatch := httptest.NewRecorder()
	r.ServeHTTP(wPatch, reqPatch)

	assert.Equal(t, http.StatusOK, wPatch.Code)
	var patched domain.ProactiveTask
	err = json.Unmarshal(wPatch.Body.Bytes(), &patched)
	require.NoError(t, err)
	assert.False(t, patched.IsActive)

	// 4. Delete task
	reqDel := httptest.NewRequest("DELETE", "/api/v1/proactive-tasks/"+created.ID, nil)
	reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), "proactiveTaskRepo", repo))
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	assert.Equal(t, http.StatusOK, wDel.Code)
	assert.Empty(t, repo.tasks)
}
