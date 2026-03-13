# Spec 20: Confirmation API [BACKEND]

## Overview

Implement REST API endpoints to manage pending tool execution approvals. Add `confirmations` SQLite table schema to track pending, approved, and denied actions. Expose `GET /api/v1/confirmations` (list pending), `POST /api/v1/confirmations/{id}/approve` (approve), `POST /api/v1/confirmations/{id}/deny` (deny). Worker pauses before executing tools with `confirmed: false` and waits for approval. UI polls endpoint to show pending actions (Spec 22).

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool execution engine exists (Spec 11)
- HTTP router and handlers exist (`internal/api/handlers/`)
- SQLite schema can be extended
- Message/tool domain models exist

## Deliverables

**Files to Create:**
- `internal/api/handlers/confirmations.go` — HTTP handlers for approval endpoints
- `internal/database/migrations/002_confirmations.sql` — schema for `confirmations` table

**Files to Modify:**
- `internal/database/schema.sql` — add or link `confirmations` table
- `internal/api/router.go` — register confirmation routes
- `internal/worker/processor.go` — add approval check before tool execution
- `internal/domain/confirmation.go` — add Confirmation model

## Acceptance Criteria

- [ ] `confirmations` table schema: `id`, `session_id`, `tool_call` (JSON), `status` (enum: pending/approved/denied), `created_at`, `expires_at`
- [ ] `GET /api/v1/confirmations` returns list of pending confirmations with: id, session_id, tool_name, tool_input (preview), created_at
- [ ] `POST /api/v1/confirmations/{id}/approve` marks as approved, returns success
- [ ] `POST /api/v1/confirmations/{id}/deny` marks as denied, returns success
- [ ] Worker checks status before executing tool; if denied, returns error to Claude (Spec 11)
- [ ] Confirmations older than 1 hour auto-expire and are treated as denied
- [ ] Approvals are idempotent (approving twice has no side effect)
- [ ] UI polls endpoint every 2–3 seconds to show pending confirmations (Spec 22)
- [ ] Latency: approve/deny <500ms p95

## API / Component Contract

**Endpoints**:

```
GET /api/v1/confirmations
  Returns: [{ id, session_id, tool_name, tool_input, created_at }]

POST /api/v1/confirmations/{id}/approve
  Body: {}
  Returns: { status: "approved" }

POST /api/v1/confirmations/{id}/deny
  Body: { reason?: string }
  Returns: { status: "denied" }
```

**`confirmations` Table**:
```sql
CREATE TABLE confirmations (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL REFERENCES sessions(id),
	tool_call TEXT NOT NULL, -- JSON: { name, input }
	status TEXT NOT NULL DEFAULT 'pending', -- pending, approved, denied
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	expires_at DATETIME NOT NULL, -- 1 hour from created_at
	approved_at DATETIME,
	denied_at DATETIME
);
```

**`internal/domain/confirmation.go`**:
```go
type Confirmation struct {
	ID        string
	SessionID string
	ToolCall  ToolCall
	Status    string // pending, approved, denied
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Worker checks before executing
func (w *Worker) shouldExecuteTool(ctx context.Context, tool *ToolCall) (bool, error) {
	// Check if confirmation exists and is approved
	// If pending and expired, deny
	// If pending and not expired, wait or return error
}
```

## Out of Scope

- Multi-level approvals (only user approval for Phase 1)
- Role-based approval (single-user MVP)
- Audit trail details (approval logged but minimal)
- Notification on expiry (logged silently)
