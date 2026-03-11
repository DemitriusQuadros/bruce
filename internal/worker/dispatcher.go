package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// Dispatcher enqueues Asynq tasks on behalf of callers.
type Dispatcher struct {
	client *asynq.Client
}

// NewDispatcher returns a new Dispatcher wrapping the given Asynq client.
func NewDispatcher(client *asynq.Client) *Dispatcher {
	return &Dispatcher{client: client}
}

// DispatchProcessIncomingMessage enqueues a message:process task.
func (d *Dispatcher) DispatchProcessIncomingMessage(ctx context.Context, payload ProcessIncomingMessagePayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ProcessIncomingMessagePayload: %w", err)
	}
	task := asynq.NewTask(TypeProcessIncomingMessage, data)
	if _, err := d.client.EnqueueContext(ctx, task); err != nil {
		return fmt.Errorf("enqueue %s: %w", TypeProcessIncomingMessage, err)
	}
	return nil
}
