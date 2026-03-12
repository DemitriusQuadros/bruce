package ai

import (
	"log"

	"bruce/internal/config"
)

// BuildProviders constructs all configured providers at startup.
// Only providers with valid API keys are registered.
func BuildProviders(cfg *config.Config) map[ProviderName]LLMService {
	providers := make(map[ProviderName]LLMService)

	if cfg.Claude.APIKey != "" {
		providers[ProviderClaude] = NewClaudeProvider(cfg.Claude)
		log.Printf("ai: registered provider %s (model: %s)", ProviderClaude, cfg.Claude.Model)
	}

	if cfg.Gemini.APIKey != "" {
		providers[ProviderGemini] = NewGeminiProvider(cfg.Gemini)
		log.Printf("ai: registered provider %s (model: %s)", ProviderGemini, cfg.Gemini.Model)
	}

	if len(providers) == 0 {
		log.Printf("WARNING: no LLM providers configured. Set claude.api_key or gemini.api_key in config.")
	}

	return providers
}
