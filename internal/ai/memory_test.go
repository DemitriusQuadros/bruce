package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeGap(t *testing.T) {
	// Zero time
	assert.Empty(t, FormatTimeGap(time.Time{}))

	// Under 1 hour
	assert.Empty(t, FormatTimeGap(time.Now().Add(-30*time.Minute)))

	// 2 hours ago
	twoHours := FormatTimeGap(time.Now().Add(-2 * time.Hour))
	assert.Contains(t, twoHours, "2 hours ago")

	// 3 days ago
	threeDays := FormatTimeGap(time.Now().Add(-72 * time.Hour))
	assert.Contains(t, threeDays, "3 days ago")
}

func TestBuildEffectiveSystemPrompt(t *testing.T) {
	base := "You are Bruce."
	summary := "- User works on Go project."
	timeGap := "The last interaction was 2 days ago."

	res := BuildEffectiveSystemPrompt(base, summary, timeGap)
	assert.True(t, strings.HasPrefix(res, "You are Bruce."))
	assert.Contains(t, res, "<conversation_context>")
	assert.Contains(t, res, summary)
	assert.Contains(t, res, "<temporal_context>")
	assert.Contains(t, res, timeGap)

	// When empty, returns base
	assert.Equal(t, base, BuildEffectiveSystemPrompt(base, "", ""))
}
