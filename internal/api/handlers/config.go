package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bruce/internal/config"
	"bruce/internal/monitoring"
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
		start := time.Now()
		w.Header().Set("Content-Type", "application/json")

		collector, ok := r.Context().Value("metricsCollector").(*monitoring.Collector)
		if !ok {
			collector = nil
		}

		switch r.Method {
		case http.MethodGet:
			handleGetConfig(w, r, collector, start)
		case http.MethodPut:
			handlePutConfig(w, r, collector, start)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// handleGetConfig returns the merged config (DB values override YAML defaults).
func handleGetConfig(w http.ResponseWriter, r *http.Request, collector *monitoring.Collector, start time.Time) {
	// Get the database connection from the request context if available.
	// For now, we'll need to pass the config repo through.
	// This is a limitation of the current architecture — see main.go for the dependency injection pattern.
	// We'll store it in the request context in main.go.
	repo, ok := r.Context().Value("configRepo").(repository.ConfigRepository)
	if !ok {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
		}
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
		"openai.api_key":               cfg.OpenAI.APIKey,
		"openai.model":                 cfg.OpenAI.Model,
		"openai.max_tokens":            intToString(cfg.OpenAI.MaxTokens),
		"llm.provider":                 cfg.LLM.Provider,
		"ui.default_system_prompt":     cfg.UI.DefaultSystemPrompt,
		"connectors.whatsapp.enabled":   boolToString(cfg.Connectors.WhatsApp.Enabled),
		"connectors.discord.enabled":    boolToString(cfg.Connectors.Discord.Enabled),
		"connectors.discord.bot_token":  cfg.Connectors.Discord.BotToken,
		"connectors.telegram.enabled":   boolToString(cfg.Connectors.Telegram.Enabled),
		"connectors.telegram.bot_token": cfg.Connectors.Telegram.BotToken,
		// Google OAuth
		"google.oauth_client_id":          cfg.Google.OAuthClientID,
		"google.oauth_client_secret":      cfg.Google.OAuthClientSecret,
		"google.oauth_redirect_uri":       cfg.Google.OAuthRedirectURI,
		// Notion
		"tools.notion.enabled":   boolToString(cfg.Tools.Notion.Enabled),
		"tools.notion.api_token": cfg.Tools.Notion.APIToken,
		// Trello
		"tools.trello.enabled":    boolToString(cfg.Tools.Trello.Enabled),
		"tools.trello.api_key":    cfg.Tools.Trello.APIKey,
		"tools.trello.api_token":  cfg.Tools.Trello.APIToken,
		// GitHub
		"tools.github.enabled":        boolToString(cfg.Tools.Github.Enabled),
		"tools.github.token":          cfg.Tools.Github.Token,
		"tools.github.default_owner":  cfg.Tools.Github.DefaultOwner,
		"tools.github.default_repo":   cfg.Tools.Github.DefaultRepo,
		// Git Local
		"tools.git_local.enabled":          boolToString(cfg.Tools.GitLocal.Enabled),
		"tools.git_local.home_dir":         cfg.Tools.GitLocal.HomeDir,
		"tools.git_local.timeout_seconds":  intToString(cfg.Tools.GitLocal.TimeoutSeconds),
		// Bash
		"tools.bash.enabled": boolToString(cfg.Tools.Bash.Enabled),
		// Google-gated tools
		"tools.gmail.enabled":     boolToString(cfg.Tools.Gmail.Enabled),
		"tools.calendar.enabled":  boolToString(cfg.Tools.Calendar.Enabled),
		"tools.docs.enabled":      boolToString(cfg.Tools.Docs.Enabled),
		// Files
		"tools.files.enabled":        boolToString(cfg.Tools.Files.Enabled),
		"tools.files.home_dir":       cfg.Tools.Files.HomeDir,
		"tools.files.max_file_size":  intToString(cfg.Tools.Files.MaxFileSize),
		// n8n integration (Spec 29)
		"tools.n8n.enabled":                     boolToString(cfg.Tools.N8n.Enabled),
		"tools.n8n.base_url":                    cfg.Tools.N8n.BaseURL,
		"tools.n8n.api_key":                     cfg.Tools.N8n.APIKey,
		"tools.n8n.webhook_timeout_seconds":      intToString(cfg.Tools.N8n.WebhookTimeoutSeconds),
		"tools.n8n.webhook_auth.method":          cfg.Tools.N8n.WebhookAuth.Method,
		"tools.n8n.webhook_auth.username":        cfg.Tools.N8n.WebhookAuth.Username,
		"tools.n8n.webhook_auth.password":        cfg.Tools.N8n.WebhookAuth.Password,
		"tools.n8n.webhook_auth.header_name":     cfg.Tools.N8n.WebhookAuth.HeaderName,
		"tools.n8n.webhook_auth.header_value":    cfg.Tools.N8n.WebhookAuth.HeaderValue,
		"tools.n8n.mcp.enabled":                 boolToString(cfg.Tools.N8n.MCP.Enabled),
		"tools.n8n.mcp.sse_url":                 cfg.Tools.N8n.MCP.SSEURL,
		"tools.n8n.mcp.bearer_token":            cfg.Tools.N8n.MCP.BearerToken,
		"tools.n8n.mcp.tool_name_prefix":        cfg.Tools.N8n.MCP.ToolNamePrefix,
		"tools.n8n.mcp.max_reconnect_attempts":  intToString(cfg.Tools.N8n.MCP.MaxReconnectAttempts),
		// HTTP client (Spec 29)
		"tools.http_client.enabled": boolToString(cfg.Tools.HTTPClient.Enabled),
	}

	// Get all DB values.
	dbEntries, err := repo.GetAll()
	if err != nil {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
		}
		writeError(w, http.StatusInternalServerError, "failed to read config")
		return
	}

	// Index DB values by key.
	dbValues := make(map[string]string)
	for _, entry := range dbEntries {
		dbValues[entry.Key] = entry.Value
	}

	// Only expose DB-stored values. YAML defaults are used by the app internally
	// but are never surfaced to the UI — the user must explicitly save each field.
	result := make([]configEntry, 0, len(knownKeys))
	for key := range knownKeys {
		dbValue, inDB := dbValues[key]
		if !inDB {
			// Not saved yet — return empty so the UI shows "Not configured".
			result = append(result, configEntry{Key: key, Value: ""})
			continue
		}

		value := dbValue
		// Mask sensitive values so the stored secret is never transmitted.
		if isSensitiveKey(key) {
			value = "****"
		}

		result = append(result, configEntry{Key: key, Value: value})
	}

	if collector != nil {
		collector.IncrementCounter("http_requests_total", map[string]string{
			"method":   r.Method,
			"endpoint": "/api/v1/config",
			"status":   "200",
		})
		collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"method":   r.Method,
			"endpoint": "/api/v1/config",
			"status":   "200",
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// handlePutConfig updates a config entry.
func handlePutConfig(w http.ResponseWriter, r *http.Request, collector *monitoring.Collector, start time.Time) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "400",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "400",
			})
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" || req.Value == "" {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "400",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "400",
			})
		}
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
		"openai.api_key",
		"openai.model",
		"openai.max_tokens",
		"llm.provider",
		"ui.default_system_prompt",
		"connectors.whatsapp.enabled",
		"connectors.discord.enabled",
		"connectors.discord.bot_token",
		"connectors.telegram.enabled",
		"connectors.telegram.bot_token",
		// Google OAuth
		"google.oauth_client_id",
		"google.oauth_client_secret",
		"google.oauth_redirect_uri",
		// Notion
		"tools.notion.enabled",
		"tools.notion.api_token",
		// Trello
		"tools.trello.enabled",
		"tools.trello.api_key",
		"tools.trello.api_token",
		// GitHub
		"tools.github.enabled",
		"tools.github.token",
		"tools.github.default_owner",
		"tools.github.default_repo",
		// Git Local
		"tools.git_local.enabled",
		"tools.git_local.home_dir",
		"tools.git_local.timeout_seconds",
		// Bash
		"tools.bash.enabled",
		// Google-gated tools
		"tools.gmail.enabled",
		"tools.calendar.enabled",
		"tools.docs.enabled",
		// Files
		"tools.files.enabled",
		"tools.files.home_dir",
		"tools.files.max_file_size",
		// n8n (Spec 29)
		"tools.n8n.enabled",
		"tools.n8n.base_url",
		"tools.n8n.api_key",
		"tools.n8n.webhook_timeout_seconds",
		"tools.n8n.webhook_auth.method",
		"tools.n8n.webhook_auth.username",
		"tools.n8n.webhook_auth.password",
		"tools.n8n.webhook_auth.header_name",
		"tools.n8n.webhook_auth.header_value",
		"tools.n8n.mcp.enabled",
		"tools.n8n.mcp.sse_url",
		"tools.n8n.mcp.bearer_token",
		"tools.n8n.mcp.tool_name_prefix",
		"tools.n8n.mcp.max_reconnect_attempts",
		// HTTP client (Spec 29)
		"tools.http_client.enabled",
	}
	if !contains(knownKeys, req.Key) {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "422",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "422",
			})
		}
		writeError(w, http.StatusUnprocessableEntity, "unknown config key")
		return
	}

	repo, ok := r.Context().Value("configRepo").(repository.ConfigRepository)
	if !ok {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
		}
		writeError(w, http.StatusInternalServerError, "config repository not initialized")
		return
	}

	if err := repo.Upsert(req.Key, req.Value); err != nil {
		if collector != nil {
			collector.IncrementCounter("http_requests_total", map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
			collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"method":   r.Method,
				"endpoint": "/api/v1/config",
				"status":   "500",
			})
		}
		writeError(w, http.StatusInternalServerError, "failed to update config")
		return
	}

	if collector != nil {
		collector.IncrementCounter("http_requests_total", map[string]string{
			"method":   r.Method,
			"endpoint": "/api/v1/config",
			"status":   "200",
		})
		collector.RecordHistogram("http_request_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"method":   r.Method,
			"endpoint": "/api/v1/config",
			"status":   "200",
		})
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// isSensitiveKey returns true if the key contains sensitive data.
func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "key") || strings.Contains(key, "token") || strings.Contains(key, "secret") || strings.Contains(key, "password")
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
