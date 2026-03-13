package jobs

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

// FlushMetricsPayload represents the payload for a FlushMetrics task.
type FlushMetricsPayload struct {
	// Empty for now; can be extended with options in the future
}

// FlushMetricsTask creates a new FlushMetrics task.
func FlushMetricsTask() *asynq.Task {
	payload, _ := json.Marshal(FlushMetricsPayload{})
	return asynq.NewTask("monitoring:flush_metrics", payload)
}

// HandleFlushMetrics is the task handler for FlushMetrics.
// It should be called from the worker processor.
// This is a placeholder that will be called by the worker.
func HandleFlushMetrics(ctx context.Context, t *asynq.Task) error {
	// The actual implementation will be in the processor
	return nil
}
