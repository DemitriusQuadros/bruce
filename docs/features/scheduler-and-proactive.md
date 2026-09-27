# Scheduler & Proactive Tasks

Bruce features an autonomous background scheduler that enables both **direct future notifications** and **scheduled AI agent briefings** across messaging channels.

---

## 1. Architecture Overview

The scheduler consists of three interacting systems:
1. **SQLite Storage (`proactive_tasks` table)**: Stores schedules, conditions, target channels, and state.
2. **Background Poller (`internal/scheduler/poller.go`)**: Ticks once every minute, finds due tasks (`next_run_at <= now AND is_active = 1`), recalculates next run times, and enqueues execution jobs to Redis.
3. **Asynq Worker Engine (`internal/worker/proactive_handlers.go`)**: Dequeues and executes the tasks according to their configured **Execution Mode**.

---

## 2. Dual Execution Modes

Bruce explicitly distinguishes between **Direct Message Delivery** and **AI Agent Briefing**:

```
┌────────────────────────────────────────────────────────┐
│               Scheduled Task (Cron)                    │
└──────────────────────────┬─────────────────────────────┘
                           │
             ┌─────────────┴─────────────┐
             ▼                           ▼
    execution_mode: "message"   execution_mode: "agent"
      (Direct Notification)        (AI Dynamic Task)
             │                           │
     Bypasses LLM                Runs Agent Loop
     0 tokens, 0 latency         Executes tools & compiles report
```

### Mode 1: Direct Message Delivery (`execution_mode: "message"`)
Designed for **reminders, alarms, and specific scheduled notifications**.
- **Bypasses the LLM completely** at trigger time.
- **Zero latency, zero token cost**, completely immune to LLM rate limits or quota exhaustion.
- Extracts and cleans the message text directly from `prompt_condition` (stripping wrapper quotes or command boilerplate like `"Send the message: ..."`).
- Dispatches the message directly to Discord, WhatsApp, or Telegram.

**Typical Examples**:
- *"Me manda um hello world daqui a dois minutos"*
- *"Lembre-me de tomar o remédio de pressão às 20h"*
- *"Send me 'Call the dentist' at 14:30"*

---

### Mode 2: Agent Briefing Mode (`execution_mode: "agent"`)
Designed for **dynamic research, ambient checks, and tool-driven summaries**.
- At the scheduled time, the worker invokes the **autonomous agent loop** (`RunAgentLoop`).
- Bruce can browse the web, check GitHub pull requests, read files, or query databases.
- The results are synthesized into a formatted report and dispatched to your channel.

**Typical Examples**:
- *"Every weekday at 9am, search for AI news and send me a 3-bullet briefing."*
- *"Check GitHub issues every morning and alert me if there are new critical bugs."*

---

## 3. Supported Schedule Syntax

### A. Relative Offsets (One-Off Tasks)
You can schedule tasks using natural relative time expressions:
- `+2m`, `2m`, `in 2 minutes`, `daqui a 2 minutos`
- `+30m`, `in 1 hour`, `daqui a 3 horas`

**How it works under the hood**:
1. When `proactive_create` receives a relative offset (e.g. `+2m`), it computes the exact target timestamp in the configured timezone.
2. It converts the relative offset into a concrete 5-token date-specific cron expression:
   `minute hour day month *` (e.g. `15 14 28 9 *`).
3. When the poller evaluates this task after execution, it detects that the subsequent run is more than 30 days in the future (next year) and automatically marks `is_active = 0` (deactivating it so it never repeats).

### B. Recurring 5-Token Cron Expressions
Standard cron syntax supported by `robfig/cron/v3`:
```text
┌───────────── minute (0 - 59)
│ ┌─────────── hour (0 - 23)
│ │ ┌───────── day of the month (1 - 31)
│ │ │ ┌─────── month (1 - 12)
│ │ │ │ ┌───── day of the week (0 - 6, Sunday to Saturday)
│ │ │ │ │
* * * * *
```
- `0 9 * * 1-5`: Every weekday at 9:00 AM.
- `*/15 * * * *`: Every 15 minutes.
- `0 18 * * 5`: Every Friday at 6:00 PM.

### C. Ambient Condition Watches (`task_type: "watch"`)
Watches run periodically to monitor conditions rather than delivering a fixed schedule:
- **Interval**: Specified in integer minutes (e.g. `30` or `60`). Enforces a **strict 5-minute minimum** safeguard to prevent API rate limits.
- **Condition**: Natural language description of what to monitor (e.g. *"unread emails from contractors"*).
- **Tool Targeting**: Specify `target_tools` (e.g. `["gmail_search"]`) to limit tool execution to specific tools.
- **Deduplication Hashing**: Bruce hashes the evaluation result (`last_result_hash`). If the condition has not changed since the last check, no alert is sent, avoiding notification spam.

---

## 4. Cross-Channel Routing

Schedules are not locked to the channel where you create them:
- You can converse with Bruce on **Discord** and schedule an alert to be delivered to your **WhatsApp** phone number:
  ```json
  {
    "target_connector": "whatsapp",
    "target_channel_id": "5511999999999"
  }
  ```
- Or schedule a report from the **Web Dashboard** delivered to a private **Discord** DM channel.

---

## 5. Conversational Management Tools

You can manage all tasks conversationally through any chat connector:

| Tool | Purpose | Example Conversational Prompt |
|---|---|---|
| `proactive_create` | Create a schedule or watch | *"Me lembre de comprar café daqui a 10 minutos"* |
| `proactive_list` | List all your scheduled tasks | *"Quais são os meus agendamentos ativos?"* |
| `proactive_toggle` | Pause or resume a task | *"Pausa o meu relatório diário de notícias"* |
| `proactive_delete` | Delete a scheduled task | *"Deleta o lembrete de café"* |

---

## 6. REST API & Web Dashboard

You can also manage tasks programmatically:
- `GET /api/v1/proactive-tasks` — List all tasks.
- `POST /api/v1/proactive-tasks` — Create a new proactive task.
- `PATCH /api/v1/proactive-tasks/{id}` — Update schedule, execution mode, or active status.
- `POST /api/v1/proactive-tasks/{id}/run` — Manually trigger an immediate test run.
- `DELETE /api/v1/proactive-tasks/{id}` — Delete a task.

The **Schedules** tab in the Web Dashboard (`http://localhost:8080`) provides visual controls, countdown timers, execution mode badges, and a modal editor.
