package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
)

// Processor handles Asynq tasks for Bruce.
type Processor struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
	configRepo  repository.ConfigRepository
	llm         ai.LLMService
	dispatcher  *DispatcherRegistry
	cfg         *config.Config
}

// NewProcessor returns a Processor wired with all required dependencies.
func NewProcessor(
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	configRepo repository.ConfigRepository,
	llm ai.LLMService,
	dispatcher *DispatcherRegistry,
	cfg *config.Config,
) *Processor {
	return &Processor{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		configRepo:  configRepo,
		llm:         llm,
		dispatcher:  dispatcher,
		cfg:         cfg,
	}
}

// HandleProcessIncomingMessageTask is the Asynq handler for message:process tasks.
func (p *Processor) HandleProcessIncomingMessageTask(ctx context.Context, t *asynq.Task) error {
	var payload ProcessIncomingMessagePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	// 1. Find or create session.
	session, err := p.sessionRepo.FindOrCreate(payload.ConnectorType, payload.ChannelID)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	// 2. Skip if agent is paused for this session.
	if !session.IsActive {
		log.Printf("session %s is paused, discarding message", session.ID)
		return nil
	}

	// 3. Resolve system prompt: session-level overrides global config default.
	systemPrompt := session.SystemPrompt
	if systemPrompt == "" {
		systemPrompt, _ = p.configRepo.Get("ui.default_system_prompt")
	}

	// 4. Log incoming user message.
	userMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      "user",
		Content:   payload.Content,
		Timestamp: time.Now().UTC(),
	}
	if err := p.messageRepo.Insert(userMsg); err != nil {
		return fmt.Errorf("insert user message: %w", err)
	}

	// 5. Fetch context window for Claude.
	historyPtrs, err := p.messageRepo.GetContextWindow(session.ID, p.cfg.Claude.ContextWindow)
	if err != nil {
		return fmt.Errorf("context window: %w", err)
	}
	history := make([]domain.Message, len(historyPtrs))
	for i, m := range historyPtrs {
		history[i] = *m
	}

	// 6. Call Claude.
	response, err := p.llm.GenerateResponse(ctx, systemPrompt, history)
	if err != nil {
		if errors.Is(err, ai.ErrRateLimited) {
			return fmt.Errorf("rate limited: %w", err)
		}
		return fmt.Errorf("llm: %w", err)
	}

	// 7. Log assistant response.
	assistantMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now().UTC(),
	}
	if err := p.messageRepo.Insert(assistantMsg); err != nil {
		return fmt.Errorf("insert assistant message: %w", err)
	}

	// 8. Dispatch response back to connector.
	// Dispatch errors are logged but do not fail the task — the message is already persisted,
	// and retrying the task would re-call Claude and duplicate messages. (See ADR-004.)
	if err := p.dispatcher.Dispatch(payload.ConnectorType, payload.ChannelID, response); err != nil {
		log.Printf("dispatch error (connector=%s, channel=%s): %v",
			payload.ConnectorType, payload.ChannelID, err)
	}

	return nil
}
