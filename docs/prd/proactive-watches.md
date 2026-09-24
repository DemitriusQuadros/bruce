# PRD: Bruce Proactive Intelligence — Ambient Watches & Scheduled Reports

> **Status**: Approved — Consolidated v2.0  
> **Date**: 2026-09-18  
> **Author**: Senior Product Manager (via /product-manager-prd)  
> **Supersedes**: `proactive-watches.md` (v1.0) and `scheduled-reports-cron.md` (Draft)  

---

### 1. 📌 Product Overview

- **One-liner**: Bruce proactively monitors your digital life on background intervals and executes scheduled agentic loops (e.g. morning briefings at 8:30 AM), pushing synthesized intelligence directly to your messaging apps without waiting to be prompted.
- **Problem Statement**: Today Bruce is purely reactive — it only answers when spoken to. A true executive assistant's highest value is ambient vigilance: making sure you never miss urgent developments and delivering proactive morning/evening briefings automatically. Users forget to check their inboxes, miss calendar preparation windows, and waste time asking repetitive daily questions ("what are my meetings today?", "any open PRs?") across disconnected tools.
- **Solution**: A unified proactive background engine built on SQLite and Asynq that supports two operational modes:
  1. **Ambient Watches (Condition-Driven)**: Background polling on intervals (e.g., every 30 min) where an LLM relevance gate verifies if an incoming email, event, or file change matches a conversational watch condition before notifying.
  2. **Scheduled Reports ("Bruce Cron" / Time-of-Day)**: Recurring cron execution (e.g., weekdays at 9:00 AM) that runs an autonomous agentic tool loop across Gmail, Calendar, GitHub, and Notion, dispatching a finalized executive report to your messaging channel.
- **Primary User**: A solo developer, tech lead, or busy professional who uses Bruce on WhatsApp, Discord, or Telegram, has tools enabled, and wants an autonomous chief of staff rather than a passive chatbot.
- **Stage assumption**: Single-user MVP. Scoped for rock-solid reliability, zero duplicate spam, and local execution on personal hardware.

---

### 2. 🎯 Goals & Success Metrics

| Goal | Launch metric | 6-month target |
|---|---|---|
| **Conversational Configuration** | Create a watch or scheduled report in chat in < 15 seconds | > 80% of tasks configured via chat without manual config edits |
| **Punctual & Reliable Timing** | Scheduled reports trigger within ±30s; watch poll cycles run without drops | 99.9% scheduled task execution reliability in 30-day continuous test |
| **High Signal-to-Noise Ratio** | LLM relevance gate filters false positives | < 3% false-positive notifications reported by user |
| **Outbound Dispatch Reliability** | 100% of fired notifications successfully delivered to active connector | Zero lost notifications across WhatsApp, Discord, and Telegram |
| **Resource & Token Efficiency** | Polling & scheduled tasks fit within standard 2-worker Asynq concurrency | Token spend on background tasks kept under $5/month (via fast/flash models) |

---

### 3. 👤 User Stories

#### Theme A — Scheduled Reports (Time-of-Day Crons)
> **As a** developer, **I want to** tell Bruce *"Every weekday at 9:00 AM, summarize open GitHub PRs and today's meetings"*, **so that** I have my daily agenda ready on Discord before my standup.

> **As a** manager, **I want to** schedule an end-of-week summary on Friday at 5:00 PM querying Trello and Notion, **so that** I get an automated recap of what shipped without manual aggregation.

#### Theme B — Ambient Watches (Condition Polling)
> **As a** user, **I want to** tell Bruce *"Watch my inbox for emails from contractor @domain.com every 30 minutes and alert me immediately"*, **so that** I catch urgent replies without constantly checking email.

> **As a** busy professional, **I want to** tell Bruce *"Alert me 30 minutes before any upcoming meeting that includes an external client"*, **so that** I have enough prep time without maintaining separate phone alarms.

#### Theme C — Conversational Lifecycle & Control
> **As a** user, **I want to** ask Bruce *"What are you currently watching or scheduled to run?"*, **so that** I can review all active jobs, intervals, and next run times in plain English.

> **As a** user, **I want to** pause, resume, or delete a task by saying *"Cancel the daily 9 AM report"* or *"Pause contractor email watch"*, **so that** I can adjust background behavior conversationally.

> **As a** power user, **I want** notifications delivered to the same chat channel where I set them up, complete with source context (email snippet, meeting link, PR URL), **so that** I can take immediate action.

