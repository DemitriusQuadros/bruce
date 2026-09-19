package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
)

func TestLoad_ExampleConfig(t *testing.T) {
	os.Setenv("CONFIG_PATH", "../../config.example.yml")
	defer os.Unsetenv("CONFIG_PATH")

	cfg := config.Load()
	require.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "America/Sao_Paulo", cfg.App.Timezone)
	assert.Equal(t, "gemini", cfg.LLM.BackgroundProvider)
	assert.Equal(t, "gemini-2.0-flash", cfg.LLM.BackgroundModel)
	assert.Equal(t, 1, cfg.Redis.DB)
}

func TestLoad_UserConfig(t *testing.T) {
	if _, err := os.Stat("../../config.yml"); os.IsNotExist(err) {
		t.Skip("config.yml not present")
	}

	os.Setenv("CONFIG_PATH", "../../config.yml")
	defer os.Unsetenv("CONFIG_PATH")

	cfg := config.Load()
	require.NotNil(t, cfg)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "America/Sao_Paulo", cfg.App.Timezone)
	assert.Equal(t, "claude", cfg.LLM.Provider)
	assert.Equal(t, "gemini", cfg.LLM.BackgroundProvider)
	assert.Equal(t, "gemini-2.0-flash", cfg.LLM.BackgroundModel)
	assert.NotEmpty(t, cfg.Claude.APIKey)
	assert.NotEmpty(t, cfg.Gemini.APIKey)
	assert.NotEmpty(t, cfg.Connectors.Discord.BotToken)
	assert.NotEmpty(t, cfg.Tools.Notion.APIToken)
	assert.NotEmpty(t, cfg.Tools.Trello.APIKey)
}
