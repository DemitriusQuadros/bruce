// Package config loads application configuration from a YAML file via Viper.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"bruce/internal/repository"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Address    string `mapstructure:"address"`
	Password   string `mapstructure:"password"`
	DB         int    `mapstructure:"db"`
	MaxRetries int    `mapstructure:"max_retries"`
}

// SQLiteConfig holds SQLite database settings.
type SQLiteConfig struct {
	DSN string `mapstructure:"dsn"`
}

// ClaudeConfig holds Anthropic Claude API settings.
type ClaudeConfig struct {
	APIKey        string `mapstructure:"api_key"`
	Model         string `mapstructure:"model"`
	MaxTokens     int    `mapstructure:"max_tokens"`
	ContextWindow int    `mapstructure:"context_window"`
}

// GeminiConfig holds Google Gemini API settings.
type GeminiConfig struct {
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

// OpenAIConfig holds OpenAI API settings.
type OpenAIConfig struct {
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

// AppConfig holds general application settings such as default timezone and external base URL.
type AppConfig struct {
	Timezone string `mapstructure:"timezone"`
	BaseURL  string `mapstructure:"base_url"` // e.g. "http://bruce.homeserver.local"
}

// WebSearchConfig holds settings for internet search and web page reading tools.
type WebSearchConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"` // "duckduckgo" (default) | "brave" | "tavily" | "searxng"
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

// ArtifactsConfig holds settings for saving and statically serving generated documents/pages.
type ArtifactsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Dir     string `mapstructure:"dir"`      // defaults to "./data/artifacts"
	BaseURL string `mapstructure:"base_url"` // optional domain override, falls back to AppConfig.BaseURL
}

// LLMConfig holds LLM provider selection settings.
type LLMConfig struct {
	Provider           string `mapstructure:"provider"`            // "claude" | "gemini" | "openai"
	BackgroundProvider string `mapstructure:"background_provider"` // "claude" | "gemini" | "openai"
	BackgroundModel    string `mapstructure:"background_model"`
}

// WhatsAppConfig holds WhatsApp connector settings.
type WhatsAppConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	DeviceStoreDSN string `mapstructure:"device_store_dsn"`
}

// DiscordConfig holds Discord connector settings.
type DiscordConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
}

// TelegramConfig holds Telegram Bot connector settings.
type TelegramConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
}

