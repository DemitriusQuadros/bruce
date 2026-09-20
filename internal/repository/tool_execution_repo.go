package repository

import (
	"database/sql"

	"bruce/internal/domain"
)

// ToolExecutionRepository defines read operations on the tool_executions table.
type ToolExecutionRepository interface {
	GetBySession(sessionID string, limit, offset int) ([]*domain.ToolExecution, error)
	CountBySession(sessionID string) (int, error)
	GetAll(limit, offset int) ([]*domain.ToolExecution, error)
	CountAll() (int, error)
}

type toolExecutionRepo struct{ db *sql.DB }

// NewToolExecutionRepository returns a SQLite-backed ToolExecutionRepository.
func NewToolExecutionRepository(db *sql.DB) ToolExecutionRepository {
	return &toolExecutionRepo{db: db}
}

func (r *toolExecutionRepo) GetBySession(sessionID string, limit, offset int) ([]*domain.ToolExecution, error) {
	rows, err := r.db.Query(`
		SELECT id, session_id, tool_name, input, output, latency_ms, success, error_msg, executed_at
		FROM tool_executions
		WHERE session_id = ?
		ORDER BY executed_at DESC
		LIMIT ? OFFSET ?`,
		sessionID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.ToolExecution
	for rows.Next() {
		var e domain.ToolExecution
		var successInt int
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.ToolName, &e.Input, &e.Output,
			&e.LatencyMS, &successInt, &e.ErrorMsg, &e.ExecutedAt,
		); err != nil {
			return nil, err
		}
		e.Success = successInt != 0
		result = append(result, &e)
	}
	return result, rows.Err()
}

func (r *toolExecutionRepo) CountBySession(sessionID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM tool_executions WHERE session_id = ?`,
		sessionID,
	).Scan(&count)
	return count, err
}

func (r *toolExecutionRepo) GetAll(limit, offset int) ([]*domain.ToolExecution, error) {
	rows, err := r.db.Query(`
		SELECT id, session_id, tool_name, input, output, latency_ms, success, error_msg, executed_at
		FROM tool_executions
		ORDER BY executed_at DESC
		LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.ToolExecution
	for rows.Next() {
		var e domain.ToolExecution
		var successInt int
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.ToolName, &e.Input, &e.Output,
			&e.LatencyMS, &successInt, &e.ErrorMsg, &e.ExecutedAt,
		); err != nil {
			return nil, err
		}
		e.Success = successInt != 0
		result = append(result, &e)
	}
	return result, rows.Err()
}

func (r *toolExecutionRepo) CountAll() (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM tool_executions`,
	).Scan(&count)
	return count, err
}
