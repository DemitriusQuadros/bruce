# Spec 21: File Delivery Dispatcher [BACKEND]

## Overview

Extend the `worker.Dispatcher` interface (currently sends text-only messages) to support file delivery. Add `SendFile(channelID, filename string, data []byte) error` method. Implement for Discord (upload to channel) and Telegram (via file upload API). Enables PDF generation (Spec 18) and future file-based tool outputs (screenshots, reports, etc.) to be delivered back through user's preferred channel.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Worker dispatcher exists (`internal/worker/dispatcher.go`)
- Discord connector exists (already implemented)
- Telegram connector exists (Spec 19)
- PDF generation tool exists (Spec 18)

## Deliverables

**Files to Modify:**
- `internal/worker/dispatcher.go` — add `SendFile` method to interface + base implementation
- `internal/connectors/discord/discord.go` — implement `SendFile` (upload file + send as attachment)
- `internal/connectors/telegram/telegram.go` — implement `SendFile` (upload file to Telegram)
- `internal/worker/processor.go` — call `SendFile` when tool returns file output

## Acceptance Criteria

- [ ] `Dispatcher` interface adds: `SendFile(ctx context.Context, channelID, filename string, data []byte) error`
- [ ] Discord implementation uploads file to Discord API, sends as message attachment
- [ ] Telegram implementation sends file via Telegram `sendDocument` or `sendFile` API
- [ ] File size limits are enforced (Discord 8MB, Telegram 50MB max)
- [ ] Files are sent with correct MIME type detection
- [ ] If upload fails, error is returned to worker (worker retries or logs)
- [ ] Filename is preserved in delivery (user sees original filename in Discord/Telegram)
- [ ] Latency: upload + send <10s p95 (mocked)
- [ ] Worker calls `SendFile` automatically when tool output is binary/file data

## API / Component Contract

**Updated `Dispatcher` Interface**:
```go
type Dispatcher interface {
	Send(ctx context.Context, channelID, text string) error
	SendFile(ctx context.Context, channelID, filename string, data []byte) error
	Stop() error
}
```

**Discord Implementation**:
```go
func (c *DiscordConnector) SendFile(ctx context.Context, channelID, filename string, data []byte) error {
	// Validate size (<8MB)
	// Upload to Discord API with multipart form
	// Send message with attachment
}
```

**Telegram Implementation**:
```go
func (c *TelegramConnector) SendFile(ctx context.Context, chatID, filename string, data []byte) error {
	// Validate size (<50MB)
	// POST to /sendDocument with file data
	// Return success/error
}
```

**Worker Integration** (pseudo-code):
```go
if toolOutput.IsFile {
	err := dispatcher.SendFile(ctx, channelID, toolOutput.Filename, toolOutput.Data)
	// Handle error
} else {
	err := dispatcher.Send(ctx, channelID, toolOutput.Text)
}
```

## Out of Scope

- WhatsApp file delivery (Spec 19 covers WhatsApp text only; file delivery deferred to Phase 2)
- Image thumbnails in Discord/Telegram (send as raw file)
- File preview generation (no watermarks or summaries)
- Streaming large files (all in-memory; limit to reasonable sizes)
