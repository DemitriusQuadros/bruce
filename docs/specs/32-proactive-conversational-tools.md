# Spec 34: Proactive Conversational Tools [BACKEND]

## Overview

Implement the LLM tools that allow users to create, list, toggle (pause/resume), and delete ambient watches and scheduled reports directly through natural language conversation across all messaging platforms.

## Phase

**Phase 1 & Phase 2** (Weeks 2–5)

## Prerequisites

- Spec 32 (`ProactiveTaskRepository`)
- Tool Registry established (`internal/tools/registry.go`)
- `tools.Tool` interface established (`internal/tools/tool.go`)

## Deliverables

### Files to Create:
- `internal/tools/proactive/create.go` — `proactive_create` tool
- `internal/tools/proactive/list.go` — `proactive_list` tool
- `internal/tools/proactive/toggle.go` — `proactive_toggle` tool
- `internal/tools/proactive/delete.go` — `proactive_delete` tool
- `internal/tools/proactive/proactive_test.go` — Unit tests covering tool invocation and validation

### Files to Modify:
- `cmd/bruce/main.go` — Register proactive tools with `tools.Registry`

---

## Tool Definitions

### 1. `proactive_create`
Allows the agent to register a new recurring cron schedule or ambient watch.

- **Parameters**:
  - `title` (string, required): Short descriptive title (e.g., "Daily Standup Briefing", "Client Email Watch").
  - `type` (string, required): `"cron"` (time-of-day report) or `"watch"` (interval condition monitor).
  - `schedule` (string, required): Standard cron expression (`"0 9 * * 1-5"`) or interval in minutes (`"30"`).
  - `prompt_condition` (string, required): For cron: report generation prompt. For watch: trigger condition.
  - `target_connector` (string, optional): Target channel (`"whatsapp"`, `"discord"`, `"telegram"`, `"web"`). Defaults to current session connector.
  - `target_channel_id` (string, optional): Target channel ID / phone number. Defaults to current session channel.
  - `timezone` (string, optional): IANA timezone string. Defaults to host machine / São Paulo fallback.
  - `target_tools` (array of strings, optional): Specific tools to query (e.g. `["gmail_search"]`).

- **Validation Rules**:
  - For `type == "watch"`: `schedule` must parse to an integer $\ge 5$ minutes (rate limit guard).
  - For `type == "cron"`: `schedule` must be a valid 5-field cron expression.
  - `target_connector` must be one of `whatsapp`, `discord`, `telegram`, `web`.
- **Output**: Confirmation string summarizing task title, type, target connector, and computed `next_run_at`.

### 2. `proactive_list`
Returns active and paused background tasks.

- **Parameters**:
  - `all_sessions` (boolean, optional, default: false): If true, lists tasks across all sessions; otherwise only current session.
- **Output**: Clean Markdown-formatted list containing:
  - Task title & ID
  - Type (Watch / Cron)
  - Target channel
  - Schedule expression & resolved timezone
  - Status (Active / Paused)
  - Next scheduled run time (human-readable format, e.g. "Tomorrow at 09:00 AM")

### 3. `proactive_toggle`
Pauses or resumes an existing background task without deleting it.

- **Parameters**:
  - `task_id_or_title` (string, required): UUID or exact title of the task.
  - `action` (string, required): `"pause"` or `"resume"`.
- **Output**: Confirmation message indicating new state.

### 4. `proactive_delete`
Permanently cancels and deletes a background task.

- **Parameters**:
  - `task_id_or_title` (string, required): UUID or exact title of the task.
- **Output**: Confirmation message that the task was removed.

---

## Acceptance Criteria

- [ ] `proactive_create` rejects intervals < 5 minutes for watches with descriptive error
- [ ] `proactive_create` rejects invalid cron syntax with clear syntax guidance
- [ ] Supports cross-channel routing: if user asks from Discord to send report to WhatsApp, `TargetConnector` and `TargetChannelID` are recorded correctly
- [ ] `proactive_list` formats human-friendly Markdown output with next run times
- [ ] `proactive_toggle` correctly flips `is_active` in SQLite
- [ ] `proactive_delete` deletes task from database
