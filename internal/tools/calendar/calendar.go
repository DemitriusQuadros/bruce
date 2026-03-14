// Package calendar provides tools.Tool implementations for Google Calendar read and create operations.
package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bruce/internal/ai"
	"bruce/internal/auth"
)

// ReadCalendarTool implements tools.Tool for listing Google Calendar events.
type ReadCalendarTool struct {
	googleAuth *auth.GoogleHandler
}

// NewReadTool creates a new ReadCalendarTool backed by the supplied GoogleHandler.
func NewReadTool(googleAuth *auth.GoogleHandler) *ReadCalendarTool {
	return &ReadCalendarTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *ReadCalendarTool) Name() string { return "calendar_read" }

// Definition returns the AI tool definition for calendar_read.
func (t *ReadCalendarTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "calendar_read",
		Description: "List Google Calendar events",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"start_date": map[string]interface{}{
					"type":        "string",
					"description": "Start of the time range in RFC3339 format (e.g. 2024-01-15T09:00:00Z)",
				},
				"end_date": map[string]interface{}{
					"type":        "string",
					"description": "End of the time range in RFC3339 format (e.g. 2024-01-15T17:00:00Z)",
				},
				"calendar_id": map[string]interface{}{
					"type":        "string",
					"description": "Calendar ID to query (defaults to \"primary\")",
				},
			},
			"required": []string{"start_date", "end_date"},
		},
	}
}

// Execute lists calendar events within the specified date range.
func (t *ReadCalendarTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[calendar] calendar_read: Execute called with input keys: %v", mapKeys(input))

	startStr, ok := input["start_date"].(string)
	if !ok || startStr == "" {
		return nil, fmt.Errorf("start_date is required and must be a non-empty RFC3339 string")
	}

	endStr, ok := input["end_date"].(string)
	if !ok || endStr == "" {
		return nil, fmt.Errorf("end_date is required and must be a non-empty RFC3339 string")
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, fmt.Errorf("start_date %q is not a valid RFC3339 timestamp: %w", startStr, err)
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, fmt.Errorf("end_date %q is not a valid RFC3339 timestamp: %w", endStr, err)
	}

	calendarID := "primary"
	if v, ok := input["calendar_id"].(string); ok && v != "" {
		calendarID = v
	}
	log.Printf("[calendar] calendar_read: calendar_id=%q start_date=%q end_date=%q", calendarID, startStr, endStr)

	log.Printf("[calendar] calendar_read: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[calendar] calendar_read: GetToken failed: %v", err)
		return nil, fmt.Errorf("calendar_read: get token: %w", err)
	}
	log.Printf("[calendar] calendar_read: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[calendar] calendar_read: building Calendar API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[calendar] calendar_read: create client failed: %v", err)
		return nil, fmt.Errorf("calendar_read: create client: %w", err)
	}
	log.Printf("[calendar] calendar_read: Calendar API client built successfully")

	log.Printf("[calendar] calendar_read: calling Events.List calendar_id=%q start=%s end=%s", calendarID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	events, err := client.ListEvents(ctx, calendarID, start, end)
	if err != nil {
		log.Printf("[calendar] calendar_read: Events.List failed: %v", err)
		return nil, fmt.Errorf("calendar_read: %w", err)
	}
	log.Printf("[calendar] calendar_read: Events.List succeeded, returned %d events", len(events))

	data, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("calendar_read: marshal response: %w", err)
	}

	return string(data), nil
}

// CreateCalendarTool implements tools.Tool for creating a new Google Calendar event.
type CreateCalendarTool struct {
	googleAuth *auth.GoogleHandler
}

// NewCreateTool creates a new CreateCalendarTool backed by the supplied GoogleHandler.
func NewCreateTool(googleAuth *auth.GoogleHandler) *CreateCalendarTool {
	return &CreateCalendarTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *CreateCalendarTool) Name() string { return "calendar_create" }

// Definition returns the AI tool definition for calendar_create.
func (t *CreateCalendarTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "calendar_create",
		Description: "Create a Google Calendar event",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Event title / summary",
				},
				"start_time": map[string]interface{}{
					"type":        "string",
					"description": "Event start time in RFC3339 format (e.g. 2024-01-15T09:00:00Z)",
				},
				"end_time": map[string]interface{}{
					"type":        "string",
					"description": "Event end time in RFC3339 format (e.g. 2024-01-15T10:00:00Z)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Optional event description",
				},
				"attendees": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Optional list of attendee email addresses",
				},
				"calendar_id": map[string]interface{}{
					"type":        "string",
					"description": "Calendar ID to create the event in (defaults to \"primary\")",
				},
			},
			"required": []string{"title", "start_time", "end_time"},
		},
	}
}

// Execute creates a new calendar event with the supplied parameters.
func (t *CreateCalendarTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[calendar] calendar_create: Execute called with input keys: %v", mapKeys(input))

	title, ok := input["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required and must be a non-empty string")
	}

	startStr, ok := input["start_time"].(string)
	if !ok || startStr == "" {
		return nil, fmt.Errorf("start_time is required and must be a non-empty RFC3339 string")
	}

	endStr, ok := input["end_time"].(string)
	if !ok || endStr == "" {
		return nil, fmt.Errorf("end_time is required and must be a non-empty RFC3339 string")
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, fmt.Errorf("start_time %q is not a valid RFC3339 timestamp: %w", startStr, err)
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, fmt.Errorf("end_time %q is not a valid RFC3339 timestamp: %w", endStr, err)
	}

	req := &EventCreateRequest{
		Title: title,
		Start: start,
		End:   end,
	}

	if v, ok := input["description"].(string); ok {
		req.Description = v
	}

	if raw, ok := input["attendees"].([]interface{}); ok {
		req.Attendees = make([]string, 0, len(raw))
		for i, v := range raw {
			addr, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("attendees[%d] must be a string", i)
			}
			req.Attendees = append(req.Attendees, addr)
		}
	}

	calendarID := "primary"
	if v, ok := input["calendar_id"].(string); ok && v != "" {
		calendarID = v
	}
	log.Printf("[calendar] calendar_create: title=%q calendar_id=%q start=%q end=%q attendees_count=%d", title, calendarID, startStr, endStr, len(req.Attendees))

	log.Printf("[calendar] calendar_create: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[calendar] calendar_create: GetToken failed: %v", err)
		return nil, fmt.Errorf("calendar_create: get token: %w", err)
	}
	log.Printf("[calendar] calendar_create: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[calendar] calendar_create: building Calendar API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[calendar] calendar_create: create client failed: %v", err)
		return nil, fmt.Errorf("calendar_create: create client: %w", err)
	}
	log.Printf("[calendar] calendar_create: Calendar API client built successfully")

	log.Printf("[calendar] calendar_create: calling Events.Insert calendar_id=%q title=%q", calendarID, title)
	eventID, err := client.CreateEvent(ctx, calendarID, req)
	if err != nil {
		log.Printf("[calendar] calendar_create: Events.Insert failed: %v", err)
		return nil, fmt.Errorf("calendar_create: %w", err)
	}
	log.Printf("[calendar] calendar_create: Events.Insert succeeded, event_id=%q", eventID)

	data, err := json.Marshal(map[string]string{"event_id": eventID, "status": "created"})
	if err != nil {
		return nil, fmt.Errorf("calendar_create: marshal response: %w", err)
	}

	return string(data), nil
}

// mapKeys returns a slice of keys from a map for debug logging.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
