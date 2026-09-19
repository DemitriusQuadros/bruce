package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bruce/internal/domain"
)

// ProactiveTaskRepository defines the data access contract for proactive tasks.
type ProactiveTaskRepository interface {
	Create(ctx context.Context, task *domain.ProactiveTask) error
	GetByID(ctx context.Context, id string) (*domain.ProactiveTask, error)
	ListBySession(ctx context.Context, sessionID string) ([]domain.ProactiveTask, error)
	ListAll(ctx context.Context) ([]domain.ProactiveTask, error)
	GetDueTasks(ctx context.Context, now time.Time) ([]domain.ProactiveTask, error)
	Update(ctx context.Context, task *domain.ProactiveTask) error
	UpdateNextRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time) error
	UpdateStatus(ctx context.Context, id string, isActive bool) error
	UpdateLastResultHash(ctx context.Context, id string, hash string) error
	Delete(ctx context.Context, id string) error
}

// SQLiteProactiveTaskRepository implements ProactiveTaskRepository for SQLite.
type SQLiteProactiveTaskRepository struct {
	db *sql.DB
}

// NewProactiveTaskRepository returns a new SQLiteProactiveTaskRepository.
func NewProactiveTaskRepository(db *sql.DB) ProactiveTaskRepository {
	return &SQLiteProactiveTaskRepository{db: db}
}

// Create inserts a new proactive task into SQLite.
func (r *SQLiteProactiveTaskRepository) Create(ctx context.Context, task *domain.ProactiveTask) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = time.Now()
	}
	if task.TargetConnector == "" {
		task.TargetConnector = task.ConnectorType
	}
	if task.TargetChannelID == "" {
		task.TargetChannelID = task.ChannelID
	}
	if task.Timezone == "" {
		task.Timezone = "UTC"
	}

	var lastRunStr *string
	if task.LastRunAt != nil {
		s := task.LastRunAt.UTC().Format(time.RFC3339)
		lastRunStr = &s
	}

	query := `INSERT INTO proactive_tasks (
		id, session_id, connector_type, channel_id, target_connector, target_channel_id,
		title, task_type, schedule_expr, timezone, prompt_condition, target_tools,
		is_active, last_run_at, next_run_at, last_result_hash, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	isActiveInt := 0
	if task.IsActive {
		isActiveInt = 1
	}

	_, err := r.db.ExecContext(
		ctx, query,
		task.ID, task.SessionID, task.ConnectorType, task.ChannelID,
		task.TargetConnector, task.TargetChannelID,
		task.Title, string(task.TaskType), task.ScheduleExpr, task.Timezone,
		task.PromptCondition, task.TargetToolsJSON(),
		isActiveInt, lastRunStr, task.NextRunAt.UTC().Format(time.RFC3339),
		task.LastResultHash, task.CreatedAt.UTC().Format(time.RFC3339), task.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create proactive task: %w", err)
	}
	return nil
}

// GetByID finds a proactive task by its ID.
func (r *SQLiteProactiveTaskRepository) GetByID(ctx context.Context, id string) (*domain.ProactiveTask, error) {
	row := r.db.QueryRowContext(ctx, selectProactiveTaskColumns+" WHERE id = ?", id)
	task, err := scanProactiveTaskRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get proactive task %s: %w", id, err)
	}
	return task, nil
}

// ListBySession returns all proactive tasks for a given session.
func (r *SQLiteProactiveTaskRepository) ListBySession(ctx context.Context, sessionID string) ([]domain.ProactiveTask, error) {
	rows, err := r.db.QueryContext(ctx, selectProactiveTaskColumns+" WHERE session_id = ? ORDER BY created_at DESC", sessionID)
	if err != nil {
		return nil, fmt.Errorf("list proactive tasks by session %s: %w", sessionID, err)
	}
	defer rows.Close()

	return scanProactiveTaskRows(rows)
}

// ListAll returns all proactive tasks across all sessions.
func (r *SQLiteProactiveTaskRepository) ListAll(ctx context.Context) ([]domain.ProactiveTask, error) {
	rows, err := r.db.QueryContext(ctx, selectProactiveTaskColumns+" ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list all proactive tasks: %w", err)
	}
	defer rows.Close()

	return scanProactiveTaskRows(rows)
}

// GetDueTasks returns all active tasks where next_run_at <= now.
func (r *SQLiteProactiveTaskRepository) GetDueTasks(ctx context.Context, now time.Time) ([]domain.ProactiveTask, error) {
	query := selectProactiveTaskColumns + " WHERE is_active = 1 AND next_run_at <= ? ORDER BY next_run_at ASC"
	rows, err := r.db.QueryContext(ctx, query, now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("get due proactive tasks: %w", err)
	}
	defer rows.Close()

	return scanProactiveTaskRows(rows)
}

// Update updates the mutable fields of a proactive task.
func (r *SQLiteProactiveTaskRepository) Update(ctx context.Context, task *domain.ProactiveTask) error {
	query := `UPDATE proactive_tasks SET
		title = ?, schedule_expr = ?, timezone = ?, prompt_condition = ?,
		target_connector = ?, target_channel_id = ?, target_tools = ?,
		is_active = ?, next_run_at = ?, updated_at = datetime('now')
		WHERE id = ?`
	isActiveInt := 0
	if task.IsActive {
		isActiveInt = 1
	}
	_, err := r.db.ExecContext(
		ctx, query,
		task.Title, task.ScheduleExpr, task.Timezone, task.PromptCondition,
		task.TargetConnector, task.TargetChannelID, task.TargetToolsJSON(),
		isActiveInt, task.NextRunAt.UTC().Format(time.RFC3339),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("update proactive task %s: %w", task.ID, err)
	}
	return nil
}

// UpdateNextRun updates last_run_at and next_run_at for a task.
func (r *SQLiteProactiveTaskRepository) UpdateNextRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time) error {
	query := `UPDATE proactive_tasks SET last_run_at = ?, next_run_at = ?, updated_at = datetime('now') WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, lastRunAt.UTC().Format(time.RFC3339), nextRunAt.UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("update next run for proactive task %s: %w", id, err)
	}
	return nil
}

