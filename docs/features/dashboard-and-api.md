# Web Dashboard & REST API Reference

Bruce provides a responsive web management interface and a comprehensive REST API for administration, inspection, and direct interaction.

---

## 1. Web Dashboard Overview

The dashboard is accessible by visiting `http://localhost:8080` (or `http://<server-ip>:9090` when running via Docker Compose).

- **Embedded Assets**: The entire frontend (HTML, CSS, JS, icons) is embedded into the compiled binary via Go's `embed` package. No external web server is needed.
- **Vanilla JavaScript**: Lightweight, zero client framework bloat, fast load times.

---

## 2. Dashboard Tabs

### 1. Chat
An embedded conversational interface:
- Message Bruce directly in your web browser.
- Create multiple independent conversation sessions.
- Full markdown formatting, lists, tables, and syntax-highlighted code blocks.

### 2. Sessions
Inspect every active conversation across all platforms:
- Filter sessions by connector (**Discord**, **WhatsApp**, **Telegram**, **Web**).
- View full message history and timestamps.
- Inspect whether an active conversation summary exists for each session.
- Delete sessions (cascades and deletes associated messages and tasks).

### 3. Schedules
Manage your background proactive tasks and ambient watches:
- **Visual Badges**: Distinguishes between `Cron`, `Watch`, and `Message` delivery modes.
- **Live Countdown**: Displays the countdown timer to the next scheduled run.
- **Immediate Trigger**: Click **Run Now** to immediately enqueue an execution without waiting for the scheduled time.
- **Controls**: Pause, resume, edit, or delete any task.
- **Modal Editor**: Create new tasks with human-friendly timezone resolution and execution mode selection.

### 4. Artifacts
Gallery of all standalone HTML documents created by Bruce:
- Card gallery displaying titles, timestamps, and file sizes.
- **Live Interactive Preview**: Test and interact with the artifact directly in a sandboxed iframe.
- **Responsive Viewport Toggles**: Preview on Desktop, Tablet, and Mobile screen widths.
- Copy shareable URL or download the raw HTML file.

### 5. Logs
Real-time audit log of system operations:
- Filter by tool executions, connector dispatches, or errors.
- Inspect JSON inputs and outputs for every tool invocation.

### 6. Settings
Dynamic runtime configuration editor:
- Modify your `ui.default_system_prompt` live without touching config files.
- Switch `llm.provider` between `claude`, `gemini`, and `openai`.
- Update API tokens and credentials (sensitive fields are masked with password toggles).
- Adjust application timezone (`app.timezone`).

---

## 3. Asynq Queue Monitor (`/monitor`)

Bruce embeds [Asynqmon](https://github.com/hibiken/asynqmon) directly at `http://localhost:8080/monitor`:
- Real-time visibility into the Redis task queues (`default`, `low`, etc.).
- Inspect active, pending, scheduled, retry, and archived tasks.
- Manually cancel, delete, or re-queue failed tasks.

---

## 4. REST API Endpoint Reference

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Service health status and uptime in seconds. |
| `GET` | `/api/v1/sessions` | List all conversation sessions with channel metadata. |
| `GET` | `/api/v1/sessions/{id}` | Fetch a specific session by ID. |
| `DELETE` | `/api/v1/sessions/{id}` | Delete a session and its associated messages. |
| `GET` | `/api/v1/chat/sessions` | List sessions specifically created for the Web Chat. |
| `POST` | `/api/v1/chat/sessions` | Create a new Web Chat conversation session. |
| `POST` | `/api/v1/chat` | Send a message to Bruce via the Web Chat API. |
| `GET` | `/api/v1/proactive-tasks` | List all proactive tasks and ambient watches. |
| `POST` | `/api/v1/proactive-tasks` | Create a new proactive task (cron or watch). |
| `GET` | `/api/v1/proactive-tasks/{id}` | Get details of a proactive task. |
| `PATCH` | `/api/v1/proactive-tasks/{id}` | Update task schedule, title, prompt, or active state. |
| `POST` | `/api/v1/proactive-tasks/{id}/run` | Immediately trigger a proactive task run. |
| `DELETE` | `/api/v1/proactive-tasks/{id}` | Delete a proactive task. |
| `GET` | `/api/v1/artifacts` | List all saved HTML artifacts. |
| `GET` | `/artifacts/{id}/{filename}` | Serve raw HTML artifact file. |
| `DELETE` | `/api/v1/artifacts/{id}` | Delete an artifact from disk and registry. |
| `GET` | `/api/v1/config` | Retrieve non-sensitive runtime config entries. |
| `PUT` | `/api/v1/config` | Update a config setting dynamically. |
| `GET` | `/api/v1/connectors` | Check active status of Discord, WhatsApp, and Telegram. |
| `GET` | `/swagger/` | Interactive Swagger / OpenAPI documentation UI. |
