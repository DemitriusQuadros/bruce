// Package worker defines Asynq task payload types and task name constants.
package worker

const (
	// TypeProcessIncomingMessage is the Asynq task type for incoming messages.
	TypeProcessIncomingMessage = "message:process"
)

// ProcessIncomingMessagePayload is the JSON payload for a message:process task.
type ProcessIncomingMessagePayload struct {
	SessionID string `json:"session_id"`
	MessageID string `json:"message_id"`
	Connector string `json:"connector"`
}
