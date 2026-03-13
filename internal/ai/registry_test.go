package ai

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/database"
	"bruce/internal/domain"
	"bruce/internal/repository"
)

// TestProviderResolution_UsesStaticConfig verifies that llm.provider
// from config.yml is properly used when no database override exists.
func TestProviderResolution_UsesStaticConfig(t *testing.T) {
	// Setup: Create in-memory SQLite
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(database.Schema)
	require.NoError(t, err)

	// Create repositories
	configRepo := repository.NewConfigRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Create mock providers
	mockClaude := &MockLLMService{}
	mockGemini := &MockLLMService{}
	providers := map[ProviderName]LLMService{
		ProviderClaude: mockClaude,
		ProviderGemini: mockGemini,
	}

	// Test Case 1: Config has "gemini" as default
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "gemini",
		},
		Claude: config.ClaudeConfig{
			Model: "claude-3-sonnet",
		},
		Gemini: config.GeminiConfig{
			Model: "gemini-2.0-flash",
		},
	}

	registry := NewProviderRegistry(configRepo, sessionRepo, providers, cfg)

	// Setup mock expectations
	mockGemini.On("GenerateResponse", mock.Anything, mock.Anything, mock.Anything).
		Return("response from gemini", nil)

	// Call GenerateResponse - should use Gemini from config
	response, err := registry.GenerateResponse(context.Background(), "prompt", []domain.Message{})
	assert.NoError(t, err)
	assert.Equal(t, "response from gemini", response)
	mockGemini.AssertCalled(t, "GenerateResponse", mock.Anything, mock.Anything, mock.Anything)
	mockClaude.AssertNotCalled(t, "GenerateResponse")
}

// TestProviderResolution_DatabaseOverridesStatic verifies that a database
// config entry takes precedence over the static config file.
func TestProviderResolution_DatabaseOverridesStatic(t *testing.T) {
	// Setup: Create in-memory SQLite
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(database.Schema)
	require.NoError(t, err)

	// Create repositories
	configRepo := repository.NewConfigRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Set database override to Claude while static config says Gemini
	err = configRepo.Upsert("llm.provider", "claude")
	assert.NoError(t, err)

	// Create mock providers
	mockClaude := &MockLLMService{}
	mockGemini := &MockLLMService{}
	providers := map[ProviderName]LLMService{
		ProviderClaude: mockClaude,
		ProviderGemini: mockGemini,
	}

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "gemini", // Static config says gemini
		},
		Claude: config.ClaudeConfig{
			Model: "claude-3-sonnet",
		},
		Gemini: config.GeminiConfig{
			Model: "gemini-2.0-flash",
		},
	}

	registry := NewProviderRegistry(configRepo, sessionRepo, providers, cfg)

	// Setup mock expectations
	mockClaude.On("GenerateResponse", mock.Anything, mock.Anything, mock.Anything).
		Return("response from claude", nil)

	// Call GenerateResponse - should use Claude from database (overrides static)
	response, err := registry.GenerateResponse(context.Background(), "prompt", []domain.Message{})
	assert.NoError(t, err)
	assert.Equal(t, "response from claude", response)
	mockClaude.AssertCalled(t, "GenerateResponse", mock.Anything, mock.Anything, mock.Anything)
	mockGemini.AssertNotCalled(t, "GenerateResponse")
}

// TestProviderResolution_FallsBackToClaudeWhenEmpty verifies that
// if no provider is configured, it falls back to Claude.
func TestProviderResolution_FallsBackToClaudeWhenEmpty(t *testing.T) {
	// Setup: Create in-memory SQLite
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(database.Schema)
	require.NoError(t, err)

	// Create repositories
	configRepo := repository.NewConfigRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Create mock providers
	mockClaude := &MockLLMService{}
	mockGemini := &MockLLMService{}
	providers := map[ProviderName]LLMService{
		ProviderClaude: mockClaude,
		ProviderGemini: mockGemini,
	}

	// Config has empty provider
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "", // Empty - should fall back to Claude
		},
		Claude: config.ClaudeConfig{
			Model: "claude-3-sonnet",
		},
		Gemini: config.GeminiConfig{
			Model: "gemini-2.0-flash",
		},
	}

	registry := NewProviderRegistry(configRepo, sessionRepo, providers, cfg)

	// Setup mock expectations
	mockClaude.On("GenerateResponse", mock.Anything, mock.Anything, mock.Anything).
		Return("response from claude (fallback)", nil)

	// Call GenerateResponse - should use Claude as fallback
	response, err := registry.GenerateResponse(context.Background(), "prompt", []domain.Message{})
	assert.NoError(t, err)
	assert.Equal(t, "response from claude (fallback)", response)
	mockClaude.AssertCalled(t, "GenerateResponse", mock.Anything, mock.Anything, mock.Anything)
	mockGemini.AssertNotCalled(t, "GenerateResponse")
}

// TestProviderResolution_SessionOverridesPrecedence verifies that
// session-level provider override takes highest precedence.
func TestProviderResolution_SessionOverridesPrecedence(t *testing.T) {
	// Setup: Create in-memory SQLite using the same pattern as other tests
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(database.Schema)
	require.NoError(t, err)

	// Create repositories
	configRepo := repository.NewConfigRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Create a session with provider override
	session, err := sessionRepo.FindOrCreate("discord", "test-channel-123")
	require.NoError(t, err)
	require.NotNil(t, session)
	sessionID := session.ID

	err = sessionRepo.UpdateProviderOverride(sessionID, "gemini")
	require.NoError(t, err)

	// Verify the override was set
	retrievedSession, err := sessionRepo.GetByID(sessionID)
	require.NoError(t, err)
	require.NotNil(t, retrievedSession)
	assert.Equal(t, "gemini", retrievedSession.ProviderOverride)

	// Create mock providers
	mockClaude := &MockLLMService{}
	mockGemini := &MockLLMService{}
	providers := map[ProviderName]LLMService{
		ProviderClaude: mockClaude,
		ProviderGemini: mockGemini,
	}

	// Static config says Claude
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "claude",
		},
		Claude: config.ClaudeConfig{
			Model: "claude-3-sonnet",
		},
		Gemini: config.GeminiConfig{
			Model: "gemini-2.0-flash",
		},
	}

	registry := NewProviderRegistry(configRepo, sessionRepo, providers, cfg)

	// Setup mock expectations
	mockGemini.On("GenerateResponse", mock.Anything, mock.Anything, mock.Anything).
		Return("response from gemini (session override)", nil)

	// Create context with session ID
	ctx := WithSessionID(context.Background(), sessionID)

	// Call GenerateResponse - should use Gemini from session override
	response, err := registry.GenerateResponse(ctx, "prompt", []domain.Message{})
	assert.NoError(t, err)
	assert.Equal(t, "response from gemini (session override)", response)
	mockGemini.AssertCalled(t, "GenerateResponse", mock.Anything, mock.Anything, mock.Anything)
	mockClaude.AssertNotCalled(t, "GenerateResponse")
}

// MockLLMService is a mock implementation of LLMService for testing.
type MockLLMService struct {
	mock.Mock
}

func (m *MockLLMService) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	args := m.Called(ctx, systemPrompt, history)
	return args.String(0), args.Error(1)
}
