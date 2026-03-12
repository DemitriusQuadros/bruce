package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bruce/internal/config"
	"bruce/internal/repository"
)

// configEntry represents a config key-value pair returned by the API.
type configEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ConfigHandler godoc
// @Summary      Get / update runtime config
// @Description  Returns or updates the runtime configuration
// @Tags         config
// @Produce      json
// @Success      200  {object}  []configEntry
// @Router       /api/v1/config [get]
// @Router       /api/v1/config [put]
func ConfigHandler() http.HandlerFunc {
	// Initialize the config repository (in a real app, this would be injected).
	// For now, we create it on each request — not ideal, but acceptable for Phase 1.
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			handleGetConfig(w, r)
		case http.MethodPut:
			handlePutConfig(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// handleGetConfig returns the merged config (DB values override YAML defaults).
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Get the database connection from the request context if available.
	// For now, we'll need to pass the config repo through.
	// This is a limitation of the current architecture — see main.go for the dependency injection pattern.
	// We'll store it in the request context in main.go.
	repo, ok := r.Context().Value("configRepo").(repository.ConfigRepository)
	if !ok {
		writeError(w, http.StatusInternalServerError, "config repository not initialized")
		return
	}

	cfg := config.Load()

	// Known config keys with their YAML defaults.
	knownKeys := map[string]string{
		"claude.api_key":               cfg.Claude.APIKey,
		"claude.model":                 cfg.Claude.Model,
		"claude.max_tokens":            intToString(cfg.Claude.MaxTokens),
		"claude.context_window":        intToString(cfg.Claude.ContextWindow),
		"gemini.api_key":               cfg.Gemini.APIKey,
		"gemini.model":                 cfg.Gemini.Model,
		"gemini.max_tokens":            intToString(cfg.Gemini.MaxTokens),
		"llm.provider":                 cfg.LLM.Provider,
		"ui.default_system_prompt":     cfg.UI.DefaultSystemPrompt,
		"connectors.whatsapp.enabled":  boolToString(cfg.Connectors.WhatsApp.Enabled),
		"connectors.discord.enabled":   boolToString(cfg.Connectors.Discord.Enabled),
		"connectors.discord.bot_token": cfg.Connectors.Discord.BotToken,
	}

	// Get all DB values.
	dbEntries, err := repo.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read config")
		return
	}

	// Build map of DB values (DB wins over YAML).
	dbValues := make(map[string]string)
	for _, entry := range dbEntries {
		dbValues[entry.Key] = entry.Value
	}

	// Merge: DB overrides YAML defaults.
	result := make([]configEntry, 0, len(knownKeys))
	for key, yamlValue := range knownKeys {
		value := yamlValue
		if dbValue, exists := dbValues[key]; exists {
			value = dbValue
		}

		// Mask sensitive values.
		if isSensitiveKey(key) {
			value = "****"
		}

		result = append(result, configEntry{Key: key, Value: value})
	}

	writeJSON(w, http.StatusOK, result)
}

// handlePutConfig updates a config entry.
func handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" || req.Value == "" {
		writeError(w, http.StatusBadRequest, "missing key or value")
		return
	}

	// Validate that the key is known.
	knownKeys := []string{
		"claude.api_key",
		"claude.model",
		"claude.max_tokens",
		"claude.context_window",
		"gemini.api_key",
		"gemini.model",
		"gemini.max_tokens",
		"llm.provider",
		"ui.default_system_prompt",
		"connectors.whatsapp.enabled",
		"connectors.discord.enabled",
		"connectors.discord.bot_token",
	}
	if !contains(knownKeys, req.Key) {
		writeError(w, http.StatusUnprocessableEntity, "unknown config key")
		return
	}

	repo, ok := r.Context().Value("configRepo").(repository.ConfigRepository)
	if !ok {
		writeError(w, http.StatusInternalServerError, "config repository not initialized")
		return
	}

	if err := repo.Upsert(req.Key, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// isSensitiveKey returns true if the key contains sensitive data.
func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "key") || strings.Contains(key, "token") || strings.Contains(key, "secret")
}

// contains checks if a string is in a slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// boolToString converts a bool to "true" or "false".
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// intToString converts an int to its string representation.
func intToString(i int) string {
	return strconv.Itoa(i)
}
