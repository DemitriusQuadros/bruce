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

// LLMConfig holds LLM provider selection settings.
type LLMConfig struct {
	Provider string `mapstructure:"provider"` // "claude" | "gemini"
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

// Config is the top-level application configuration.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Redis      RedisConfig      `mapstructure:"redis"`
	SQLite     SQLiteConfig     `mapstructure:"sqlite"`
	Claude     ClaudeConfig     `mapstructure:"claude"`
	Gemini     GeminiConfig     `mapstructure:"gemini"`
	LLM        LLMConfig        `mapstructure:"llm"`
	Connectors ConnectorsConfig `mapstructure:"connectors"`
	UI         UIConfig         `mapstructure:"ui"`
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

	if cfg.Claude.APIKey == "" && cfg.Gemini.APIKey == "" {
		log.Printf("WARNING: no LLM providers configured — set claude.api_key or gemini.api_key in config")
	}

	return &cfg
}
