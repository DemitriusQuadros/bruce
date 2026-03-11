# Spec 05: REST API Backend

## Objective
A Gorilla Mux HTTP server on port `8080` serving three concerns: configuration management,
session/prompt control, and message log retrieval. Also serves the static web UI. This API
is the control plane for Bruce — everything the web UI does goes through here.

---

## 1. Middleware Stack

```
Request
  └─► RecoveryMiddleware       (catch panics, return 500)
        └─► LoggingMiddleware  (log method, path, status, duration)
              └─► CORSMiddleware (allow * for local network access)
                    └─► Router  (Gorilla Mux)
```

```go
// internal/api/router.go
func NewRouter(handlers ...Handler) *mux.Router {
    r := mux.NewRouter()

    r.Use(recoveryMiddleware)
    r.Use(loggingMiddleware)
    r.Use(corsMiddleware)

    api := r.PathPrefix("/api/v1").Subrouter()
    for _, h := range handlers {
        h.Register(api)
    }

    // Health check (no /api/v1 prefix — used by Docker healthcheck)
    r.HandleFunc("/health", healthHandler).Methods(http.MethodGet)

    // Prometheus metrics
    r.Handle("/metrics", promhttp.Handler())

    // Static web UI — catch-all, must be registered last
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/public/")))

    return r
}
```

**CORS policy**: `Access-Control-Allow-Origin: *` is acceptable — Bruce is only accessible
on a private LAN. Do not add auth middleware in Phase 1. The API is unauthenticated by
design for single-user local use.

---

## 2. Full API Contract

### Health

```
GET /health
  Auth: none
  Returns: 200 { "status": "ok", "uptime_seconds": 142 }
```

---

### Config

```
GET /api/v1/config
  Auth: none
  Returns: 200
  Body: [
    { "key": "claude.api_key", "value": "****" },
    { "key": "ui.default_system_prompt", "value": "You are Bruce..." },
    { "key": "claude.model", "value": "claude-opus-4-6" }
  ]
  Notes:
    - Keys containing "key", "token", or "secret" are masked with "****"
    - Returns merged view: DB values take precedence over config.yml defaults
    - Returns all keys known to the system, even if not yet set in DB

PUT /api/v1/config
  Auth: none
  Body: { "key": "claude.api_key", "value": "sk-ant-..." }
  Returns: 200 { "ok": true }
  Errors: 400 (missing key or value), 422 (unknown config key)
  Notes:
    - Writes to config_entries table (not back to config.yml)
    - Does NOT restart connectors — connector restart is manual (docker compose restart)
    - Known keys: claude.api_key, claude.model, claude.max_tokens, claude.context_window,
                  ui.default_system_prompt, connectors.whatsapp.enabled,
                  connectors.discord.enabled, connectors.discord.bot_token
```

**Scope creep trap**: Do not implement live connector restart from the API in Phase 1.
Saving a Discord token takes effect on next Docker restart — document this clearly in the UI.

---

### Sessions

```
GET /api/v1/sessions
  Auth: none
  Returns: 200
  Body: [
    {
      "id": "uuid",
      "connector_type": "whatsapp",
      "channel_id": "5511999999999",
      "system_prompt": "",
      "is_active": true,
      "created_at": "2026-03-10T14:00:00Z",
      "updated_at": "2026-03-10T14:05:00Z"
    }
  ]
  Notes: Ordered by updated_at DESC (most recently active first)

GET /api/v1/sessions/{id}
  Auth: none
  Returns: 200 (Session object) | 404 (not found)

PATCH /api/v1/sessions/{id}
  Auth: none
  Body: { "system_prompt": "You are my coding assistant." }
    OR: { "is_active": false }
    OR: { "system_prompt": "...", "is_active": true }
  Returns: 200 (updated Session object)
  Errors: 400 (invalid body), 404 (session not found)
  Notes:
    - Only fields present in the body are updated (partial update)
    - Changing system_prompt does not affect in-flight tasks (takes effect on next message)
    - Setting is_active=false causes the worker to discard future messages silently
```

---

### Messages

```
GET /api/v1/sessions/{id}/messages
  Auth: none
  Query params:
    limit  (int, default: 50, max: 200)
    before (ISO-8601 datetime, optional — for pagination)
  Returns: 200
  Body: [
    {
      "id": "uuid",
      "session_id": "uuid",
      "role": "user",
      "content": "What is 2+2?",
      "timestamp": "2026-03-10T14:05:01Z"
    },
    {
      "id": "uuid",
      "session_id": "uuid",
      "role": "assistant",
      "content": "4.",
      "timestamp": "2026-03-10T14:05:03Z"
    }
  ]
  Notes:
    - Ordered by timestamp DESC (newest first) — UI reverses for display
    - 404 if session_id does not exist
    - No filtering by role — return all roles
```

