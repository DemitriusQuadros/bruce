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

// AllApplicationKeys lists all valid application configuration keys managed via DB and UI.
var AllApplicationKeys = []string{
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
	"llm.background_provider",
	"llm.background_model",
	"app.timezone",
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
	// n8n integration (Spec 29)
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

// handleGetConfig returns configuration stored in SQLite config_entries.
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	repo, ok := r.Context().Value("configRepo").(repository.ConfigRepository)
	if !ok {
		writeError(w, http.StatusInternalServerError, "config repository not initialized")
		return
	}

	// Get all DB values.
	dbEntries, err := repo.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read config")
		return
	}

	// Index DB values by key.
	dbValues := make(map[string]string)
	for _, entry := range dbEntries {
		dbValues[entry.Key] = entry.Value
	}

	result := make([]configEntry, 0, len(AllApplicationKeys))
	for _, key := range AllApplicationKeys {
		dbValue, inDB := dbValues[key]
		if !inDB || dbValue == "" {
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

	writeJSON(w, http.StatusOK, result)
}

// handlePutConfig updates a config entry in the database.
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

	if !contains(AllApplicationKeys, req.Key) {
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

	if appCfg, ok := r.Context().Value("appConfig").(*config.Config); ok && appCfg != nil {
		_ = config.ApplyDatabaseConfig(appCfg, repo)
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// isSensitiveKey returns true if the key contains sensitive data.
func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	if strings.HasSuffix(key, "max_tokens") {
		return false
	}
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