// ConnectorsConfig holds all connector settings.
type ConnectorsConfig struct {
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`
	Discord  DiscordConfig  `mapstructure:"discord"`
	Telegram TelegramConfig `mapstructure:"telegram"`
}

// UIConfig holds web UI settings.
type UIConfig struct {
	DefaultSystemPrompt string `mapstructure:"default_system_prompt"`
}

// GoogleConfig holds Google OAuth 2.0 credentials.
type GoogleConfig struct {
	OAuthClientID     string `mapstructure:"oauth_client_id"`
	OAuthClientSecret string `mapstructure:"oauth_client_secret"`
	OAuthRedirectURI  string `mapstructure:"oauth_redirect_uri"`
}

// BashConfig holds configuration for the bash execution tool.
type BashConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	AllowedCommands []string `mapstructure:"allowed_commands"`
	WorkingDir      string   `mapstructure:"working_dir"`
	MaxOutputBytes  int      `mapstructure:"max_output_bytes"`
	TimeoutSeconds  int      `mapstructure:"timeout_seconds"`
}

// GmailConfig gates the Gmail tool (Spec 14).
type GmailConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// CalendarConfig gates the Google Calendar tool (Spec 15).
type CalendarConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// DocsConfig gates the Google Docs tool (Spec 18).
type DocsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// NotionConfig gates the Notion tool (Spec 19).
type NotionConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	APIToken string `mapstructure:"api_token"`
}

// TrelloConfig gates the Trello tool (Spec 20).
type TrelloConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	APIKey   string `mapstructure:"api_key"`
	APIToken string `mapstructure:"api_token"`
}

// FilesConfig holds configuration for the file I/O tools (Spec 17).
type FilesConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	HomeDir     string `mapstructure:"home_dir"`
	MaxFileSize int    `mapstructure:"max_file_size"` // bytes
}

// GithubConfig gates the GitHub REST API tools (Spec 21).
type GithubConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Token        string `mapstructure:"token"`
	DefaultOwner string `mapstructure:"default_owner"`
	DefaultRepo  string `mapstructure:"default_repo"`
}

// GitLocalConfig gates the local Git CLI tools (Spec 22).
type GitLocalConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	HomeDir        string `mapstructure:"home_dir"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// HTTPClientConfig holds configuration for the generic HTTP client tool (Spec 29).
type HTTPClientConfig struct {
	Enabled              bool `mapstructure:"enabled"`
	MaxTimeoutSeconds    int  `mapstructure:"max_timeout_seconds"`    // default 120
	MaxResponseBytes     int  `mapstructure:"max_response_bytes"`     // default 524288
	FollowRedirects      bool `mapstructure:"follow_redirects"`       // default false
	AllowPrivateNetworks bool `mapstructure:"allow_private_networks"` // default false
}

// WebhookAuthConfig holds auth credentials for n8n webhook calls (Spec 29).
type WebhookAuthConfig struct {
	Method      string `mapstructure:"method"` // none | basic | header
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	HeaderName  string `mapstructure:"header_name"`
	HeaderValue string `mapstructure:"header_value"`
}

// MCPConfig holds MCP-over-SSE settings for the n8n integration (Spec 29).
type MCPConfig struct {
	Enabled              bool   `mapstructure:"enabled"`
	SSEURL               string `mapstructure:"sse_url"`
	BearerToken          string `mapstructure:"bearer_token"`
	ToolNamePrefix       string `mapstructure:"tool_name_prefix"`       // default "n8n_mcp_"
	MaxReconnectAttempts int    `mapstructure:"max_reconnect_attempts"` // default 5
}

// N8nConfig holds all n8n integration settings (Spec 29).
type N8nConfig struct {
	Enabled               bool              `mapstructure:"enabled"`
	BaseURL               string            `mapstructure:"base_url"`
	WebhookTimeoutSeconds int               `mapstructure:"webhook_timeout_seconds"` // default 30
	WebhookAuth           WebhookAuthConfig `mapstructure:"webhook_auth"`
	APIKey                string            `mapstructure:"api_key"`
	MCP                   MCPConfig         `mapstructure:"mcp"`
}

// ToolsConfig gates individual tool integrations.
type ToolsConfig struct {
	Bash       BashConfig       `mapstructure:"bash"`
	Gmail      GmailConfig      `mapstructure:"gmail"`
	Calendar   CalendarConfig   `mapstructure:"calendar"`
	Files      FilesConfig      `mapstructure:"files"`
	Docs       DocsConfig       `mapstructure:"docs"`
	Notion     NotionConfig     `mapstructure:"notion"`
	Trello     TrelloConfig     `mapstructure:"trello"`
	Github     GithubConfig     `mapstructure:"github"`
	GitLocal   GitLocalConfig   `mapstructure:"git_local"`
	HTTPClient HTTPClientConfig `mapstructure:"http_client"`
	N8n        N8nConfig        `mapstructure:"n8n"`
	WebSearch  WebSearchConfig  `mapstructure:"web_search"`
	Artifacts  ArtifactsConfig  `mapstructure:"artifacts"`
}

// Config is the top-level application configuration.
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Server     ServerConfig     `mapstructure:"server"`
	Redis      RedisConfig      `mapstructure:"redis"`
	SQLite     SQLiteConfig     `mapstructure:"sqlite"`
	Claude     ClaudeConfig     `mapstructure:"claude"`
	Gemini     GeminiConfig     `mapstructure:"gemini"`
	OpenAI     OpenAIConfig     `mapstructure:"openai"`
	LLM        LLMConfig        `mapstructure:"llm"`
	Connectors ConnectorsConfig `mapstructure:"connectors"`
	Tools      ToolsConfig      `mapstructure:"tools"`
	UI         UIConfig         `mapstructure:"ui"`
	Google     GoogleConfig     `mapstructure:"google"`
}

