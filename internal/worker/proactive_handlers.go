package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"bruce/internal/ai"
	"bruce/internal/domain"
	"bruce/internal/logging"
	"bruce/internal/repository"
)

// SetProactiveRepo injects the ProactiveTaskRepository into Processor.
func (p *Processor) SetProactiveRepo(repo repository.ProactiveTaskRepository) {
	p.proactiveRepo = repo
}

// SetProviderRegistry injects the ProviderRegistry to allow resolving background models.
func (p *Processor) SetProviderRegistry(reg *ai.ProviderRegistry) {
	p.providerRegistry = reg
}

// getBackgroundLLM returns the designated background LLM service.
func (p *Processor) getBackgroundLLM() ai.LLMService {
	if p.providerRegistry != nil {
		if bg := p.providerRegistry.GetBackgroundProvider(); bg != nil {
			return bg
		}
	}
	return p.llm
}

// notifyError proactively alerts the user via chat when a task fails.
func (p *Processor) notifyError(connectorType, channelID, title string, taskErr error) {
	if connectorType == "" || channelID == "" || p.dispatcher == nil {
		return
	}
	notice := fmt.Sprintf("⚠️ Bruce Proactive Notice: Failed to execute '%s': %v", title, taskErr)
	if err := p.dispatcher.Dispatch(connectorType, channelID, notice); err != nil {
		logging.Errorf("failed to dispatch proactive error notice to %s/%s: %v", connectorType, channelID, err)
	}
}

// HandleEvaluateWatchTask processes an ambient condition-monitoring watch.
func (p *Processor) HandleEvaluateWatchTask(ctx context.Context, t *asynq.Task) error {
	var payload EvaluateWatchPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal evaluate watch payload: %w", err)
	}

	logging.Infof("evaluating proactive watch %s (%q) for %s/%s", payload.TaskID, payload.Title, payload.TargetConnector, payload.TargetChannelID)

	// 1. Gather outputs from target tools
	var toolOutputs []string
	if p.toolRegistry != nil && len(payload.TargetTools) > 0 {
		for _, toolName := range payload.TargetTools {
			toolInput := map[string]interface{}{}
			if toolName == "email_search" {
				toolInput["query"] = "newer_than:1d"
			}
			out, err := p.toolRegistry.Execute(ctx, toolName, toolInput)
			if err != nil {
				logging.Warnf("watch %s tool %s execution error: %v", payload.TaskID, toolName, err)
				continue
			}
			if strings.TrimSpace(out) != "" {
				toolOutputs = append(toolOutputs, fmt.Sprintf("[%s]: %s", toolName, out))
			}
		}
	}

	// If no outputs were retrieved, nothing to evaluate
	if len(toolOutputs) == 0 {
		logging.Debugf("watch %s yielded no tool outputs, skipping relevance gate", payload.TaskID)
		return nil
	}

	combinedOutput := strings.Join(toolOutputs, "\n\n")

	// 2. Evaluate relevance via Background LLM
	gatePrompt := fmt.Sprintf(`You are an ambient alert gate for Bruce personal AI assistant.
Evaluate the following tool output against this watch condition:
"%s"

If the content strictly matches the condition and contains new, actionable information for the user:
Reply ONLY in this format:
MATCH: <concise, human-friendly summary highlighting the urgent item, sender, subject, or time>

If it does NOT match or contains nothing requiring user attention:
Reply with:
NO_MATCH`, payload.Condition)

	bgLLM := p.getBackgroundLLM()
	if bgLLM == nil {
		err := fmt.Errorf("no LLM service available for watch evaluation")
		p.notifyError(payload.TargetConnector, payload.TargetChannelID, payload.Title, err)
		return err
	}

	resp, err := bgLLM.GenerateResponse(ctx, gatePrompt, []domain.Message{
		{Role: "user", Content: combinedOutput},
	})
	if err != nil {
		logging.Errorf("watch %s LLM evaluation error: %v", payload.TaskID, err)
		p.notifyError(payload.TargetConnector, payload.TargetChannelID, payload.Title, err)
		return fmt.Errorf("relevance gate LLM: %w", err)
	}

	resp = strings.TrimSpace(resp)
	if !strings.HasPrefix(resp, "MATCH:") {
		logging.Debugf("watch %s condition not matched (response: %q)", payload.TaskID, resp)
		return nil
	}

	summary := strings.TrimSpace(strings.TrimPrefix(resp, "MATCH:"))

	// 3. Deduplication check via SHA256
	hasher := sha256.New()
	hasher.Write([]byte(summary))
	currentHash := hex.EncodeToString(hasher.Sum(nil))

	if payload.LastResultHash != "" && payload.LastResultHash == currentHash {
		logging.Infof("watch %s alert deduplicated (hash match: %s)", payload.TaskID, currentHash[:8])
		return nil
	}

	// 4. Dispatch alert to target connector
	formattedAlert := fmt.Sprintf("🔔 **Bruce Watch Alert: %s**\n\n%s", payload.Title, summary)
	if p.dispatcher != nil {
		if err := p.dispatcher.Dispatch(payload.TargetConnector, payload.TargetChannelID, formattedAlert); err != nil {
			logging.Errorf("watch %s dispatch failed: %v", payload.TaskID, err)
		}
	}

	// 5. Record message in session history if session exists
	if payload.SessionID != "" && p.messageRepo != nil {
		_ = p.messageRepo.Insert(&domain.Message{
			ID:        uuid.New().String(),
			SessionID: payload.SessionID,
			Role:      "assistant",
			Content:   formattedAlert,
			Timestamp: time.Now(),
		})
	}

	// 6. Update last result hash in database
	if p.proactiveRepo != nil {
		if err := p.proactiveRepo.UpdateLastResultHash(ctx, payload.TaskID, currentHash); err != nil {
			logging.Warnf("failed to update last result hash for watch %s: %v", payload.TaskID, err)
		}
	}

	logging.Infof("proactive watch %s delivered successfully to %s/%s", payload.TaskID, payload.TargetConnector, payload.TargetChannelID)
	return nil
}

