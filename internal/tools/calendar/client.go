// Package calendar provides a low-level Google Calendar API client backed by OAuth 2.0 tokens.
package calendar

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Event represents a Google Calendar event.
type Event struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Description string   `json:"description,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	Location    string   `json:"location,omitempty"`
}

// EventCreateRequest holds the fields required to create a new calendar event.
type EventCreateRequest struct {
	Title       string
	Start       time.Time
	End         time.Time
	Description string
	Attendees   []string
}

// Client wraps a Google Calendar API service.
type Client struct {
	svc *calendar.Service
}

// NewClient creates a new Calendar API client using the supplied OAuth token and config.
func NewClient(ctx context.Context, token *oauth2.Token, oauthCfg *oauth2.Config) (*Client, error) {
	log.Printf("[calendar] client: building service (token_type=%q, expiry=%s)", token.TokenType, token.Expiry.String())
	src := oauthCfg.TokenSource(ctx, token)
	svc, err := calendar.NewService(ctx, option.WithTokenSource(src))
	if err != nil {
		log.Printf("[calendar] client: calendar.NewService failed: %v", err)
		return nil, fmt.Errorf("calendar: create service: %w", err)
	}
	log.Printf("[calendar] client: service built successfully")
	return &Client{svc: svc}, nil
}

// ListEvents returns events from calendarID between start and end (inclusive).
func (c *Client) ListEvents(ctx context.Context, calendarID string, start, end time.Time) ([]Event, error) {
	log.Printf("[calendar] client: Events.List — calendarID=%q timeMin=%s timeMax=%s", calendarID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	resp, err := c.svc.Events.List(calendarID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Context(ctx).
		Do()
	if err != nil {
		log.Printf("[calendar] client: Events.List raw error: %T — %v", err, err)
		return nil, mapCalendarError(err)
	}
	log.Printf("[calendar] client: Events.List OK — items=%d nextPageToken=%q", len(resp.Items), resp.NextPageToken)

	events := make([]Event, 0, len(resp.Items))
	for _, item := range resp.Items {
		e := Event{
			ID:          item.Id,
			Title:       item.Summary,
			Description: item.Description,
			Location:    item.Location,
		}

		// Prefer dateTime, fall back to date (all-day events).
		if item.Start != nil {
			if item.Start.DateTime != "" {
				e.Start = item.Start.DateTime
			} else {
				e.Start = item.Start.Date
			}
		}
		if item.End != nil {
			if item.End.DateTime != "" {
				e.End = item.End.DateTime
			} else {
				e.End = item.End.Date
			}
		}

		if len(item.Attendees) > 0 {
			e.Attendees = make([]string, 0, len(item.Attendees))
			for _, a := range item.Attendees {
				e.Attendees = append(e.Attendees, a.Email)
			}
		}

		events = append(events, e)
	}

	return events, nil
}

// CreateEvent creates a new event in calendarID and returns the created event ID.
func (c *Client) CreateEvent(ctx context.Context, calendarID string, req *EventCreateRequest) (string, error) {
	event := &calendar.Event{
		Summary:     req.Title,
		Description: req.Description,
		Start: &calendar.EventDateTime{
			DateTime: req.Start.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: req.End.Format(time.RFC3339),
		},
	}

	if len(req.Attendees) > 0 {
		event.Attendees = make([]*calendar.EventAttendee, 0, len(req.Attendees))
		for _, addr := range req.Attendees {
			event.Attendees = append(event.Attendees, &calendar.EventAttendee{Email: addr})
		}
	}

	log.Printf("[calendar] client: Events.Insert — calendarID=%q title=%q start=%s end=%s attendees=%d", calendarID, req.Title, req.Start.Format(time.RFC3339), req.End.Format(time.RFC3339), len(req.Attendees))
	created, err := c.svc.Events.Insert(calendarID, event).Context(ctx).Do()
	if err != nil {
		log.Printf("[calendar] client: Events.Insert raw error: %T — %v", err, err)
		return "", mapCalendarError(err)
	}
	log.Printf("[calendar] client: Events.Insert OK — event_id=%q htmlLink=%q", created.Id, created.HtmlLink)

	return created.Id, nil
}

// mapCalendarError converts well-known Google API errors into descriptive errors.
func mapCalendarError(err error) error {
	if apiErr, ok := err.(*googleapi.Error); ok {
		log.Printf("[calendar] client: Google API error HTTP %d — %s (details: %v)", apiErr.Code, apiErr.Message, apiErr.Details)
		switch apiErr.Code {
		case 401:
			return fmt.Errorf("token expired or revoked")
		case 429:
			return fmt.Errorf("calendar rate limit exceeded")
		}
		return fmt.Errorf("calendar api error %d: %s", apiErr.Code, apiErr.Message)
	}
	log.Printf("[calendar] client: non-API error: %v", err)
	return err
}
