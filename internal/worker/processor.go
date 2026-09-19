package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/logging"
	"bruce/internal/repository"
)

// Processor handles Asynq tasks for Bruce.
type Processor struct {
	sessionRepo      repository.SessionRepository
	messageRepo      repository.MessageRepository
	configRepo       repository.ConfigRepository
	proactiveRepo    repository.ProactiveTaskRepository
	llm              ai.LLMService
	providerRegistry *ai.ProviderRegistry
	toolRegistry     ai.ToolRegistry
	dispatcher       *DispatcherRegistry
	cfg              *config.Config
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
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		configRepo:   configRepo,
		llm:          llm,
		toolRegistry: nil, // Spec 12 will wire this
		dispatcher:   dispatcher,
		cfg:          cfg,
	}
}

// SetToolRegistry sets the tool registry for agentic loop execution.
// Called by main.go after Spec 12 ships.
func (p *Processor) SetToolRegistry(registry ai.ToolRegistry) {
	p.toolRegistry = registry
}

// HandleProcessIncomingMessageTask is the Asynq handler for message:process tasks.
func (p *Processor) HandleProcessIncomingMessageTask(ctx context.Context, t *asynq.Task) error {
	var payload ProcessIncomingMessagePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logging.Errorf("failed to unmarshal task payload: %v", err)
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logging.Debugf("task received - connector=%s, channel=%s, content_len=%d, content_preview=%s",
		payload.ConnectorType, payload.ChannelID, len(payload.Content),
		truncateForLog(payload.Content, 100))

	// 1. Find or create session.
	logging.Debugf("finding or creating session for connector=%s, channel=%s",
		payload.ConnectorType, payload.ChannelID)
	session, err := p.sessionRepo.FindOrCreate(payload.ConnectorType, payload.ChannelID)
	if err != nil {
		logging.Errorf("failed to find or create session: %v", err)
		return fmt.Errorf("session: %w", err)
	}
	logging.Debugf("session found/created - session_id=%s, is_active=%v", session.ID, session.IsActive)

	// 2. Skip if agent is paused for this session.
	if !session.IsActive {
		logging.Warnf("session %s is paused, discarding message", session.ID)
		return nil
	}

	// 3. Resolve system prompt: session-level overrides DB, DB overrides YAML default.
	systemPrompt := session.SystemPrompt
	if systemPrompt == "" {
		logging.Debug("session has no system prompt, fetching from config database")
		dbPrompt, err := p.configRepo.Get("ui.default_system_prompt")
		if err != nil {
			logging.Debug("config database has no ui.default_system_prompt, using YAML default")
			systemPrompt = p.cfg.UI.DefaultSystemPrompt
		} else {
			systemPrompt = dbPrompt
		}
	}
	logging.Debugf("system prompt resolved - len=%d, prompt_preview=%s",
		len(systemPrompt), truncateForLog(systemPrompt, 80))

	// 4. Insert incoming user message.
	userMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      "user",
		Content:   payload.Content,
		Timestamp: time.Now().UTC(),
	}
	logging.Debugf("inserting user message - msg_id=%s, session_id=%s", userMsg.ID, session.ID)
	if err := p.messageRepo.Insert(userMsg); err != nil {
		logging.Errorf("failed to insert user message: %v", err)
		return fmt.Errorf("insert user message: %w", err)
	}
	logging.Debug("user message inserted successfully")

	// 5. Fetch context window.
	logging.Debugf("fetching context window - session_id=%s, window_size=%d",
		session.ID, p.cfg.Claude.ContextWindow)
	historyPtrs, err := p.messageRepo.GetContextWindow(session.ID, p.cfg.Claude.ContextWindow)
	if err != nil {
		logging.Errorf("failed to fetch context window: %v", err)
		return fmt.Errorf("context window: %w", err)
	}
	history := make([]domain.Message, len(historyPtrs))
	for i, m := range historyPtrs {
		history[i] = *m
	}
	logging.Debugf("context window fetched - message_count=%d", len(history))
	for i, msg := range history {
		logging.Debugf("history[%d] - role=%s, len=%d, preview=%s",
			i, msg.Role, len(msg.Content), truncateForLog(msg.Content, 80))
	}

	// 6. Call LLM — choose execution path based on tool registry availability.
	logging.Debug("calling LLM")
	ctx = ai.WithSessionContext(ctx, session.ID, payload.ConnectorType, payload.ChannelID)

	var response string
	if p.toolRegistry != nil {
		// Use agentic loop when tool registry is wired (Spec 12+)
		response, err = ai.RunAgentLoop(ctx, p.llm, p.toolRegistry, systemPrompt, history, 10)
	} else {
		// Fallback to simple generation when no tool registry
		response, err = p.llm.GenerateResponse(ctx, systemPrompt, history)
	}

	if err != nil {
		if errors.Is(err, ai.ErrRateLimited) {
			logging.Warn("rate limited by LLM provider")
			return fmt.Errorf("rate limited: %w", err)
		}
		logging.Errorf("LLM call failed: %v", err)
		return fmt.Errorf("llm: %w", err)
	}
	logging.Debugf("LLM response received - len=%d, preview=%s",
		len(response), truncateForLog(response, 150))

	// 7. Insert assistant response.
	assistantMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now().UTC(),
	}
	logging.Debugf("inserting assistant message - msg_id=%s, session_id=%s", assistantMsg.ID, session.ID)
	if err := p.messageRepo.Insert(assistantMsg); err != nil {
		logging.Errorf("failed to insert assistant message: %v", err)
		return fmt.Errorf("insert assistant message: %w", err)
	}
	logging.Debug("assistant message inserted successfully")

	// 8. Dispatch response back to connector.
	// Dispatch errors are logged but do not fail the task — the message is already persisted,
	// and retrying the task would re-call Claude and duplicate messages. (See ADR-004.)
	logging.Debugf("dispatching response to connector - connector=%s, channel=%s, response_len=%d",
		payload.ConnectorType, payload.ChannelID, len(response))
	if err := p.dispatcher.Dispatch(payload.ConnectorType, payload.ChannelID, response); err != nil {
		logging.Errorf("dispatch failed (connector=%s, channel=%s): %v",
			payload.ConnectorType, payload.ChannelID, err)
	} else {
		logging.Debug("response dispatched successfully")
	}

	logging.Debugf("message processing completed successfully - session_id=%s", session.ID)
	return nil
}

// truncateForLog returns a shortened version of s for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