---

### Connectors (Status Only — Phase 1)

```
GET /api/v1/connectors
  Auth: none
  Returns: 200
  Body: [
    {
      "type": "whatsapp",
      "enabled": true,
      "status": "connected"   // "connected" | "disconnected" | "needs_qr" | "disabled"
    },
    {
      "type": "discord",
      "enabled": false,
      "status": "disabled"
    }
  ]
  Notes:
    - Status is derived from in-memory connector state, not the DB
    - "needs_qr" = whatsmeow client not yet paired (Store.ID == nil)
    - Read-only in Phase 1 — connector lifecycle managed via config + restart
```

---

## 3. Handler Structure

Each resource gets its own handler file:

```go
// internal/api/handlers/sessions.go
type SessionHandler struct {
    sessionRepo repository.SessionRepository
}

func (h *SessionHandler) Register(r *mux.Router) {
    r.HandleFunc("/sessions", h.list).Methods(http.MethodGet)
    r.HandleFunc("/sessions/{id}", h.get).Methods(http.MethodGet)
    r.HandleFunc("/sessions/{id}", h.patch).Methods(http.MethodPatch)
    r.HandleFunc("/sessions/{id}/messages", h.messages).Methods(http.MethodGet)
}
```

All handlers follow the same pattern:
1. Parse and validate input
2. Call repository
3. Write JSON response via a shared `writeJSON(w, status, v)` helper
4. Return early on error with `writeError(w, status, message)`

---

## 4. Static File Serving

```go
// Registered last in the router — catches everything not matched above
webDir := "./web/public"
r.PathPrefix("/").Handler(http.FileServer(http.Dir(webDir)))
```

**Embedding alternative (preferred for distribution):**
```go
//go:embed web/public
var webFS embed.FS

r.PathPrefix("/").Handler(http.FileServer(http.FS(webFS)))
```

Embedding the web UI inside the binary means the Docker image needs only the binary — no
separate volume mount for static files. Use this approach in Phase 2 when the UI stabilizes.

---

## 5. Error Response Format

All error responses use a consistent shape:

```json
{
  "error": "session not found",
  "code": 404
}
```

```go
func writeError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]interface{}{"error": msg, "code": status})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}
```

---

## 6. Logging Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &responseWriter{ResponseWriter: w, status: 200}
        next.ServeHTTP(wrapped, r)
        log.Printf("%s %s %d %dms", r.Method, r.URL.Path, wrapped.status,
            time.Since(start).Milliseconds())
    })
}
```

A minimal `responseWriter` wrapper captures the status code written by the handler.

---

## ADRs

**ADR-005: Unauthenticated API**
- Decision: No auth on the REST API in Phase 1.
- Alternatives: API key in header, session cookie.
- Rationale: Bruce runs on a private LAN accessible only to the owner. Adding auth adds
  complexity (key management, cookie storage) with zero security benefit for a
  single-user local tool.
- Consequences: Do not expose port 8080 to the public internet. Document this explicitly.
  Phase 2 can add a simple API key if remote access is desired.

**ADR-006: Config stored in DB, not written back to config.yml**
- Decision: `PUT /api/v1/config` writes to `config_entries` SQLite table.
- Alternatives: Rewrite `config.yml` on every PUT.
- Rationale: `config.yml` is a bind-mounted file — writing to it from inside the container
  creates permission and race condition complexity. The DB-wins merge strategy is cleaner
  and auditable (updated_at on every row).
- Consequences: Two sources of truth for config. Document the merge rule: DB overrides YAML.

---

## Deliverable

All endpoints return correct JSON responses when tested with `curl`:
- `GET /health` → `{"status":"ok"}`
- `PUT /api/v1/config` with `claude.api_key` → persists to DB, masked on GET
- `GET /api/v1/sessions` → empty array on fresh install, populated after first message
- `PATCH /api/v1/sessions/{id}` with `{"is_active": false}` → session paused, worker discards subsequent messages
- `GET /api/v1/sessions/{id}/messages` → returns conversation history in correct order
- `/` → serves `index.html` from `./web/public/`
