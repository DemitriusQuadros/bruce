# Spec 35: Proactive REST API & Web UI Dashboard [BACKEND + FRONTEND]

## Overview

Expose the proactive intelligence engine via REST API endpoints and provide an intuitive web dashboard interface in `web/public` under a dedicated **"Schedules & Watches"** tab. Users can inspect all active jobs, view next run times, pause/resume tasks, delete tasks, and manually trigger test runs.

## Phase

**Phase 2** (Weeks 4–7)

## Prerequisites

- Spec 32 (`ProactiveTaskRepository`)
- Spec 33 (Asynq task dispatchers and worker handlers)
- Web UI architecture established (`web/public/`)

## Deliverables

### Files to Create:
- `internal/api/handlers/proactive_tasks.go` — REST handlers for `/api/v1/proactive-tasks`
- `internal/api/handlers/proactive_tasks_test.go` — HTTP handler unit tests
- `web/public/js/modules/schedules.js` — Vanilla ES module for the Schedules tab
- `web/public/css/components/schedules.css` — Component styles for schedule cards and badges

### Files to Modify:
- `internal/api/router.go` — Mount proactive task routes
- `web/public/index.html` — Add tab button in navigation and section container
- `web/public/js/main.js` — Initialize `schedules.js` module on route change

---

## API Contract Definitions

### Base Path: `/api/v1/proactive-tasks`

#### 1. `GET /api/v1/proactive-tasks`
- **Query Params**:
  - `session_id` (string, optional): Filter by session
  - `is_active` (bool, optional): Filter by active status
- **Response**: `200 OK`
  ```json
  [
    {
      "id": "tsk_12345",
      "session_id": "sess_abc",
      "connector_type": "discord",
      "channel_id": "123456789",
      "target_connector": "whatsapp",
      "target_channel_id": "5511999999999",
      "title": "Daily Standup Briefing",
      "task_type": "cron",
      "schedule_expr": "0 9 * * 1-5",
      "timezone": "America/Sao_Paulo",
      "prompt_condition": "Summarize today's meetings and urgent PRs",
      "is_active": true,
      "last_run_at": "2026-09-18T09:00:02-03:00",
      "next_run_at": "2026-09-19T09:00:00-03:00",
      "created_at": "2026-09-15T12:00:00Z"
    }
  ]
  ```

#### 2. `POST /api/v1/proactive-tasks`
- **Body**: JSON object with required fields: `session_id`, `title`, `task_type`, `schedule_expr`, `prompt_condition`.
- **Response**: `201 Created` with created task.

#### 3. `GET /api/v1/proactive-tasks/{id}`
- **Response**: `200 OK` with task details or `404 Not Found`.

#### 4. `PATCH /api/v1/proactive-tasks/{id}`
- **Body**: `{"is_active": false}` or updated fields.
- **Response**: `200 OK` with updated task. Automatically recalculates `next_run_at` if schedule changes.

#### 5. `DELETE /api/v1/proactive-tasks/{id}`
- **Response**: `200 OK` `{"message": "task deleted"}`.

#### 6. `POST /api/v1/proactive-tasks/{id}/run`
- **Description**: Immediately enqueues the task in Redis for an instant test run without waiting for the timer.
- **Response**: `202 Accepted` `{"message": "execution enqueued"}`.

---

## Web UI Design & Interaction

### Layout & Components
1. **Header Toolbar**:
   - Title: "Scheduled Tasks & Ambient Watches"
   - Action Button: "+ New Task" modal
   - Filter dropdown: All / Active / Paused / Watches / Crons
2. **Task Cards / Grid**:
   - **Type Badge**: Purple pill for `Cron`, Blue pill for `Watch`.
   - **Status Badge**: Green `Active`, Amber `Paused`.
   - **Target Channel**: Icon + channel ID (e.g. WhatsApp icon with number).
   - **Timing Info**: Schedule string, timezone, and countdown to `next_run_at`.
   - **Action Controls**:
     - ▶️ "Run Now" (triggers immediate test execution via `/run`)
     - ⏸️ / ⏵ "Pause / Resume" toggle
     - 🗑️ "Delete" button with confirmation prompt

---

## Acceptance Criteria

- [ ] All REST endpoints return consistent JSON error envelopes on validation failure (`{"error": "...", "code": 400}`)
- [ ] `/run` endpoint successfully pushes immediate task to Redis and returns 202 Accepted
- [ ] Web UI displays all active and paused tasks with correct countdowns
- [ ] Users can toggle tasks between active and paused states directly from the web interface
- [ ] Manual test run button provides instant toast feedback
