# PRD: Bruce Proactive Monitoring ("Watches")

> **Status**: Draft — v1.0
> **Date**: 2026-03-15
> **Author**: Product (via /product-manager-prd)

---

## 1. 📌 Product Overview

**One-liner**: Bruce monitors your inbox and calendar in the background and proactively messages you when something you care about happens — configured entirely through conversation.

**Problem Statement**: Today Bruce is purely reactive — it only responds when you message it first. A personal assistant's highest value is making sure you don't miss things, not just answering when asked. Users forget to check their email for important contacts or miss calendar prep reminders because there is no ambient monitoring. Current workaround: manually asking Bruce to check, or maintaining fragile third-party rule engines.

**Solution**: Users configure "watches" in natural language through conversation ("keep me posted if a contractor emails me, check every 30 min"). Bruce stores the watch, runs it on a background schedule, and proactively messages the user through the same connector (Discord, WhatsApp, web) if the condition is met. No new interface required — the conversation is the UI.

**Primary User**: A solo power user who already relies on Bruce as a personal assistant across Discord/WhatsApp, has Gmail and Calendar tools enabled, and wants Bruce to act like a proactive chief of staff, not just a chatbot.

**Stage assumption**: MVP / single-user — optimize for correctness and configurability, not multi-tenancy.

---

## 2. 🎯 Goals & Success Metrics

| Goal | Launch metric | 6-month target |
|------|--------------|----------------|
| User can create a watch via conversation | 1 watch created end-to-end without config file edits | 10+ watches in active use |
| Background polling runs reliably | Zero missed poll cycles in 48h smoke test | 99.9% poll cycle reliability |
| Proactive notifications delivered | Notification arrives within 1 poll interval of condition firing | Same |
| User can manage watches via conversation | list/pause/delete work in chat | Same |
| No notification spam | LLM decides relevance before notifying | < 5% false-positive reports from user |

---

## 3. 👤 User Stories

### Theme A — Creating Watches

**As a** user, **I want to** tell Bruce in natural language to watch my inbox for emails from contractors every 30 minutes, **so that** I'm notified without manually checking throughout the day.

**As a** user, **I want to** tell Bruce to remind me 30 minutes before a specific calendar event type (e.g., nutritionist), **so that** I can prepare without setting manual phone alarms.

**As a** user, **I want to** specify the condition loosely ("emails that look urgent from clients"), **so that** Bruce uses LLM reasoning to decide relevance rather than rigid pattern matching.

### Theme B — Managing Watches

**As a** user, **I want to** ask Bruce "what are you watching for me?", **so that** I can see all active watches in a readable list without knowing watch IDs.

**As a** user, **I want to** pause or delete a watch by saying "stop watching for contractor emails", **so that** I can manage watches without a dashboard.

**As a** user, **I want to** change a watch's interval by saying "check every hour instead of 30 min", **so that** I can tune polling frequency as my needs change.

### Theme C — Notifications

**As a** user, **I want to** receive a proactive message from Bruce when a condition fires, **so that** I get alerted in the same chat I use for everything else.

**As a** user, **I want to** Bruce to include context in the notification (email subject + sender, or event name + time), **so that** I can decide whether to act without switching apps.

---

## 4. 🧱 Feature List with MoSCoW Prioritization

| Feature | Priority | Description | Why |
|---------|----------|-------------|-----|
| `watch_create` tool | M | LLM tool to create a watch (type, condition, interval_minutes, session_id) | Core capability |
| `watch_list` tool | M | LLM tool returning all watches for the current session | Required for management |
| `watch_delete` tool | M | LLM tool to delete a watch by ID | Required for management |
| `watches` DB table | M | Persist watches across restarts | Durability |
| Background poll loop | M | Goroutine that evaluates each due watch and enqueues Asynq tasks | Core proactive engine |
| `watch:evaluate` Asynq task | M | Handler that checks condition via Gmail/Calendar tools, dispatches notification if met | Core |
| Notification delivery | M | Use existing `dispatcherRegistry.Dispatch()` to send message to user's connector | Reuse existing infra |
| LLM-judged relevance | M | Poll result fed to LLM to decide if it's worth notifying — avoids spam | Quality gate |
| `watch_pause` / `watch_resume` tool | S | Pause without deleting a watch | UX polish |
| Watch deduplication | S | Don't re-notify for the same email/event within a cooldown window | Prevents duplicate alerts |
| `last_result` field on watch | S | Store last poll result to detect changes | Powers deduplication |
| Human-readable watch list | S | Bruce summarizes each watch in plain English, not raw JSON | UX |
| Per-watch context field | C | Extra context per watch ("I care about billing, not newsletters") | Power user |
| Watch expiry | C | Auto-delete watches after N days of inactivity | Hygiene |
| Multi-session watches | W | One watch shared across multiple connectors | Out of scope for MVP |

---

## 5. 🗺️ Product Roadmap

### Phase 1 — Core (Weeks 1–3)

