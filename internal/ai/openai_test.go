package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/domain"
)

func newTestOpenAIProvider(serverURL string) *openaiProvider {
	cfg := config.OpenAIConfig{
		APIKey:    "test-key",
		Model:     "gpt-4o",
		MaxTokens: 1024,
	}
	p := NewOpenAIProvider(cfg).(*openaiProvider)
	p.baseURL = serverURL
	return p
}

func TestOpenAIProvider_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		var req openaiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "gpt-4o", req.Model)
		assert.Equal(t, 1024, req.MaxTokens)

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []openaiChoice{
				{Message: openaiMessage{Role: "assistant", Content: "Hello from OpenAI!"}},
			},
		})
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	resp, err := p.GenerateResponse(context.Background(), "You are Bruce.", []domain.Message{
		{Role: "user", Content: "Hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from OpenAI!", resp)
}

func TestOpenAIProvider_SystemPromptPrepended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaiRequest
		json.NewDecoder(r.Body).Decode(&req)

		// System prompt should be first message
		require.Len(t, req.Messages, 2)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "Be helpful.", req.Messages[0].Content)
		assert.Equal(t, "user", req.Messages[1].Role)

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []openaiChoice{
				{Message: openaiMessage{Role: "assistant", Content: "OK"}},
			},
		})
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "Be helpful.", []domain.Message{
		{Role: "user", Content: "Test"},
	})
	require.NoError(t, err)
}

func TestOpenAIProvider_EmptySystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaiRequest
		json.NewDecoder(r.Body).Decode(&req)

		// No system message when prompt is empty
		require.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []openaiChoice{
				{Message: openaiMessage{Role: "assistant", Content: "OK"}},
			},
		})
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "", []domain.Message{
		{Role: "user", Content: "Test"},
	})
	require.NoError(t, err)
}

func TestOpenAIProvider_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "", []domain.Message{
		{Role: "user", Content: "Test"},
	})
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestOpenAIProvider_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "", []domain.Message{
		{Role: "user", Content: "Test"},
	})
	assert.ErrorIs(t, err, ErrProviderDown)
}

func TestOpenAIProvider_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "invalid model"}`))
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "", []domain.Message{
		{Role: "user", Content: "Test"},
	})
	assert.ErrorIs(t, err, ErrBadRequest)
	assert.Contains(t, err.Error(), "invalid model")
}

func TestOpenAIProvider_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openaiResponse{Choices: []openaiChoice{}})
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "", []domain.Message{
		{Role: "user", Content: "Test"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty choices")
}

func TestOpenAIProvider_HistorySanitization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaiRequest
		json.NewDecoder(r.Body).Decode(&req)

		// System messages from history should be stripped (only explicit system prompt kept)
		// Consecutive same-role messages should be merged
		assert.Equal(t, "system", req.Messages[0].Role)       // explicit system prompt
		assert.Equal(t, "user", req.Messages[1].Role)          // merged user messages
		assert.Contains(t, req.Messages[1].Content, "Hello\n") // merged content

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []openaiChoice{
				{Message: openaiMessage{Role: "assistant", Content: "OK"}},
			},
		})
	}))
	defer server.Close()

	p := newTestOpenAIProvider(server.URL)
	_, err := p.GenerateResponse(context.Background(), "System", []domain.Message{
		{Role: "system", Content: "should be stripped"},
		{Role: "user", Content: "Hello"},
		{Role: "user", Content: "World"},
	})
	require.NoError(t, err)
}