// HandleExecuteScheduledReportTask executes a scheduled autonomous agent loop.
func (p *Processor) HandleExecuteScheduledReportTask(ctx context.Context, t *asynq.Task) error {
	var payload ExecuteScheduledReportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal scheduled report payload: %w", err)
	}

	logging.Infof("executing scheduled report %s (%q) for %s/%s", payload.TaskID, payload.Title, payload.TargetConnector, payload.TargetChannelID)

	bgLLM := p.getBackgroundLLM()
	if bgLLM == nil {
		err := fmt.Errorf("no LLM service available for scheduled report")
		p.notifyError(payload.TargetConnector, payload.TargetChannelID, payload.Title, err)
		return err
	}

	systemPrompt := "You are Bruce, a proactive personal AI assistant. Execute the necessary tools to compile this scheduled briefing and synthesize a clear, well-structured report."
	if p.configRepo != nil {
		if defPrompt, err := p.configRepo.Get("ui.default_system_prompt"); err == nil && defPrompt != "" {
			systemPrompt = defPrompt
		}
	}

	messages := []domain.Message{
		{Role: "user", Content: payload.Prompt},
	}

	var reportText string
	var runErr error

	if p.toolRegistry != nil {
		reportText, runErr = ai.RunAgentLoop(ctx, bgLLM, p.toolRegistry, systemPrompt, messages, 5)
	} else {
		reportText, runErr = bgLLM.GenerateResponse(ctx, systemPrompt, messages)
	}

	if runErr != nil {
		logging.Errorf("scheduled report %s execution failed: %v", payload.TaskID, runErr)
		p.notifyError(payload.TargetConnector, payload.TargetChannelID, payload.Title, runErr)
		return fmt.Errorf("run agent loop for scheduled report: %w", runErr)
	}

	// Dispatch synthesized report
	formattedReport := fmt.Sprintf("📋 **%s**\n\n%s", payload.Title, reportText)
	if p.dispatcher != nil {
		if err := p.dispatcher.Dispatch(payload.TargetConnector, payload.TargetChannelID, formattedReport); err != nil {
			logging.Errorf("scheduled report %s dispatch failed: %v", payload.TaskID, err)
		}
	}

	// Persist message in session history
	if payload.SessionID != "" && p.messageRepo != nil {
		_ = p.messageRepo.Insert(&domain.Message{
			ID:        uuid.New().String(),
			SessionID: payload.SessionID,
			Role:      "assistant",
			Content:   formattedReport,
			Timestamp: time.Now(),
		})
	}

	logging.Infof("scheduled report %s delivered successfully to %s/%s", payload.TaskID, payload.TargetConnector, payload.TargetChannelID)
	return nil
}
