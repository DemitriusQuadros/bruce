# Spec 19: Telegram Connector [BACKEND]

## Overview

Implement Telegram bot connector using the Telegram Bot API. Unlike WhatsApp (which requires QR pairing), Telegram uses simple polling of the bot API. Receive messages from users, enqueue in Asynq for processing (via existing message pipeline), dispatch responses back to Telegram, and support file uploads (for PDF delivery from Spec 18). Requires `TELEGRAM_BOT_TOKEN` in config.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Asynq worker and message pipeline exist
- Message model exists (`internal/domain/message.go`)
- HTTP router exists (`internal/api/router.go`)
- Config system extended (`internal/config/config.go`)

## Deliverables

**Files to Create:**
- `internal/connectors/telegram/telegram.go` — Bot client and polling loop
- `internal/connectors/telegram/types.go` — Telegram API types
- `internal/connectors/telegram/handler.go` — Message handler (receive + dispatch)

**Files to Modify:**
- `internal/api/router.go` — optional webhook endpoint `/telegram/webhook` (if webhook mode used; polling is default)
- `internal/worker/dispatcher.go` — extend `Dispatcher` interface to support Telegram send (or use existing Send method)
- `cmd/bruce/main.go` — start Telegram connector at startup
- `internal/config/config.go` — add `connectors.telegram.bot_token`, `connectors.telegram.enabled`
- `config.example.yml` — document Telegram setup (obtain bot token from BotFather)

## Acceptance Criteria

- [ ] Telegram bot polls API for updates every 1–2s (long polling mode)
- [ ] Bot receives messages and enqueues as `ProcessMessageTask` in Asynq
- [ ] Messages are routed to existing message processor (Spec 11)
- [ ] Bot receives response text and sends back to user via Telegram API
- [ ] Bot handles text-only responses initially (file delivery in Spec 21)
- [ ] Bot handles user typing indicator / read receipts gracefully (log, don't error)
- [ ] Bot handles Telegram API errors (timeout, 429, 5xx) with exponential backoff
- [ ] Bot recovers from connection loss and resumes polling
- [ ] Bot token is read from config; bot is disabled if `enabled: false`
- [ ] Latency: receive + dispatch <5s p95 (same as Discord)

## API / Component Contract

**Config**:
```yaml
connectors:
  telegram:
    enabled: true
    bot_token: "YOUR_BOT_TOKEN"
```

**`internal/connectors/telegram/telegram.go`**:
```go
type Connector struct {
	token  string
	client *http.Client
	// polling state
}

func (c *Connector) Start(ctx context.Context) error
func (c *Connector) Stop() error
func (c *Connector) SendMessage(ctx context.Context, chatID string, text string) error
func (c *Connector) SendFile(ctx context.Context, chatID, filename string, data []byte) error

type Update struct {
	UpdateID int
	Message  *Message
}

type Message struct {
	MessageID int
	Chat      Chat
	From      User
	Text      string
	Timestamp int64
}
```

## Out of Scope

- Webhook mode (polling only for Phase 1)
- Group chat support (DM only)
- Inline buttons / interactive keyboards
- Stickers, voice messages
- User presence tracking
- Bot commands (slash commands — Phase 2+)
