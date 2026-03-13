# Spec 20: File Delivery Dispatcher [BACKEND]

## Overview

Implement a system to deliver generated files (PDFs, exports, etc.) to users via email, WhatsApp, Discord, or direct download. Creates `file_deliveries` table to track dispatch status. Adds `POST /api/v1/files/{file_id}/deliver` endpoint that accepts delivery method and recipient. Works with tools that generate files (Spec 18, etc.) and connectors (WhatsApp, Discord, Telegram).

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- File storage system exists (temp files from Spec 18 or similar)
- Email capability (Gmail tool from Spec 14)
- Connectors exist: WhatsApp, Discord, Telegram (Specs 33–34)

## Deliverables

**Files to Create:**
- `internal/api/handlers/files.go` — file delivery endpoints
- `internal/domain/file_delivery.go` — FileDelivery model
- `internal/worker/delivery_task.go` — delivery worker task payload and handler

**Files to Modify:**
- `internal/database/schema.sql` — add `file_deliveries` table
- `internal/api/router.go` — register file delivery routes
- `internal/worker/processor.go` — dispatch delivery tasks
- `config.example.yml` — document delivery methods

## Acceptance Criteria

- [ ] `POST /api/v1/files/{file_id}/deliver` accepts: `method` (email/whatsapp/discord/download), `recipient` (email or phone/user ID)
- [ ] `GET /api/v1/files/{file_id}/download` returns file with correct Content-Type and Content-Disposition headers
- [ ] Email delivery sends file as attachment via Gmail tool (Spec 14)
- [ ] WhatsApp delivery sends file to contact (uses Spec 33 connector)
- [ ] Discord delivery sends file to channel/DM (uses connector)
- [ ] `file_deliveries` table tracks: file_id, method, recipient, status (pending/sent/failed), created_at, sent_at, error_message
- [ ] Delivery tasks are queued via Asynq and processed asynchronously
- [ ] Failed deliveries retry up to 3 times with exponential backoff (Spec 23)
- [ ] Download links are valid for 24 hours; after expiry, file is cleaned up
- [ ] Latency: delivery endpoint response <100ms, actual delivery <10s
- [ ] Large files (>50MB) are handled with streaming or chunking (if applicable)

## API / Component Contract

**Endpoints**:

```
POST /api/v1/files/{file_id}/deliver
  Body: { "method": "email|whatsapp|discord", "recipient": "user@example.com or phone/channel_id" }
  Returns: { file_id, delivery_id, status: "pending", method, recipient }

GET /api/v1/files/{file_id}/download
  Returns: binary file with headers (Content-Type, Content-Disposition, Content-Length)

GET /api/v1/files/{file_id}/status
  Returns: { file_id, filename, size_bytes, created_at, deliveries: [...] }
```

**`file_deliveries` Table**:
```sql
CREATE TABLE file_deliveries (
	id TEXT PRIMARY KEY,
	file_id TEXT NOT NULL REFERENCES files(id),
	method TEXT NOT NULL, -- 'email', 'whatsapp', 'discord', 'telegram'
	recipient TEXT NOT NULL, -- email, phone number, or channel ID
	status TEXT NOT NULL DEFAULT 'pending', -- pending, sent, failed
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	sent_at DATETIME,
	error_message TEXT,
	retry_count INTEGER DEFAULT 0
);
```

**`internal/domain/file_delivery.go`**:
```go
type FileDelivery struct {
	ID           string
	FileID       string
	Method       string // email, whatsapp, discord
	Recipient    string
	Status       string // pending, sent, failed
	CreatedAt    time.Time
	SentAt       *time.Time
	ErrorMessage string
	RetryCount   int
}
```

**`internal/worker/delivery_task.go`**:
```go
type DeliveryTaskPayload struct {
	DeliveryID string
	FileID     string
	Method     string
	Recipient  string
}

func ProcessDeliveryTask(ctx context.Context, payload *DeliveryTaskPayload) error {
	// Dispatch file based on method
	// Update delivery status in DB
	// Retry on failure (Spec 23)
}
```

## Out of Scope

- Cloud storage integration (local temp files only)
- File encryption or password protection
- Delivery scheduling (send immediately)
- Delivery notifications to user
- File versioning or history