---

### 4. 🧱 Feature List with MoSCoW Prioritization

| Feature | Priority | Description | Why |
|---|---|---|---|
| **Unified `proactive_tasks` Table** | **M** | SQLite persistence for both interval watches and cron schedules (type, cron/interval, condition/prompt, session/channel ID). | Ensures all background tasks survive application restarts. |
| **Conversational Tooling (`watch_*` & `schedule_*`)** | **M** | Tools allowing LLM to create, list, and delete watches and scheduled reports from natural language. | Natural chat is the primary UI for configuring the assistant. |
| **Asynq Task Dispatcher & Worker Handlers** | **M** | Asynq handlers for `proactive:evaluate_watch` and `proactive:execute_report`. | Leverages existing Redis worker architecture with retries and concurrency control. |
| **Background Ticker & Cron Runner** | **M** | Integration of `asynq.Scheduler` / ticker evaluating due tasks and enqueuing jobs. | Core engine orchestrating punctual execution. |
| **Outbound Dispatching via `worker.Dispatcher`** | **M** | Proactive message delivery to WhatsApp, Discord, or Telegram using existing dispatchers. | Zero new delivery code needed; reuses active connector channels. |
| **LLM Relevance Gate (for Watches)** | **M** | Evaluates fetched tool results against natural language conditions before notifying. | Eliminates false positives and notification spam. |
| **Autonomous ReAct Loop (for Reports)** | **M** | Invokes `ai.RunAgentLoop` to execute necessary tools (Calendar, GitHub, Gmail) and synthesize report. | Produces rich, aggregated summaries rather than raw API dumps. |
| **Timezone-Aware Scheduling** | **S** | Support for user timezone (`America/Sao_Paulo`, etc.) for time-of-day crons. | Guarantees morning reports trigger at user's local hour. |
| **Watch Deduplication & Fingerprinting** | **S** | Cache last evaluated result ID/hash to prevent duplicate alerts within cooldown windows. | Prevents repetitive alerts for the same unread email or event. |
| **Pause & Resume Controls** | **S** | Conversational commands to toggle tasks active/inactive without deletion. | Flexible management without re-prompting task definitions. |
| **Human-Readable Summaries** | **S** | Formats list of active watches and schedules in clean bulleted markdown. | Clean conversational UX. |
| **Web Dashboard "Schedules & Watches" View** | **S** | Embedded UI tab in `web/public` to view, pause, and manually trigger jobs. | Full operational visibility. |
| **One-Shot Delayed Reminders** | **C** | Non-recurring delayed tasks (`asynq.ProcessIn`) for reminders like *"remind me in 45m"*. | High convenience for daily productivity. |
| **Dynamic Multi-Connector Routing** | **C** | Create watch in Web UI or Discord, but route alerts to WhatsApp phone number. | Cross-platform flexibility. |
| **Multi-User Partitioning** | **W** | Isolated per-user schedules with authentication boundaries. | Explicitly out of scope for single-user self-hosted MVP. |

---

### 5. 🗺️ Product Roadmap

#### Phase 1 — Core Unified Engine (Weeks 1–3)
- **Deliverables**:
  - `proactive_tasks` SQLite schema migration (`id`, `session_id`, `connector_type`, `channel_id`, `task_type`, `schedule_expr`, `timezone`, `condition_prompt`, `is_active`, `last_run_at`, `created_at`).
  - Asynq task definitions: `TaskEvaluateWatch` and `TaskExecuteReport`.
  - Ticker/Scheduler integration in `cmd/bruce/main.go` using `asynq.Scheduler`.
  - LLM tools: `proactive_create` (handles both watch and cron), `proactive_list`, `proactive_delete`.
  - Watch relevance gate prompt and report generation agent loop.
  - Proactive notification dispatching via `dispatcherRegistry.Dispatch()`.
- **Key Milestone**: User creates a 30m email watch AND a 9:00 AM daily GitHub summary in chat; both fire on schedule and deliver notifications to WhatsApp/Discord.

#### Phase 2 — Deduplication & UX Polish (Weeks 4–5)
- **Deliverables**:
  - Alert deduplication with state hashing in `last_result` column.
  - Timezone conversion for cron expressions (`cron.WithLocation`).
  - Pause/resume commands (`proactive_toggle`).
  - Human-friendly conversational summaries for active tasks.
  - Rate-limiting guards: enforce 5-minute minimum interval on watches.
  - Web UI tab in `web/public/` displaying live proactive tasks and next run times.
