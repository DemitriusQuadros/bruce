package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"

	"bruce/internal/domain"
	"bruce/internal/logging"
)

// HandleSummarizeSessionTask summarizes older messages in a session and stores a rolling summary.
func (p *Processor) HandleSummarizeSessionTask(ctx context.Context, t *asynq.Task) error {
	if p.summaryRepo == nil {
		logging.Debug("summaryRepo not configured; skipping session summarization")
		return nil
	}

	var payload SummarizeSessionPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal summarize payload: %w", err)
	}

	logging.Debugf("starting background summarization for session=%s", payload.SessionID)

	// 1. Verify session exists
	_, err := p.sessionRepo.GetByID(payload.SessionID)
	if err != nil {
		logging.Warnf("session %s not found for summarization: %v", payload.SessionID, err)
		return nil // don't retry if session was deleted
	}

	// 2. Check total messages
	totalMsgs, err := p.messageRepo.CountBySession(payload.SessionID)
	if err != nil {
		return fmt.Errorf("count messages: %w", err)
	}
	if totalMsgs <= 5 {
		logging.Debugf("session %s has only %d messages; skipping summarization", payload.SessionID, totalMsgs)
		return nil
	}

	// 3. Keep the most recent messages intact for immediate context, summarize the older messages.
	keepRecent := 5
	if p.cfg != nil && p.cfg.Claude.ContextWindow > 0 {
		keepRecent = p.cfg.Claude.ContextWindow / 2
		if keepRecent < 4 {
			keepRecent = 4
		}
	}
	olderMsgs, err := p.messageRepo.GetOlderMessages(payload.SessionID, keepRecent)
	if err != nil {
		return fmt.Errorf("get older messages: %w", err)
	}
	if len(olderMsgs) == 0 {
		logging.Debugf("session %s has no older messages to summarize", payload.SessionID)
		return nil
	}

	// 4. Fetch existing summary if any
	existingSummary, err := p.summaryRepo.Get(payload.SessionID)
	if err != nil {
		logging.Warnf("fetch existing summary error: %v", err)
	}
	var prevSummaryText string
	if existingSummary != nil {
		prevSummaryText = existingSummary.Summary
	}

	// 5. Format transcript of older messages
	var transcriptBuilder strings.Builder
	for _, m := range olderMsgs {
		roleName := "User"
		if m.Role == "assistant" {
			roleName = "Bruce"
		}
		timeStr := m.Timestamp.Format("2006-01-02 15:04")
		transcriptBuilder.WriteString(fmt.Sprintf("[%s] %s: %s\n", timeStr, roleName, m.Content))
	}

	// 6. Build prompt for Background LLM
	systemPrompt := `You are Bruce's background memory condensation assistant.
Your task is to maintain a concise, durable rolling summary of a conversation between a user and Bruce.

Guidelines:
1. Extract and preserve:
   - Explicit user preferences, instructions, and personal/professional facts
   - Key project goals, technical stacks, database choices, credentials/keys mentioned
   - Current state of tasks, unresolved issues, decisions made
   - Relevant file paths, repository details, and configurations
2. Discard:
   - Greetings, conversational filler, politeness, and trivial remarks
3. Output format:
   - Dense, structured bullet points grouped by topic if applicable
   - Strictly under 250 words total
   - Plain text with clean bullet points`

	userPrompt := fmt.Sprintf("Previous Summary:\n%s\n\nNew Messages to Incorporate into Summary:\n%s",
		prevSummaryText, transcriptBuilder.String(),
	)

	bgLLM := p.getBackgroundLLM()
	if bgLLM == nil {
		return fmt.Errorf("no background LLM available for summarization")
	}

	response, err := bgLLM.GenerateResponse(ctx, systemPrompt, []domain.Message{
		{
			Role:    "user",
			Content: userPrompt,
		},
	})
	if err != nil {
		logging.Errorf("summary LLM generation failed for session %s: %v", payload.SessionID, err)
		return fmt.Errorf("generate summary with background LLM: %w", err)
	}

	summaryText := strings.TrimSpace(response)
	if summaryText == "" {
		logging.Warnf("background LLM returned empty summary for session=%s", payload.SessionID)
		return nil
	}

	lastMsgID := olderMsgs[len(olderMsgs)-1].ID
	summaryRecord := &domain.SessionSummary{
		SessionID:           payload.SessionID,
		Summary:             summaryText,
		LastSummarizedMsgID: lastMsgID,
		MessageCount:        totalMsgs,
	}

	if err := p.summaryRepo.Upsert(summaryRecord); err != nil {
		return fmt.Errorf("upsert session summary: %w", err)
	}

	logging.Infof("session %s summarized successfully (%d total messages, %d older messages compressed)",
		payload.SessionID, totalMsgs, len(olderMsgs))
	return nil
}
