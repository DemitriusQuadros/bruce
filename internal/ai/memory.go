package ai

import (
	"fmt"
	"time"
)

// FormatTimeGap formats the elapsed time between last interaction and now.
// Returns an empty string if elapsed time is less than 1 hour.
func FormatTimeGap(lastTime time.Time) string {
	if lastTime.IsZero() {
		return ""
	}
	diff := time.Since(lastTime)
	if diff < time.Hour {
		return ""
	}

	var gapStr string
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			gapStr = "1 hour ago"
		} else {
			gapStr = fmt.Sprintf("%d hours ago", hours)
		}
	} else {
		days := int(diff.Hours() / 24)
		if days == 1 {
			gapStr = "yesterday"
		} else {
			gapStr = fmt.Sprintf("%d days ago", days)
		}
	}

	return fmt.Sprintf("The last interaction in this conversation was %s (%s). Keep this time gap in mind if the user refers to past discussions or recent events.",
		gapStr, lastTime.Format("2006-01-02 15:04 MST"),
	)
}

// BuildEffectiveSystemPrompt injects conversation summary and temporal context into the base system prompt.
func BuildEffectiveSystemPrompt(basePrompt, summary, timeGapNotice string) string {
	prompt := basePrompt
	if summary != "" {
		prompt += fmt.Sprintf("\n\n<conversation_context>\nBelow is a concise summary of previous conversation history and key facts for this session:\n%s\n</conversation_context>", summary)
	}
	if timeGapNotice != "" {
		prompt += fmt.Sprintf("\n\n<temporal_context>\n%s\n</temporal_context>", timeGapNotice)
	}
	return prompt
}
