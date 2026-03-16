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

// LLMConfig holds LLM provider selection settings.
type LLMConfig struct {
	Provider string `mapstructure:"provider"` // "claude" | "gemini" | "openai"
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

// ConnectorsConfig holds all connector settings.
type ConnectorsConfig struct {
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`
	Discord  DiscordConfig  `mapstructure:"discord"`
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

// FilesConfig holds configuration for the file I/O tools (Spec 17).
type FilesConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	HomeDir     string `mapstructure:"home_dir"`
	MaxFileSize int    `mapstructure:"max_file_size"` // bytes
}

// ToolsConfig gates individual tool integrations.
type ToolsConfig struct {
	Bash     BashConfig     `mapstructure:"bash"`
	Gmail    GmailConfig    `mapstructure:"gmail"`
	Calendar CalendarConfig `mapstructure:"calendar"`
	Files    FilesConfig    `mapstructure:"files"`
	Docs     DocsConfig     `mapstructure:"docs"`
	Notion   NotionConfig   `mapstructure:"notion"`
}

// Config is the top-level application configuration.
type Config struct {
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
