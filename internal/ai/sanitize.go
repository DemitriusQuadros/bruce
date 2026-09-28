package ai

import "bruce/internal/domain"

// isPlainTextMessage returns true if the message is a regular conversational text message.
func isPlainTextMessage(m domain.Message) bool {
	return (m.Type == "" || m.Type == "text") && len(m.ToolCalls) == 0 && m.ToolResult == nil
}

// sanitizeHistory ensures messages alternate roles for plain text conversations.
// If two consecutive messages share the same role and are both plain text messages,
// their text content is merged. Tool calls and tool results are preserved as separate
// messages so providers can properly format multi-tool invocations.
// Strips system-role messages (handled separately by each provider).
func sanitizeHistory(history []domain.Message) []domain.Message {
	// Filter out system-role messages
	var filtered []domain.Message
	for _, m := range history {
		if m.Role == "system" {
			continue
		}
		filtered = append(filtered, m)
	}

	// Merge consecutive plain text messages with the same role
	var result []domain.Message
	for _, m := range filtered {
		if len(result) > 0 &&
			result[len(result)-1].Role == m.Role &&
			isPlainTextMessage(result[len(result)-1]) &&
			isPlainTextMessage(m) {
			// Merge: concatenate content with newline
			result[len(result)-1].Content += "\n" + m.Content
		} else {
			result = append(result, m)
		}
	}
	return result
}

