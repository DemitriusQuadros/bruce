package jobs

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

// CleanupLogsPayload represents the payload for a CleanupLogs task.
type CleanupLogsPayload struct {
	RetentionDays int `json:"retention_days"`
}

// CleanupLogsTask creates a new CleanupLogs task.
func CleanupLogsTask(retentionDays int) *asynq.Task {
	payload, _ := json.Marshal(CleanupLogsPayload{RetentionDays: retentionDays})
	return asynq.NewTask("monitoring:cleanup_logs", payload)
}

// HandleCleanupLogs is the task handler for CleanupLogs.
// It should be called from the worker processor.
// This is a placeholder that will be called by the worker.
func HandleCleanupLogs(ctx context.Context, t *asynq.Task) error {
	// The actual implementation will be in the processor
	return nil
}
