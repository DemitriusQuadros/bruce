# Spec 32: Proactive Core — Data Models, SQLite Repository & Background Scheduler Poller [BACKEND]

## Overview

Implement the core data persistence layer and background scheduler poller for Bruce's proactive intelligence engine. This forms the foundation for both **Ambient Condition Watches** (interval-based monitoring) and **Scheduled Reports** (time-of-day crons).

## Phase

**Phase 1** (Weeks 1–3)

## Prerequisites

- SQLite connection with WAL mode established (`internal/database`)
- Domain model architecture established (`internal/domain`)
- Config loader in place (`internal/config`)

## Deliverables

### Files to Create:
- `internal/domain/proactive_task.go` — Domain struct, TaskType enum, and helpers
- `internal/repository/proactive_task_repo.go` — `ProactiveTaskRepository` interface and SQLite implementation
- `internal/repository/proactive_task_repo_test.go` — Unit tests against SQLite in-memory (`:memory:`)
- `internal/scheduler/poller.go` — 60-second ticker evaluating due tasks and queuing Asynq tasks
- `internal/scheduler/poller_test.go` — Unit tests for timezone resolution and cron calculations

### Files to Modify:
- `internal/database/schema.sql` — Add `proactive_tasks` table and composite indexes
- `internal/config/config.go` — Add `AppConfig.Timezone`, `LLMConfig.BackgroundProvider`, `LLMConfig.BackgroundModel`
- `config.example.yml` — Document `app.timezone` and background LLM configuration
- `cmd/bruce/main.go` — Initialize repository and start poller goroutine

---

## Detailed Specifications

### 1. Database Schema (`internal/database/schema.sql`)

```sql
CREATE TABLE IF NOT EXISTS proactive_tasks (
    id                 TEXT PRIMARY KEY,
    session_id         TEXT NOT NULL,
    connector_type     TEXT NOT NULL CHECK(connector_type IN ('whatsapp', 'discord', 'telegram', 'web')),
    channel_id         TEXT NOT NULL,
    target_connector   TEXT NOT NULL CHECK(target_connector IN ('whatsapp', 'discord', 'telegram', 'web')),
    target_channel_id  TEXT NOT NULL,
    title              TEXT NOT NULL,
    task_type          TEXT NOT NULL CHECK(task_type IN ('watch', 'cron')),
    schedule_expr      TEXT NOT NULL, -- Interval minutes (e.g. '30') or standard cron (e.g. '0 9 * * 1-5')
    timezone           TEXT NOT NULL DEFAULT 'UTC',
    prompt_condition   TEXT NOT NULL,
    target_tools       TEXT NOT NULL DEFAULT '[]', -- JSON array of tool names (for watches)
    is_active          INTEGER NOT NULL DEFAULT 1,
    last_run_at        DATETIME,
    next_run_at        DATETIME NOT NULL,
    last_result_hash   TEXT NOT NULL DEFAULT '',
    created_at         DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at         DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_proactive_tasks_due 
    ON proactive_tasks (is_active, next_run_at);

CREATE INDEX IF NOT EXISTS idx_proactive_tasks_session 
    ON proactive_tasks (session_id);
```

### 2. Domain Model (`internal/domain/proactive_task.go`)

```go
package domain

import "time"

type TaskType string

const (
    TaskTypeWatch TaskType = "watch"
    TaskTypeCron  TaskType = "cron"
)

type ProactiveTask struct {
    ID              string     `json:"id"`
    SessionID       string     `json:"session_id"`
    ConnectorType   string     `json:"connector_type"`
    ChannelID       string     `json:"channel_id"`
    TargetConnector string     `json:"target_connector"`
    TargetChannelID string     `json:"target_channel_id"`
    Title           string     `json:"title"`
    TaskType        TaskType   `json:"task_type"`
    ScheduleExpr    string     `json:"schedule_expr"`
    Timezone        string     `json:"timezone"`
    PromptCondition string     `json:"prompt_condition"`
    TargetTools     []string   `json:"target_tools"`
    IsActive        bool       `json:"is_active"`
    LastRunAt       *time.Time `json:"last_run_at,omitempty"`
    NextRunAt       time.Time  `json:"next_run_at"`
    LastResultHash  string     `json:"last_result_hash"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 3. Repository Interface (`internal/repository/proactive_task_repo.go`)

```go
type ProactiveTaskRepository interface {
    Create(ctx context.Context, task *domain.ProactiveTask) error
    GetByID(ctx context.Context, id string) (*domain.ProactiveTask, error)
    ListBySession(ctx context.Context, sessionID string) ([]domain.ProactiveTask, error)
    ListAll(ctx context.Context) ([]domain.ProactiveTask, error)
    GetDueTasks(ctx context.Context, now time.Time) ([]domain.ProactiveTask, error)
    UpdateNextRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time) error
    UpdateStatus(ctx context.Context, id string, isActive bool) error
    UpdateLastResultHash(ctx context.Context, id string, hash string) error
    Delete(ctx context.Context, id string) error
}
```

### 4. Background Scheduler Poller (`internal/scheduler/poller.go`)

- Evaluates due tasks every 60 seconds.
- **Timezone Resolution Algorithm**:
  1. If task has a valid non-empty `Timezone`, parse with `time.LoadLocation(task.Timezone)`.
  2. If empty or invalid, check host system timezone: `time.Local`.
  3. If `time.Local` is UTC or cannot be loaded, fallback to `"America/Sao_Paulo"`.
- **Next Run Time Calculation**:
  - For `TaskTypeWatch`: `nextRunAt = now.Add(time.Duration(intervalMinutes) * time.Minute)`. Minimum 5 minutes.
  - For `TaskTypeCron`: Uses `robfig/cron/v3` parser with location. Computes `schedule.Next(now)`.
- Updates `last_run_at` and `next_run_at` atomically in SQLite before enqueueing to prevent duplicate firings.
- Enqueues Asynq task to Redis:
  - `proactive:evaluate_watch` for `TaskTypeWatch`
  - `proactive:execute_report` for `TaskTypeCron`

---

## Acceptance Criteria

- [ ] `schema.sql` creates `proactive_tasks` table and indexes without syntax errors
- [ ] In-memory SQLite tests cover: task creation, get by ID, list by session, `GetDueTasks()` filtering on `(is_active=1 AND next_run_at <= now)`, status update, and deletion
- [ ] Poller correctly resolves host timezone and falls back to `"America/Sao_Paulo"`
- [ ] Poller correctly schedules next execution for standard 5-field cron strings (`0 9 * * 1-5`)
- [ ] Concurrency-safe: poller updates `next_run_at` immediately upon dequeueing so a slow worker does not cause double-enqueueing
