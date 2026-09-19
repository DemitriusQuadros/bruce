# Spec 33: Proactive Worker Engine — Scheduled Reports, Ambient Watches & Cross-Channel Dispatcher [BACKEND]

## Overview

Implement the Asynq worker handlers that execute scheduled tasks and ambient watches in the background, including:
1. Running autonomous agent loops (`ai.RunAgentLoop`) for scheduled reports.
2. Executing read tools and evaluating natural language conditions via a fast background LLM relevance gate.
3. Cross-channel dispatching to WhatsApp, Discord, Telegram, or Web Chat via `worker.DispatcherRegistry`.
4. Fingerprint deduplication to eliminate repeated alert spam.
5. Proactive in-chat error reporting when execution fails.

## Phase

**Phase 1 & Phase 2** (Weeks 1–5)

## Prerequisites

- Spec 32 (`proactive_tasks` repository and poller)
- `worker.DispatcherRegistry` operational (`internal/worker/dispatcher.go`)
- `ai.RunAgentLoop` and Provider Registry operational (`internal/ai`)
- Existing tools (`gmail_search`, `calendar_read`, `github_*`, etc.)

## Deliverables

### Files to Create:
- `internal/worker/proactive_handlers.go` — Handler methods for `proactive:evaluate_watch` and `proactive:execute_report`
- `internal/worker/proactive_handlers_test.go` — Unit tests with mocked dispatchers and LLM service

### Files to Modify:
- `internal/worker/payloads.go` — Task type constants and payload structs
- `internal/ai/registry.go` — Add `GetBackgroundProvider()` method
- `cmd/bruce/main.go` — Register proactive task handlers on Asynq ServeMux

---

## Detailed Specifications

### 1. Task Definitions & Payloads (`internal/worker/payloads.go`)

```go
const (
    TaskEvaluateWatch          = "proactive:evaluate_watch"
    TaskExecuteScheduledReport = "proactive:execute_report"
)

type EvaluateWatchPayload struct {
    TaskID          string   `json:"task_id"`
    SessionID       string   `json:"session_id"`
    TargetConnector string   `json:"target_connector"`
    TargetChannelID string   `json:"target_channel_id"`
    Title           string   `json:"title"`
    Condition       string   `json:"condition"`
    TargetTools     []string `json:"target_tools"`
    LastResultHash  string   `json:"last_result_hash"`
}

type ExecuteScheduledReportPayload struct {
    TaskID          string `json:"task_id"`
    SessionID       string `json:"session_id"`
    TargetConnector string `json:"target_connector"`
    TargetChannelID string `json:"target_channel_id"`
    Title           string `json:"title"`
    Prompt          string `json:"prompt"`
}
```

### 2. Background LLM Provider Resolution (`internal/ai/registry.go`)

- Add method: `GetBackgroundProvider() LLMService`
- Reads `cfg.LLM.BackgroundProvider` and `cfg.LLM.BackgroundModel` (e.g. `gemini` / `gemini-2.0-flash`).
- If not configured, falls back to the default LLM provider.
- Keeps background evaluations fast, responsive, and cost-effective.

### 3. Watch Evaluation Handler (`HandleEvaluateWatchTask`)

1. **Tool Invocations**: Iterates through `payload.TargetTools` (or default tools like `gmail_search`, `calendar_read`), calling `tools.Registry.Execute(ctx, toolName, input)`.
2. **Relevance Gate**: Passes tool outputs to the background LLM service with the prompt:
   ```text
   You are an ambient alert gate for a personal assistant.
   Evaluate the following tool output against this condition:
   "{Condition}"
   
   If the content strictly matches the condition and contains new, actionable information for the user:
   Reply with:
   MATCH: <concise, human-friendly summary highlighting sender, subject, or event time>
   
   If it does NOT match or is trivial:
   Reply with:
   NO_MATCH
   ```
3. **Deduplication Check**:
   - Computes SHA256 of the matched output snippet.
   - If `hash == payload.LastResultHash`, discard without dispatching (cooldown).
4. **Dispatch**:
   - Dispatches formatted notification via `dispatcherRegistry.Dispatch(payload.TargetConnector, payload.TargetChannelID, notification)`.
   - Inserts the delivered alert into `messages` table for chat history continuity.
   - Updates `last_result_hash` in `proactiveTaskRepo`.
5. **Error Notification**:
   - If a tool fails unrecoverably or the LLM cannot be reached, sends an alert directly to the channel:
     `"⚠️ Bruce Proactive Notice: Watch '%s' encountered an error: %v"`

### 4. Scheduled Report Handler (`HandleExecuteScheduledReportTask`)

1. **Clean Context Initialization**: Prepares an isolated slice of messages containing the system prompt and the user's report prompt.
2. **Agentic Loop**: Calls `ai.RunAgentLoop(ctx, bgLLM, toolRegistry, systemPrompt, messages, 5)`.
3. **Dispatch & Persistence**:
   - Dispatches synthesized report text via `dispatcherRegistry.Dispatch(payload.TargetConnector, payload.TargetChannelID, reportText)`.
   - Persists assistant message to `messages` table.
4. **Error Notification**:
   - If report generation fails after retries, dispatches:
     `"⚠️ Bruce Proactive Notice: Failed to generate scheduled report '%s': %v"`

---

## Acceptance Criteria

- [ ] `HandleEvaluateWatchTask` extracts data, invokes relevance gate, and dispatches only when matching `MATCH:`
- [ ] Subsequent runs with identical content trigger no notification due to hash match
- [ ] `HandleExecuteScheduledReportTask` runs `ai.RunAgentLoop` with `maxRetries=5` and dispatches markdown output
- [ ] Cross-channel delivery: tasks target the specified `TargetConnector` and `TargetChannelID`
- [ ] Unrecoverable errors proactively message the user's channel with a clear notice
- [ ] Uses background LLM provider if configured in `config.yml`
