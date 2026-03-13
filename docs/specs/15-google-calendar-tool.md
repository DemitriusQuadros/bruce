# Spec 15: Google Calendar Tool [BACKEND]

## Overview

Implement `calendar_read` and `calendar_create` tools that list events within a date range and create new events. Uses Google Calendar API v3 with stored OAuth token (from Spec 13). `calendar_read` accepts `start_date`, `end_date`; `calendar_create` accepts title, start time, end time, description, attendees. Both tools update the primary calendar.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- Google OAuth flow complete (Spec 13)
- Google Calendar API enabled in OAuth scopes

## Deliverables

**Files to Create:**
- `internal/tools/calendar/calendar.go` — Tool implementations (read + create as separate methods)
- `internal/tools/calendar/client.go` — Google Calendar API wrapper

**Files to Modify:**
- `internal/tools/registry.go` — register both tools at startup
- `cmd/bruce/main.go` — instantiate Calendar tool with OAuth client
- `config.example.yml` — document Calendar tool section

## Acceptance Criteria

- [ ] Tool name `calendar_read` exists; schema includes: `start_date` (RFC3339), `end_date` (RFC3339), `calendar_id` (optional, defaults to "primary")
- [ ] Tool name `calendar_create` exists; schema includes: `title`, `start_time` (RFC3339), `end_time` (RFC3339), `description`, `attendees` (array of emails, optional)
- [ ] `calendar_read` returns list of events with: title, start time, end time, location, attendees, description
- [ ] `calendar_create` creates event and returns event ID + confirmation
- [ ] Tool handles 401 (expired token) by refreshing via Spec 13
- [ ] Tool handles 429 (rate limit) gracefully
- [ ] Tool handles invalid date formats with clear error message
- [ ] Create event is atomic (transaction-like) or idempotent
- [ ] Latency: read <2s p95, create <3s p95

## API / Component Contract

**`calendar_read` Schema**:
```json
{
	"name": "calendar_read",
	"description": "List calendar events within a date range",
	"input_schema": {
		"type": "object",
		"properties": {
			"start_date": {
				"type": "string",
				"format": "date-time",
				"description": "Start of date range (RFC3339)"
			},
			"end_date": {
				"type": "string",
				"format": "date-time",
				"description": "End of date range (RFC3339)"
			},
			"calendar_id": {
				"type": "string",
				"description": "Calendar ID; defaults to 'primary'"
			}
		},
		"required": ["start_date", "end_date"]
	}
}
```

**`calendar_create` Schema**:
```json
{
	"name": "calendar_create",
	"description": "Create a new calendar event",
	"input_schema": {
		"type": "object",
		"properties": {
			"title": { "type": "string", "description": "Event title" },
			"start_time": { "type": "string", "format": "date-time" },
			"end_time": { "type": "string", "format": "date-time" },
			"description": { "type": "string", "description": "Event description (optional)" },
			"attendees": {
				"type": "array",
				"items": { "type": "string" },
				"description": "List of attendee emails (optional)"
			}
		},
		"required": ["title", "start_time", "end_time"]
	}
}
```

**`internal/tools/calendar/client.go`**:
```go
type Client struct {
	svc   *calendar.Service
	token *oauth2.Token
}

func (c *Client) ListEvents(ctx context.Context, calendarID string, start, end time.Time) ([]Event, error)
func (c *Client) CreateEvent(ctx context.Context, calendarID string, event *EventCreateRequest) (string, error)

type Event struct {
	ID          string
	Title       string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	Description string
	Attendees   []string
}
```

## Out of Scope

- Calendar write (update/delete existing events — Phase 2+)
- Attendee RSVP status tracking
- Recurring events (simple event creation for Phase 1)
- Calendar notifications/reminders
