package ai

import (
	"context"
	"fmt"
	"sync"

	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
)

// ProviderRegistry implements LLMService and routes calls to the appropriate provider.
// It holds all initialized providers and resolves which one to use per call.
type ProviderRegistry struct {
	providers   map[ProviderName]LLMService
	configRepo  repository.ConfigRepository
	sessionRepo repository.SessionRepository
	cfg         *config.Config
	mu          sync.RWMutex
}

// NewProviderRegistry returns a new ProviderRegistry.
func NewProviderRegistry(
	configRepo repository.ConfigRepository,
	sessionRepo repository.SessionRepository,
	providers map[ProviderName]LLMService,
	cfg *config.Config,
) *ProviderRegistry {
	return &ProviderRegistry{
		providers:   providers,
		configRepo:  configRepo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

// GenerateResponse resolves the provider for this call and delegates.
// sessionID is passed via context — see WithSessionID below.
func (r *ProviderRegistry) GenerateResponse(
	ctx context.Context,
	systemPrompt string,
	history []domain.Message,
) (string, error) {
	provider, err := r.resolveProvider(ctx)
	if err != nil {
		return "", err
	}
	return provider.GenerateResponse(ctx, systemPrompt, history)
}

func (r *ProviderRegistry) resolveProvider(ctx context.Context) (LLMService, error) {
	// 1. Check session-level override (set via web UI per session)
	if sessionID, ok := sessionIDFromContext(ctx); ok {
		session, err := r.sessionRepo.GetByID(sessionID)
		if err == nil && session.ProviderOverride != "" {
			name := ProviderName(session.ProviderOverride)
			r.mu.RLock()
			p, ok := r.providers[name]
			r.mu.RUnlock()
			if ok {
				return p, nil
			}
		}
	}

	// 2. Fall back to global config default
	globalDefault, _ := r.configRepo.Get("llm.provider") // e.g. "claude"
	if globalDefault == "" {
		globalDefault = string(ProviderClaude) // hardcoded fallback
	}

	name := ProviderName(globalDefault)
	r.mu.RLock()
	p, ok := r.providers[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}
	return p, nil
}

// ListProviders returns information about all configured providers.
// Used by the API to populate provider selectors.
type ProviderInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Model     string `json:"model"`
}

func (r *ProviderRegistry) ListProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Always return both providers, showing availability status
	_, claudeAvailable := r.providers[ProviderClaude]
	_, geminiAvailable := r.providers[ProviderGemini]

	result := []ProviderInfo{
		{
			Name:      string(ProviderClaude),
			Available: claudeAvailable,
			Model:     r.cfg.Claude.Model,
		},
		{
			Name:      string(ProviderGemini),
			Available: geminiAvailable,
			Model:     r.cfg.Gemini.Model,
		},
	}
	return result
}