**Features included**:
- `watches` DB table + migration
- `WatchRepository` (Create, List, GetDue, UpdateLastChecked, Delete)
- `watch_create`, `watch_list`, `watch_delete` LLM tools
- Background poll loop (time.Ticker, evaluates due watches)
- `watch:evaluate` Asynq task handler
- LLM relevance gate before notifying
- Notification delivery via existing `dispatcherRegistry.Dispatch()`

**Key milestone**: User says "watch my inbox every 30 min for contractor emails" → Bruce creates watch → 30 min later Bruce sends a proactive Discord/WhatsApp message when a matching email appears.

**Risks**: Gmail API rate limits with short intervals. Enforce minimum poll interval of 5 minutes.

---

### Phase 2 — Stability (Weeks 4–5)

**Features included**:
- `watch_pause` / `watch_resume` tools
- Watch deduplication (`last_result` column + cooldown window)
- Human-readable watch list summaries
- Minimum interval enforcement in tool validation
- Error backoff on failed polls (exponential, up to 4x interval)

**Key milestone**: Zero duplicate notifications in 1 week of live usage. Watch list readable without IDs.

**Risks**: LLM-judged deduplication may drift — may need deterministic semantic hash of result as fallback.

---

### Phase 3 — Polish (Weeks 6+)

**Features included**:
- Per-watch context field
- Watch expiry / auto-cleanup
- Retry on transient API failures
- Monitoring metrics for watch evaluation cycles (success rate, latency)
- `watch:evaluate` task logged to `tool_executions` table

**Key milestone**: Watches survive restarts, API outages, and 30-day unattended runs without manual intervention.

---

## 6. 🔌 Technical Considerations

- **Reuse existing delivery**: `dispatcherRegistry.Dispatch(connectorType, channelID, message)` already handles outbound messages to Discord/WhatsApp — zero new delivery code needed for Phase 1.
- **Reuse existing tools**: `email_search` (Gmail) and `calendar_read` (Calendar) are already implemented — watches invoke them via the tool executor, not direct API calls.
- **New DB table**: Add `watches` table to `schema.sql`. Key fields: `id`, `session_id`, `connector_type`, `channel_id`, `type` (email|calendar), `condition` (natural language), `interval_minutes`, `is_active`, `last_checked_at`, `last_result` (JSON), `created_at`.
- **Poll scheduling**: Use a single 1-minute `time.Ticker` in `main.go` that loads all due watches (where `last_checked_at + interval_minutes ≤ now`) and enqueues one `watch:evaluate` Asynq task per watch. Simpler than per-watch goroutines and easier to manage lifecycle.
- **Rate limit guard**: Enforce `min_interval_minutes = 5` in `watch_create` tool. Cap total concurrent evaluations via Asynq concurrency config (already set to 2).
- **LLM relevance gate**: After fetching email/calendar data, pass result to LLM: "Does this match the watch condition: '{condition}'? Reply YES or NO with a brief reason." Only dispatch notification on YES.
- **Data model core entities**: `Watch` (condition, interval, last_result), `Session` (already exists — provides connector_type + channel_id for delivery).
- **No new auth**: Watches evaluate using the same Google OAuth token already stored in `oauth_tokens` table — no extra credential management.

---

## 7. 🚀 Go-to-Market Steps

*(Internal personal tool — adapted GTM for single-user dogfood context)*

1. **Beachhead**: Single user (Demitrius) on the Discord connector, Gmail tool enabled. No multi-user concerns for Phase 1.
2. **Activation hook**: First successful end-to-end cycle — user creates a watch via chat, Bruce replies "Got it. I'll check every 30 min and message you here if I find anything matching '{condition}'." Then Bruce actually sends a proactive notification.
3. **Feedback loop**: Review `tool_executions` and `log_entries` after 1 week to measure false-positive rate and missed evaluations. Adjust LLM relevance prompt based on false positives.
4. **Expand to WhatsApp**: Once Discord is stable for 2 weeks, enable for WhatsApp connector — same `Dispatcher` interface, zero code changes.
5. **Growth unlock**: The flywheel is behavioral — every time a watch catches something the user would have missed, it reinforces reliance on Bruce. More catches → more watches created → more value.

---

## 8. ⚠️ Open Questions & Assumptions

**Assumptions baked in — validate within 30 days:**
- Assumed: notifications are delivered to the same connector/channel the watch was created in. Revisit if cross-connector delivery (e.g., watch created on Discord, notification via WhatsApp) is needed.
- Assumed: Gmail and Calendar tools are already enabled in config when watches are used. No graceful degradation planned for Phase 1 if tools are disabled.
- Assumed: minimum poll interval of 5 minutes is acceptable. Revisit if near-realtime (< 1 min) email monitoring is needed — would require webhook/push approach instead of polling.
- Assumed: up to 20 active watches per session is a safe cap. Revisit after seeing real usage patterns.

**Deliberate decisions:**
- "Chose polling over webhooks for Phase 1" — webhooks require a public endpoint and more infrastructure. Polling is simpler and sufficient for personal use.
- "LLM relevance gate over regex matching" — more flexible, handles natural language conditions, but adds LLM API cost per evaluation. Acceptable for single-user scale.
- "Watches tied to session's connector_type + channel_id, not just session_id" — ensures notifications go to the right place even if session is recreated.