// Load reads configuration from CONFIG_PATH env var (defaults to "config.yml").
// It logs a warning if claude.api_key is empty but does not panic.
func Load() *Config {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yml"
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.BindEnv("redis.address", "REDIS_ADDRESS") //nolint:errcheck

	v.SetDefault("tools.web_search.enabled", true)
	v.SetDefault("tools.web_search.provider", "duckduckgo")
	v.SetDefault("tools.artifacts.enabled", true)
	v.SetDefault("tools.artifacts.dir", "./data/artifacts")

	if err := v.ReadInConfig(); err != nil {
		log.Printf("WARNING: could not read config file %q: %v — using defaults", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Printf("WARNING: could not unmarshal config: %v — using defaults", err)
	}

	return &cfg
}

// ApplyDatabaseConfig overlays configuration values from SQLite config_entries table onto cfg.
func ApplyDatabaseConfig(cfg *Config, repo repository.ConfigRepository) error {
	if cfg == nil || repo == nil {
		return nil
	}
	entries, err := repo.GetAll()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		applyEntry(cfg, entry.Key, entry.Value)
	}
	return nil
}

// MigrateLegacyYamlToDB inspects legacyCfg and inserts any configured application keys into
// configRepo if they do not already exist in the database.
func MigrateLegacyYamlToDB(legacyCfg *Config, repo repository.ConfigRepository) (int, error) {
	if legacyCfg == nil || repo == nil {
		return 0, nil
	}
	legacyMap := getLegacyEntries(legacyCfg)
	migrated := 0
	for k, v := range legacyMap {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if _, err := repo.Get(k); err != nil {
			if err := repo.Upsert(k, v); err != nil {
				return migrated, err
			}
			migrated++
		}
	}
	return migrated, nil
}

