package discord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkMessage(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		maxLen int
		want   []string
	}{
		{
			name:   "short message",
			msg:    "hello",
			maxLen: 100,
			want:   []string{"hello"},
		},
		{
			name:   "message exactly at max length",
			msg:    "hello world",
			maxLen: 11,
			want:   []string{"hello world"},
		},
		{
			name:   "message requires chunking on word boundary",
			msg:    "hello world this is a test",
			maxLen: 11,
			want:   []string{"hello world", " this is a", " test"},
		},
		{
			name:   "message with multiple chunks",
			msg:    "one two three four five six seven",
			maxLen: 10,
			want:   []string{"one two", " three", " four five", " six seven"},
		},
		{
			name:   "long word without spaces",
			msg:    "verylongwordwithoutspaces short",
			maxLen: 10,
			want:   []string{"verylongwo", "rdwithouts", "paces", " short"},
		},
		{
			name:   "discord 2000 char limit case",
			msg:    "a b c d e f g h i j",
			maxLen: 1900,
			want:   []string{"a b c d e f g h i j"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunkMessage(tt.msg, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChunkMessage_DiscordLimit(t *testing.T) {
	// Test with Discord's 2000 character limit
	// Create a message with 4000 characters
	msg := ""
	for i := 0; i < 200; i++ {
		msg += "12345678901234567890" // 20 chars per iteration = 4000 chars total
	}

	chunks := chunkMessage(msg, 1900)

	// Should be split into at least 2 chunks
	assert.GreaterOrEqual(t, len(chunks), 2, "message should be split into multiple chunks")

	// All chunks except the last should be <= 1900 characters
	for i, chunk := range chunks[:len(chunks)-1] {
		assert.LessOrEqual(t, len(chunk), 1900, "chunk %d exceeds max length", i)
	}

	// Last chunk should be <= 1900 characters
	assert.LessOrEqual(t, len(chunks[len(chunks)-1]), 1900, "last chunk exceeds max length")

	// Concatenating chunks should give us the original message
	concatenated := ""
	for _, chunk := range chunks {
		concatenated += chunk
	}
	assert.Equal(t, msg, concatenated, "concatenated chunks should equal original message")
}
