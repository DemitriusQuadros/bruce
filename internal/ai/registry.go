package ai

import (
	"context"
	"fmt"
	"sync"

	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/logging"
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
	// Log initialization details for debugging
	availableProviders := make([]string, 0, len(providers))
	for name := range providers {
		availableProviders = append(availableProviders, string(name))
	}
	logging.Infof("ProviderRegistry initialized: available=%v, config_default=%q", availableProviders, cfg.LLM.Provider)

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

// GenerateWithTools resolves the provider for this call and delegates.
func (r *ProviderRegistry) GenerateWithTools(
	ctx context.Context,
	systemPrompt string,
	messages []domain.Message,
	tools []ToolDefinition,
) (*ToolCallResponse, error) {
	provider, err := r.resolveProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GenerateWithTools(ctx, systemPrompt, messages, tools)
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
				logging.Infof("resolveProvider: using session override: %s", session.ProviderOverride)
				return p, nil
			}
		}
	}

	// 2. Check global default from database (runtime override)
	globalDefault, dbErr := r.configRepo.Get("llm.provider")
	if dbErr == nil && globalDefault != "" {
		name := ProviderName(globalDefault)
		r.mu.RLock()
		p, ok := r.providers[name]
		r.mu.RUnlock()
		if ok {
			logging.Infof("resolveProvider: using database config: %s", globalDefault)
			return p, nil
		}
	}

	// 3. Fall back to static config file default
	globalDefault = r.cfg.LLM.Provider
	if globalDefault == "" {
		globalDefault = string(ProviderClaude) // hardcoded fallback
	}

	name := ProviderName(globalDefault)
	r.mu.RLock()
	p, ok := r.providers[name]
	r.mu.RUnlock()

	if !ok {
		logging.Warnf("resolveProvider: unknown provider: %s, falling back to %s", globalDefault, ProviderClaude)
		// Try Claude as last resort
		p, ok := r.providers[ProviderClaude]
		if !ok {
			return nil, fmt.Errorf("%w: %s (no fallback available)", ErrUnknownProvider, name)
		}
		return p, nil
	}

	logging.Infof("resolveProvider: using static config: %s", globalDefault)
	return p, nil
}

// ResolveProviderName returns the name of the provider that would be used for a given context.
func (r *ProviderRegistry) ResolveProviderName(ctx context.Context) string {
	// 1. Check session-level override
	if sessionID, ok := sessionIDFromContext(ctx); ok {
		session, err := r.sessionRepo.GetByID(sessionID)
		if err == nil && session.ProviderOverride != "" {
			name := ProviderName(session.ProviderOverride)
			r.mu.RLock()
			_, exists := r.providers[name]
			r.mu.RUnlock()
			if exists {
				return string(name)
			}
		}
	}

	// 2. Check global default from database
	globalDefault, dbErr := r.configRepo.Get("llm.provider")
	if dbErr == nil && globalDefault != "" {
		name := ProviderName(globalDefault)
		r.mu.RLock()
		_, exists := r.providers[name]
		r.mu.RUnlock()
		if exists {
			return string(name)
		}
	}

	// 3. Fall back to static config
	globalDefault = r.cfg.LLM.Provider
	if globalDefault == "" {
		return string(ProviderClaude)
	}
	return globalDefault
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

	// Always return all providers, showing availability status
	_, claudeAvailable := r.providers[ProviderClaude]
	_, geminiAvailable := r.providers[ProviderGemini]
	_, openaiAvailable := r.providers[ProviderOpenAI]

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
		{
			Name:      string(ProviderOpenAI),
			Available: openaiAvailable,
			Model:     r.cfg.OpenAI.Model,
		},
	}
	return result
}

// GetBackgroundProvider returns the designated background LLM provider (e.g. Gemini 2.0 Flash).
// If not configured, or if the configured background provider is unavailable, it falls back
// to the default provider.
func (r *ProviderRegistry) GetBackgroundProvider() LLMService {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Check background provider configured in LLMConfig
	if r.cfg != nil && r.cfg.LLM.BackgroundProvider != "" {
		bgName := ProviderName(r.cfg.LLM.BackgroundProvider)
		if p, exists := r.providers[bgName]; exists {
			return p
		}
	}

	// 2. Fall back to Gemini if available (fast/cheap default for background loops)
	if p, exists := r.providers[ProviderGemini]; exists {
		return p
	}

	// 3. Fall back to static default or whatever is available
	if r.cfg != nil && r.cfg.LLM.Provider != "" {
		defaultName := ProviderName(r.cfg.LLM.Provider)
		if p, exists := r.providers[defaultName]; exists {
			return p
		}
	}

	for _, p := range r.providers {
		return p
	}
	return nil
}
