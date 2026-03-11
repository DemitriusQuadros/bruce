// Package worker defines Asynq task payload types, task constructors, and task name constants.
package worker

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// TaskProcessIncomingMessage is the Asynq task type for incoming connector messages.
const TaskProcessIncomingMessage = "message:process"

// ProcessIncomingMessagePayload is the JSON payload for a message:process task.
type ProcessIncomingMessagePayload struct {
	ConnectorType string `json:"connector_type"` // "whatsapp" | "discord"
	ChannelID     string `json:"channel_id"`     // phone number or Discord channel ID
	Content       string `json:"content"`        // raw text from the user
}

// NewProcessIncomingMessageTask creates an Asynq task for processing an incoming message.
// MaxRetry is 3; Timeout is 90s (must exceed Claude's 60s HTTP timeout).
func NewProcessIncomingMessageTask(p ProcessIncomingMessagePayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskProcessIncomingMessage, payload,
		asynq.MaxRetry(3),
		asynq.Timeout(90*time.Second),
	), nil
}
