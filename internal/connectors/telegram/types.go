// Package telegram provides a Telegram Bot connector for Bruce using long-polling.
package telegram

import "encoding/json"

// Update represents a Telegram Bot API Update object.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message represents a Telegram Bot API Message object.
type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	From      User   `json:"from"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"` // "private", "group", "supergroup", "channel"
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

// apiResponse is the generic Telegram Bot API response envelope.
type apiResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}
