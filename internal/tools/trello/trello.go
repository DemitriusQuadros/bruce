// Package trello provides tools.Tool implementations for Trello board and card operations.
package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bruce/internal/ai"
)

// BoardListResponse is the JSON response returned by the trello_board_list tool.
type BoardListResponse struct {
	Boards []BoardItem `json:"boards"`
}

// BoardItem represents a single Trello board.
type BoardItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// CardCreateResponse is the JSON response returned by the trello_card_create tool.
type CardCreateResponse struct {
	CardID    string `json:"card_id"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"` // RFC3339
}

// CardMoveResponse is the JSON response returned by the trello_card_move tool.
type CardMoveResponse struct {
	CardID  string `json:"card_id"`
	ListID  string `json:"list_id"`
	MovedAt string `json:"moved_at"` // RFC3339
}

// CardGetResponse is the JSON response returned by the trello_card_get tool.
type CardGetResponse struct {
	CardID      string   `json:"card_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DueDate     string   `json:"due_date,omitempty"`
	Labels      []string `json:"labels"`
	ListID      string   `json:"list_id"`
	ListName    string   `json:"list_name,omitempty"`
	BoardID     string   `json:"board_id"`
	URL         string   `json:"url"`
}

// TrelloBoardListTool implements tools.Tool for listing all open Trello boards.
type TrelloBoardListTool struct {
	apiKey   string
	apiToken string
}

// NewBoardListTool creates a new TrelloBoardListTool.
func NewBoardListTool(apiKey, apiToken string) *TrelloBoardListTool {
	return &TrelloBoardListTool{apiKey: apiKey, apiToken: apiToken}
}

// Name returns the tool name.
func (t *TrelloBoardListTool) Name() string { return "trello_board_list" }

// Definition returns the AI tool definition for trello_board_list.
func (t *TrelloBoardListTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "trello_board_list",
		Description: "List all open Trello boards for the authenticated user",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

// Execute lists all open Trello boards and returns them as JSON.
func (t *TrelloBoardListTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[trello] trello_board_list: Execute called")

	client := NewClient(t.apiKey, t.apiToken)

	boards, err := client.ListBoards(ctx)
	if err != nil {
		log.Printf("[trello] trello_board_list: ListBoards failed: %v", err)
		return nil, fmt.Errorf("trello_board_list: %w", err)
	}

	items := make([]BoardItem, 0, len(boards))
	for _, b := range boards {
		id, _ := b["id"].(string)
		name, _ := b["name"].(string)
		desc, _ := b["desc"].(string)
		url, _ := b["url"].(string)
		items = append(items, BoardItem{
			ID:          id,
			Name:        name,
			Description: desc,
			URL:         url,
		})
	}

	resp := BoardListResponse{Boards: items}
	log.Printf("[trello] trello_board_list: OK — boardCount=%d", len(items))

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("trello_board_list: marshal response: %w", err)
	}
	return string(data), nil
}

// TrelloCardCreateTool implements tools.Tool for creating a new Trello card.
type TrelloCardCreateTool struct {
	apiKey   string
	apiToken string
}

// NewCardCreateTool creates a new TrelloCardCreateTool.
func NewCardCreateTool(apiKey, apiToken string) *TrelloCardCreateTool {
	return &TrelloCardCreateTool{apiKey: apiKey, apiToken: apiToken}
}

// Name returns the tool name.
func (t *TrelloCardCreateTool) Name() string { return "trello_card_create" }

// Definition returns the AI tool definition for trello_card_create.
func (t *TrelloCardCreateTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "trello_card_create",
		Description: "Create a new card in a Trello list",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"list_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the Trello list to create the card in",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name (title) of the new card",
				},
				"board_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the Trello board (optional, informational only)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Optional card description",
				},
				"due_date": map[string]interface{}{
					"type":        "string",
					"description": "Optional due date for the card (date string, e.g. 2024-12-31)",
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"description": "Optional list of label IDs to attach to the card",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"list_id", "name"},
		},
	}
}

// Execute creates a new Trello card and returns its ID and URL.
func (t *TrelloCardCreateTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[trello] trello_card_create: Execute called with input keys: %v", mapKeys(input))

	listID, ok := input["list_id"].(string)
	if !ok || listID == "" {
		return nil, fmt.Errorf("list_id is required and must be a non-empty string")
	}

	name, ok := input["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required and must be a non-empty string")
	}

	description, _ := input["description"].(string)
	dueDate, _ := input["due_date"].(string)

	var labels []string
	if raw, ok := input["labels"].([]interface{}); ok {
		for _, l := range raw {
			if s, ok := l.(string); ok && s != "" {
				labels = append(labels, s)
			}
		}
	}

	// board_id is accepted in schema but silently unused — Trello's POST /cards only needs idList.
	log.Printf("[trello] trello_card_create: listID=%q name=%q", listID, name)

	client := NewClient(t.apiKey, t.apiToken)

	cardID, cardURL, err := client.CreateCard(ctx, listID, name, description, dueDate, labels)
	if err != nil {
		log.Printf("[trello] trello_card_create: CreateCard failed: %v", err)
		return nil, fmt.Errorf("trello_card_create: %w", err)
	}
	log.Printf("[trello] trello_card_create: CreateCard OK — cardID=%q url=%q", cardID, cardURL)

	resp := CardCreateResponse{
		CardID:    cardID,
		URL:       cardURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("trello_card_create: marshal response: %w", err)
	}
	return string(data), nil
}

// TrelloCardMoveTool implements tools.Tool for moving a Trello card to a different list.
type TrelloCardMoveTool struct {
	apiKey   string
	apiToken string
}

// NewCardMoveTool creates a new TrelloCardMoveTool.
func NewCardMoveTool(apiKey, apiToken string) *TrelloCardMoveTool {
	return &TrelloCardMoveTool{apiKey: apiKey, apiToken: apiToken}
}

// Name returns the tool name.
func (t *TrelloCardMoveTool) Name() string { return "trello_card_move" }

// Definition returns the AI tool definition for trello_card_move.
func (t *TrelloCardMoveTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "trello_card_move",
		Description: "Move a Trello card to a different list",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"card_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the Trello card to move",
				},
				"list_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the destination Trello list",
				},
			},
			"required": []string{"card_id", "list_id"},
		},
	}
}

// Execute moves a Trello card to a different list and returns confirmation.
func (t *TrelloCardMoveTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[trello] trello_card_move: Execute called with input keys: %v", mapKeys(input))

	cardID, ok := input["card_id"].(string)
	if !ok || cardID == "" {
		return nil, fmt.Errorf("card_id is required and must be a non-empty string")
	}

	listID, ok := input["list_id"].(string)
	if !ok || listID == "" {
		return nil, fmt.Errorf("list_id is required and must be a non-empty string")
	}
	log.Printf("[trello] trello_card_move: cardID=%q listID=%q", cardID, listID)

	client := NewClient(t.apiKey, t.apiToken)

	if err := client.MoveCard(ctx, cardID, listID); err != nil {
		log.Printf("[trello] trello_card_move: MoveCard failed: %v", err)
		return nil, fmt.Errorf("trello_card_move: %w", err)
	}
	log.Printf("[trello] trello_card_move: MoveCard OK")

	resp := CardMoveResponse{
		CardID:  cardID,
		ListID:  listID,
		MovedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("trello_card_move: marshal response: %w", err)
	}
	return string(data), nil
}

// TrelloCardGetTool implements tools.Tool for fetching details of a Trello card.
type TrelloCardGetTool struct {
	apiKey   string
	apiToken string
}

// NewCardGetTool creates a new TrelloCardGetTool.
func NewCardGetTool(apiKey, apiToken string) *TrelloCardGetTool {
	return &TrelloCardGetTool{apiKey: apiKey, apiToken: apiToken}
}

// Name returns the tool name.
func (t *TrelloCardGetTool) Name() string { return "trello_card_get" }

// Definition returns the AI tool definition for trello_card_get.
func (t *TrelloCardGetTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "trello_card_get",
		Description: "Get details of a Trello card by ID, including its list and board",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"card_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the Trello card to retrieve",
				},
			},
			"required": []string{"card_id"},
		},
	}
}

// Execute fetches a Trello card and returns its details as JSON.
func (t *TrelloCardGetTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[trello] trello_card_get: Execute called with input keys: %v", mapKeys(input))

	cardID, ok := input["card_id"].(string)
	if !ok || cardID == "" {
		return nil, fmt.Errorf("card_id is required and must be a non-empty string")
	}
	log.Printf("[trello] trello_card_get: cardID=%q", cardID)

	client := NewClient(t.apiKey, t.apiToken)

	card, err := client.GetCard(ctx, cardID)
	if err != nil {
		log.Printf("[trello] trello_card_get: GetCard failed: %v", err)
		return nil, fmt.Errorf("trello_card_get: %w", err)
	}

	labels, _ := card["labels"].([]string)
	if labels == nil {
		labels = []string{}
	}

	resp := CardGetResponse{
		CardID:      fmt.Sprintf("%v", card["id"]),
		Name:        fmt.Sprintf("%v", card["name"]),
		Description: fmt.Sprintf("%v", card["description"]),
		DueDate:     fmt.Sprintf("%v", card["due_date"]),
		Labels:      labels,
		ListID:      fmt.Sprintf("%v", card["list_id"]),
		ListName:    fmt.Sprintf("%v", card["list_name"]),
		BoardID:     fmt.Sprintf("%v", card["board_id"]),
		URL:         fmt.Sprintf("%v", card["url"]),
	}

	// Clean up empty due_date so it's omitted from JSON.
	if resp.DueDate == "" || resp.DueDate == "<nil>" {
		resp.DueDate = ""
	}
	// Clean up empty list_name so it's omitted from JSON.
	if resp.ListName == "<nil>" {
		resp.ListName = ""
	}

	log.Printf("[trello] trello_card_get: OK — name=%q listName=%q", resp.Name, resp.ListName)

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("trello_card_get: marshal response: %w", err)
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
