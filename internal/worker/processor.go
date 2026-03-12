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
		log.Printf("ERROR: failed to unmarshal task payload: %v", err)
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("DEBUG: task received - connector=%s, channel=%s, content_len=%d, content_preview=%s",
		payload.ConnectorType, payload.ChannelID, len(payload.Content),
		truncateForLog(payload.Content, 100))

	// 1. Find or create session.
	log.Printf("DEBUG: finding or creating session for connector=%s, channel=%s",
		payload.ConnectorType, payload.ChannelID)
	session, err := p.sessionRepo.FindOrCreate(payload.ConnectorType, payload.ChannelID)
	if err != nil {
		log.Printf("ERROR: failed to find or create session: %v", err)
		return fmt.Errorf("session: %w", err)
	}
	log.Printf("DEBUG: session found/created - session_id=%s, is_active=%v", session.ID, session.IsActive)

	// 2. Skip if agent is paused for this session.
	if !session.IsActive {
		log.Printf("WARN: session %s is paused, discarding message", session.ID)
		return nil
	}

	// 3. Resolve system prompt: session-level overrides global config default.
	systemPrompt := session.SystemPrompt
	if systemPrompt == "" {
		log.Printf("DEBUG: session has no system prompt, fetching from config")
		systemPrompt, _ = p.configRepo.Get("ui.default_system_prompt")
	}
	log.Printf("DEBUG: system prompt resolved - len=%d, prompt_preview=%s",
		len(systemPrompt), truncateForLog(systemPrompt, 80))

	// 4. Insert incoming user message.
	userMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      "user",
		Content:   payload.Content,
		Timestamp: time.Now().UTC(),
	}
	log.Printf("DEBUG: inserting user message - msg_id=%s, session_id=%s", userMsg.ID, session.ID)
	if err := p.messageRepo.Insert(userMsg); err != nil {
		log.Printf("ERROR: failed to insert user message: %v", err)
		return fmt.Errorf("insert user message: %w", err)
	}
	log.Printf("DEBUG: user message inserted successfully")

	// 5. Fetch context window for Claude.
	log.Printf("DEBUG: fetching context window - session_id=%s, window_size=%d",
		session.ID, p.cfg.Claude.ContextWindow)
	historyPtrs, err := p.messageRepo.GetContextWindow(session.ID, p.cfg.Claude.ContextWindow)
	if err != nil {
		log.Printf("ERROR: failed to fetch context window: %v", err)
		return fmt.Errorf("context window: %w", err)
	}
	history := make([]domain.Message, len(historyPtrs))
	for i, m := range historyPtrs {
		history[i] = *m
	}
	log.Printf("DEBUG: context window fetched - message_count=%d", len(history))
	for i, msg := range history {
		log.Printf("  DEBUG: history[%d] - role=%s, len=%d, preview=%s",
			i, msg.Role, len(msg.Content), truncateForLog(msg.Content, 80))
	}

	// 6. Call Claude.
	log.Printf("DEBUG: calling Claude API - model=%s, max_tokens=%d",
		p.cfg.Claude.Model, p.cfg.Claude.MaxTokens)
	response, err := p.llm.GenerateResponse(ctx, systemPrompt, history)
	if err != nil {
		if errors.Is(err, ai.ErrRateLimited) {
			log.Printf("WARN: rate limited by Claude API")
			return fmt.Errorf("rate limited: %w", err)
		}
		log.Printf("ERROR: Claude API call failed: %v", err)
		return fmt.Errorf("llm: %w", err)
	}
	log.Printf("DEBUG: Claude response received - len=%d, preview=%s",
		len(response), truncateForLog(response, 150))

	// 7. Insert assistant response.
	assistantMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now().UTC(),
	}
	log.Printf("DEBUG: inserting assistant message - msg_id=%s, session_id=%s", assistantMsg.ID, session.ID)
	if err := p.messageRepo.Insert(assistantMsg); err != nil {
		log.Printf("ERROR: failed to insert assistant message: %v", err)
		return fmt.Errorf("insert assistant message: %w", err)
	}
	log.Printf("DEBUG: assistant message inserted successfully")

	// 8. Dispatch response back to connector.
	// Dispatch errors are logged but do not fail the task — the message is already persisted,
	// and retrying the task would re-call Claude and duplicate messages. (See ADR-004.)
	log.Printf("DEBUG: dispatching response to connector - connector=%s, channel=%s, response_len=%d",
		payload.ConnectorType, payload.ChannelID, len(response))
	if err := p.dispatcher.Dispatch(payload.ConnectorType, payload.ChannelID, response); err != nil {
		log.Printf("ERROR: dispatch failed (connector=%s, channel=%s): %v",
			payload.ConnectorType, payload.ChannelID, err)
	} else {
		log.Printf("DEBUG: response dispatched successfully")
	}

	log.Printf("DEBUG: message processing completed successfully - session_id=%s", session.ID)
	return nil
}

// truncateForLog returns a shortened version of s for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
