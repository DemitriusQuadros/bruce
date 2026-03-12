package ai

import "bruce/internal/domain"

// sanitizeHistory ensures messages alternate roles.
// If two consecutive messages share the same role, merge them.
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

	// Merge consecutive messages with the same role
	var result []domain.Message
	for _, m := range filtered {
		if len(result) > 0 && result[len(result)-1].Role == m.Role {
			// Merge: concatenate content with newline
			result[len(result)-1].Content += "\n" + m.Content
		} else {
			result = append(result, m)
		}
	}
	return result
}