// UpdateStatus toggles the is_active flag.
func (r *SQLiteProactiveTaskRepository) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	isActiveInt := 0
	if isActive {
		isActiveInt = 1
	}
	query := `UPDATE proactive_tasks SET is_active = ?, updated_at = datetime('now') WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, isActiveInt, id)
	if err != nil {
		return fmt.Errorf("update status for proactive task %s: %w", id, err)
	}
	return nil
}

// UpdateLastResultHash updates the last result hash for deduplication.
func (r *SQLiteProactiveTaskRepository) UpdateLastResultHash(ctx context.Context, id string, hash string) error {
	query := `UPDATE proactive_tasks SET last_result_hash = ?, updated_at = datetime('now') WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, hash, id)
	if err != nil {
		return fmt.Errorf("update last result hash for proactive task %s: %w", id, err)
	}
	return nil
}

// Delete permanently removes a task.
func (r *SQLiteProactiveTaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM proactive_tasks WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete proactive task %s: %w", id, err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("proactive task %s not found", id)
	}
	return nil
}

const selectProactiveTaskColumns = `
	SELECT id, session_id, connector_type, channel_id, target_connector, target_channel_id,
	       title, task_type, schedule_expr, timezone, prompt_condition, target_tools,
	       is_active, last_run_at, next_run_at, last_result_hash, created_at, updated_at
	FROM proactive_tasks`

func scanProactiveTaskRow(row *sql.Row) (*domain.ProactiveTask, error) {
	var t domain.ProactiveTask
	var taskTypeStr, targetToolsJSON, nextRunStr, createdAtStr, updatedAtStr string
	var lastRunStr *string
	var isActiveInt int

	err := row.Scan(
		&t.ID, &t.SessionID, &t.ConnectorType, &t.ChannelID,
		&t.TargetConnector, &t.TargetChannelID,
		&t.Title, &taskTypeStr, &t.ScheduleExpr, &t.Timezone,
		&t.PromptCondition, &targetToolsJSON,
		&isActiveInt, &lastRunStr, &nextRunStr,
		&t.LastResultHash, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	t.TaskType = domain.TaskType(taskTypeStr)
	t.IsActive = isActiveInt == 1
	t.SetTargetToolsFromJSON(targetToolsJSON)

	if nextRun, err := parseSQLiteTime(nextRunStr); err == nil {
		t.NextRunAt = nextRun
	}
	if createdAt, err := parseSQLiteTime(createdAtStr); err == nil {
		t.CreatedAt = createdAt
	}
	if updatedAt, err := parseSQLiteTime(updatedAtStr); err == nil {
		t.UpdatedAt = updatedAt
	}
	if lastRunStr != nil && *lastRunStr != "" {
		if lastRun, err := parseSQLiteTime(*lastRunStr); err == nil {
			t.LastRunAt = &lastRun
		}
	}

	return &t, nil
}

func scanProactiveTaskRows(rows *sql.Rows) ([]domain.ProactiveTask, error) {
	var list []domain.ProactiveTask
	for rows.Next() {
		var t domain.ProactiveTask
		var taskTypeStr, targetToolsJSON, nextRunStr, createdAtStr, updatedAtStr string
		var lastRunStr *string
		var isActiveInt int

		err := rows.Scan(
			&t.ID, &t.SessionID, &t.ConnectorType, &t.ChannelID,
			&t.TargetConnector, &t.TargetChannelID,
			&t.Title, &taskTypeStr, &t.ScheduleExpr, &t.Timezone,
			&t.PromptCondition, &targetToolsJSON,
			&isActiveInt, &lastRunStr, &nextRunStr,
			&t.LastResultHash, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan proactive task row: %w", err)
		}

		t.TaskType = domain.TaskType(taskTypeStr)
		t.IsActive = isActiveInt == 1
		t.SetTargetToolsFromJSON(targetToolsJSON)

		if nextRun, err := parseSQLiteTime(nextRunStr); err == nil {
			t.NextRunAt = nextRun
		}
		if createdAt, err := parseSQLiteTime(createdAtStr); err == nil {
			t.CreatedAt = createdAt
		}
		if updatedAt, err := parseSQLiteTime(updatedAtStr); err == nil {
			t.UpdatedAt = updatedAt
		}
		if lastRunStr != nil && *lastRunStr != "" {
			if lastRun, err := parseSQLiteTime(*lastRunStr); err == nil {
				t.LastRunAt = &lastRun
			}
		}

		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