- **Key Milestone**: Zero duplicate email notifications during 7 days of live usage; user tunes and pauses schedules directly from chat.

#### Phase 3 — Robustness & Advanced Reminders (Weeks 6+)
- **Deliverables**:
  - One-shot delayed reminders (`"remind me in 1 hour to check deploy"`).
  - Exponential backoff on external tool failures (Gmail/GitHub API outages).
  - Cross-connector routing (configure on Web Chat, receive on Telegram).
  - Audit logging of all background runs into `tool_executions`.
- **Key Milestone**: Unattended 30-day operation without missed schedules or memory leaks.

---

### 6. 🔌 Technical Considerations

- **Unified Data Model**: Consolidate watches and cron schedules into a single SQLite table (`proactive_tasks`) with a `task_type` discriminator (`watch` vs `cron`). Avoids maintaining two separate storage schemas and pollers.
- **Asynq Scheduling Architecture**:
  - Use `asynq.Scheduler` in `cmd/bruce/main.go` for cron schedules (e.g. `0 9 * * 1-5`).
  - Use a 1-minute poller ticker to evaluate due interval watches (`last_run_at + interval <= now`), enqueuing `TaskEvaluateWatch` into Redis.
  - Keep Asynq concurrency bounded (`Concurrency: 2`) to ensure background runs do not starve live user chat requests.
- **Agent Loop Reentrancy**:
  - For **Watches**: Worker invokes specific read tools (`gmail_search`, `calendar_read`), then passes the raw result through a lightweight LLM classification prompt: *"Does this content match condition '{condition}'? Reply YES or NO with a concise summary."*
  - For **Scheduled Reports**: Worker spins up `ai.RunAgentLoop` with a fresh session context, granting full tool access to compose an aggregated report.
- **Outbound Dispatch Integration**: Messages are delivered via `dispatcherRegistry.Dispatch(connectorType, channelID, text)`. Outbound routing works identically across WhatsApp (`whatsmeow`), Discord (`discordgo`), and Telegram.
- **Cost & Token Safeguards**:
  - Scheduled reports are capped at `maxRetries = 5` iterations and a 90s execution timeout.
  - Default background evaluations to fast/flash models (Gemini 2.0 Flash or GPT-4o-mini) to minimize token costs.

---

### 7. 🚀 Go-to-Market Steps (Dogfooding & Personal Deployment)

1. **Beachhead**: Single-user deployment on primary messaging channel (Telegram or Discord) with Google Calendar and Gmail tools enabled.
2. **Activation Hook**: The first autonomous morning briefing: waking up to an already summarized calendar and priority inbox delivered at 8:30 AM without opening any apps.
3. **High-Impact Retention Trigger**: The first "saved by Bruce" moment: Bruce catches an urgent contractor email or flags a meeting conflict 30 minutes in advance.
4. **Expansion to All Connectors**: After validating stability on Discord/Telegram, enable outbound delivery for WhatsApp via `whatsmeow`.
5. **Feedback & Quality Tuning**: Review `tool_executions` and Asynqmon (`/monitor`) weekly to track poll latencies, API rate limits, and LLM relevance false-positive rates.
6. **Community Recipe Sharing**: Document sample natural language recipes ("Daily Standup Prep", "Executive Inbox Watcher", "Sprint PR Tracker") in the repository README.

---

### 8. ⚠️ Open Questions & Assumptions

1. **Timezone Handling**: Scheduled cron jobs require accurate user localization. *Decision*: Add a `user.timezone` field in `config.yml` (e.g., `America/Sao_Paulo`), and allow the LLM to parse and store explicit timezones when requested.
2. **Context Window for Scheduled Reports**: Should morning reports include previous chat context? *Decision*: No. Scheduled reports run with a clean, isolated context to prevent hallucination drift and minimize token usage.
3. **Notification Delivery Channel**: If a user interacts across multiple apps, where should alerts go? *Decision*: Default delivery to the exact `connector_type` and `channel_id` where the watch or schedule was created.
4. **Push vs. Pull Limits**: External APIs (Gmail, GitHub) enforce rate limits. *Mitigation*: Enforce a hard minimum interval of 5 minutes for interval watches to stay safely within free-tier quotas.