func getLegacyEntries(c *Config) map[string]string {
	m := make(map[string]string)
	addIfNotEmpty := func(k, v string) {
		if v != "" {
			m[k] = v
		}
	}
	addInt := func(k string, v int) {
		if v > 0 {
			m[k] = strconv.Itoa(v)
		}
	}
	addBool := func(k string, v bool) {
		m[k] = strconv.FormatBool(v)
	}

	addIfNotEmpty("claude.api_key", c.Claude.APIKey)
	addIfNotEmpty("claude.model", c.Claude.Model)
	addInt("claude.max_tokens", c.Claude.MaxTokens)
	addInt("claude.context_window", c.Claude.ContextWindow)

	addIfNotEmpty("gemini.api_key", c.Gemini.APIKey)
	addIfNotEmpty("gemini.model", c.Gemini.Model)
	addInt("gemini.max_tokens", c.Gemini.MaxTokens)

	addIfNotEmpty("openai.api_key", c.OpenAI.APIKey)
	addIfNotEmpty("openai.model", c.OpenAI.Model)
	addInt("openai.max_tokens", c.OpenAI.MaxTokens)

	addIfNotEmpty("llm.provider", c.LLM.Provider)
	addIfNotEmpty("llm.background_provider", c.LLM.BackgroundProvider)
	addIfNotEmpty("llm.background_model", c.LLM.BackgroundModel)

	addIfNotEmpty("app.timezone", c.App.Timezone)
	addIfNotEmpty("app.base_url", c.App.BaseURL)

	addBool("connectors.whatsapp.enabled", c.Connectors.WhatsApp.Enabled)
	addBool("connectors.discord.enabled", c.Connectors.Discord.Enabled)
	addIfNotEmpty("connectors.discord.bot_token", c.Connectors.Discord.BotToken)
	addBool("connectors.telegram.enabled", c.Connectors.Telegram.Enabled)
	addIfNotEmpty("connectors.telegram.bot_token", c.Connectors.Telegram.BotToken)

	addIfNotEmpty("google.oauth_client_id", c.Google.OAuthClientID)
	addIfNotEmpty("google.oauth_client_secret", c.Google.OAuthClientSecret)
	addIfNotEmpty("google.oauth_redirect_uri", c.Google.OAuthRedirectURI)

	addIfNotEmpty("ui.default_system_prompt", c.UI.DefaultSystemPrompt)

	addBool("tools.bash.enabled", c.Tools.Bash.Enabled)
	addIfNotEmpty("tools.bash.working_dir", c.Tools.Bash.WorkingDir)
	addInt("tools.bash.timeout_seconds", c.Tools.Bash.TimeoutSeconds)
	addInt("tools.bash.max_output_bytes", c.Tools.Bash.MaxOutputBytes)
	addBool("tools.gmail.enabled", c.Tools.Gmail.Enabled)
	addBool("tools.calendar.enabled", c.Tools.Calendar.Enabled)
	addBool("tools.docs.enabled", c.Tools.Docs.Enabled)

	addBool("tools.notion.enabled", c.Tools.Notion.Enabled)
	addIfNotEmpty("tools.notion.api_token", c.Tools.Notion.APIToken)

	addBool("tools.trello.enabled", c.Tools.Trello.Enabled)
	addIfNotEmpty("tools.trello.api_key", c.Tools.Trello.APIKey)
	addIfNotEmpty("tools.trello.api_token", c.Tools.Trello.APIToken)

	addBool("tools.github.enabled", c.Tools.Github.Enabled)
	addIfNotEmpty("tools.github.token", c.Tools.Github.Token)
	addIfNotEmpty("tools.github.default_owner", c.Tools.Github.DefaultOwner)
	addIfNotEmpty("tools.github.default_repo", c.Tools.Github.DefaultRepo)

	addBool("tools.git_local.enabled", c.Tools.GitLocal.Enabled)
	addIfNotEmpty("tools.git_local.home_dir", c.Tools.GitLocal.HomeDir)
	addInt("tools.git_local.timeout_seconds", c.Tools.GitLocal.TimeoutSeconds)

	addBool("tools.files.enabled", c.Tools.Files.Enabled)
	addIfNotEmpty("tools.files.home_dir", c.Tools.Files.HomeDir)
	addInt("tools.files.max_file_size", c.Tools.Files.MaxFileSize)

	addBool("tools.http_client.enabled", c.Tools.HTTPClient.Enabled)

	addBool("tools.web_search.enabled", c.Tools.WebSearch.Enabled)
	addIfNotEmpty("tools.web_search.provider", c.Tools.WebSearch.Provider)
	addIfNotEmpty("tools.web_search.api_key", c.Tools.WebSearch.APIKey)
	addIfNotEmpty("tools.web_search.base_url", c.Tools.WebSearch.BaseURL)

	addBool("tools.artifacts.enabled", c.Tools.Artifacts.Enabled)
	addIfNotEmpty("tools.artifacts.dir", c.Tools.Artifacts.Dir)
	addIfNotEmpty("tools.artifacts.base_url", c.Tools.Artifacts.BaseURL)

	addBool("tools.n8n.enabled", c.Tools.N8n.Enabled)
	addIfNotEmpty("tools.n8n.base_url", c.Tools.N8n.BaseURL)
	addIfNotEmpty("tools.n8n.api_key", c.Tools.N8n.APIKey)
	addInt("tools.n8n.webhook_timeout_seconds", c.Tools.N8n.WebhookTimeoutSeconds)
	addIfNotEmpty("tools.n8n.webhook_auth.method", c.Tools.N8n.WebhookAuth.Method)
	addIfNotEmpty("tools.n8n.webhook_auth.username", c.Tools.N8n.WebhookAuth.Username)
	addIfNotEmpty("tools.n8n.webhook_auth.password", c.Tools.N8n.WebhookAuth.Password)
	addIfNotEmpty("tools.n8n.webhook_auth.header_name", c.Tools.N8n.WebhookAuth.HeaderName)
	addIfNotEmpty("tools.n8n.webhook_auth.header_value", c.Tools.N8n.WebhookAuth.HeaderValue)
	addBool("tools.n8n.mcp.enabled", c.Tools.N8n.MCP.Enabled)
	addIfNotEmpty("tools.n8n.mcp.sse_url", c.Tools.N8n.MCP.SSEURL)
	addIfNotEmpty("tools.n8n.mcp.bearer_token", c.Tools.N8n.MCP.BearerToken)
	addIfNotEmpty("tools.n8n.mcp.tool_name_prefix", c.Tools.N8n.MCP.ToolNamePrefix)
	addInt("tools.n8n.mcp.max_reconnect_attempts", c.Tools.N8n.MCP.MaxReconnectAttempts)

	return m
}

