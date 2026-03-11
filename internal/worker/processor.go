package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// Processor handles Asynq tasks for Bruce.
type Processor struct{}

// NewProcessor returns a new Processor stub.
func NewProcessor() *Processor {
	return &Processor{}
}

// HandleProcessIncomingMessage processes a message:process task.
// The real implementation will be added in a later spec.
func (p *Processor) HandleProcessIncomingMessage(ctx context.Context, t *asynq.Task) error {
	var payload ProcessIncomingMessagePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal ProcessIncomingMessagePayload: %w", err)
	}
	log.Printf("INFO: ProcessIncomingMessage stub — sessionID=%s messageID=%s connector=%s",
		payload.SessionID, payload.MessageID, payload.Connector)
	return nil
}
