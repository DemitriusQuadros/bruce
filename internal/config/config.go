// Package config loads application configuration from a YAML file via Viper.
package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Address    string `mapstructure:"address"`
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

// AppConfig holds general application settings such as default timezone.
type AppConfig struct {
	Timezone string `mapstructure:"timezone"`
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

	if err := v.ReadInConfig(); err != nil {
		log.Printf("WARNING: could not read config file %q: %v — using defaults", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Printf("WARNING: could not unmarshal config: %v — using defaults", err)
	}

	if cfg.Claude.APIKey == "" && cfg.Gemini.APIKey == "" && cfg.OpenAI.APIKey == "" {
		log.Printf("WARNING: no LLM providers configured — set claude.api_key, gemini.api_key, or openai.api_key in config")
	}

	return &cfg
}