func applyEntry(cfg *Config, key, val string) {
	switch key {
	case "claude.api_key":
		cfg.Claude.APIKey = val
	case "claude.model":
		cfg.Claude.Model = val
	case "claude.max_tokens":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Claude.MaxTokens = n
		}
	case "claude.context_window":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Claude.ContextWindow = n
		}
	case "gemini.api_key":
		cfg.Gemini.APIKey = val
	case "gemini.model":
		cfg.Gemini.Model = val
	case "gemini.max_tokens":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Gemini.MaxTokens = n
		}
	case "openai.api_key":
		cfg.OpenAI.APIKey = val
	case "openai.model":
		cfg.OpenAI.Model = val
	case "openai.max_tokens":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.OpenAI.MaxTokens = n
		}
	case "llm.provider":
		cfg.LLM.Provider = val
	case "llm.background_provider":
		cfg.LLM.BackgroundProvider = val
	case "llm.background_model":
		cfg.LLM.BackgroundModel = val
	case "app.timezone":
		cfg.App.Timezone = val
	case "app.base_url":
		cfg.App.BaseURL = val
	case "connectors.discord.enabled":
		cfg.Connectors.Discord.Enabled = (val == "true")
	case "connectors.discord.bot_token":
		cfg.Connectors.Discord.BotToken = val
	case "connectors.whatsapp.enabled":
		cfg.Connectors.WhatsApp.Enabled = (val == "true")
	case "connectors.telegram.enabled":
		cfg.Connectors.Telegram.Enabled = (val == "true")
	case "connectors.telegram.bot_token":
		cfg.Connectors.Telegram.BotToken = val
	case "google.oauth_client_id":
		cfg.Google.OAuthClientID = val
	case "google.oauth_client_secret":
		cfg.Google.OAuthClientSecret = val
	case "google.oauth_redirect_uri":
		cfg.Google.OAuthRedirectURI = val
	case "ui.default_system_prompt":
		cfg.UI.DefaultSystemPrompt = val
	case "tools.bash.enabled":
		cfg.Tools.Bash.Enabled = (val == "true")
	case "tools.bash.working_dir":
		cfg.Tools.Bash.WorkingDir = val
	case "tools.bash.timeout_seconds":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.Tools.Bash.TimeoutSeconds = n
		}
	case "tools.bash.max_output_bytes":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.Tools.Bash.MaxOutputBytes = n
		}
	case "tools.gmail.enabled":
		cfg.Tools.Gmail.Enabled = (val == "true")
	case "tools.calendar.enabled":
		cfg.Tools.Calendar.Enabled = (val == "true")
	case "tools.docs.enabled":
		cfg.Tools.Docs.Enabled = (val == "true")
	case "tools.notion.enabled":
		cfg.Tools.Notion.Enabled = (val == "true")
	case "tools.notion.api_token":
		cfg.Tools.Notion.APIToken = val
	case "tools.trello.enabled":
		cfg.Tools.Trello.Enabled = (val == "true")
	case "tools.trello.api_key":
		cfg.Tools.Trello.APIKey = val
	case "tools.trello.api_token":
		cfg.Tools.Trello.APIToken = val
	case "tools.github.enabled":
		cfg.Tools.Github.Enabled = (val == "true")
	case "tools.github.token":
		cfg.Tools.Github.Token = val
	case "tools.github.default_owner":
		cfg.Tools.Github.DefaultOwner = val
	case "tools.github.default_repo":
		cfg.Tools.Github.DefaultRepo = val
	case "tools.git_local.enabled":
		cfg.Tools.GitLocal.Enabled = (val == "true")
	case "tools.git_local.home_dir":
		cfg.Tools.GitLocal.HomeDir = val
	case "tools.git_local.timeout_seconds":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Tools.GitLocal.TimeoutSeconds = n
		}
	case "tools.files.enabled":
		cfg.Tools.Files.Enabled = (val == "true")
	case "tools.files.home_dir":
		cfg.Tools.Files.HomeDir = val
	case "tools.files.max_file_size":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Tools.Files.MaxFileSize = n
		}
	case "tools.http_client.enabled":
		cfg.Tools.HTTPClient.Enabled = (val == "true")
	case "tools.web_search.enabled":
		cfg.Tools.WebSearch.Enabled = (val == "true")
	case "tools.web_search.provider":
		cfg.Tools.WebSearch.Provider = val
	case "tools.web_search.api_key":
		cfg.Tools.WebSearch.APIKey = val
	case "tools.web_search.base_url":
		cfg.Tools.WebSearch.BaseURL = val
	case "tools.artifacts.enabled":
		cfg.Tools.Artifacts.Enabled = (val == "true")
	case "tools.artifacts.dir":
		cfg.Tools.Artifacts.Dir = val
	case "tools.artifacts.base_url":
		cfg.Tools.Artifacts.BaseURL = val
	case "tools.n8n.enabled":
		cfg.Tools.N8n.Enabled = (val == "true")
	case "tools.n8n.base_url":
		cfg.Tools.N8n.BaseURL = val
	case "tools.n8n.api_key":
		cfg.Tools.N8n.APIKey = val
	case "tools.n8n.webhook_timeout_seconds":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Tools.N8n.WebhookTimeoutSeconds = n
		}
	case "tools.n8n.webhook_auth.method":
		cfg.Tools.N8n.WebhookAuth.Method = val
	case "tools.n8n.webhook_auth.username":
		cfg.Tools.N8n.WebhookAuth.Username = val
	case "tools.n8n.webhook_auth.password":
		cfg.Tools.N8n.WebhookAuth.Password = val
	case "tools.n8n.webhook_auth.header_name":
		cfg.Tools.N8n.WebhookAuth.HeaderName = val
	case "tools.n8n.webhook_auth.header_value":
		cfg.Tools.N8n.WebhookAuth.HeaderValue = val
	case "tools.n8n.mcp.enabled":
		cfg.Tools.N8n.MCP.Enabled = (val == "true")
	case "tools.n8n.mcp.sse_url":
		cfg.Tools.N8n.MCP.SSEURL = val
	case "tools.n8n.mcp.bearer_token":
		cfg.Tools.N8n.MCP.BearerToken = val
	case "tools.n8n.mcp.tool_name_prefix":
		cfg.Tools.N8n.MCP.ToolNamePrefix = val
	case "tools.n8n.mcp.max_reconnect_attempts":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Tools.N8n.MCP.MaxReconnectAttempts = n
		}
	}
}
