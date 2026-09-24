package config_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/repository"
)

func TestLoad_ExampleConfig(t *testing.T) {
	os.Setenv("CONFIG_PATH", "../../config.example.yml")
	defer os.Unsetenv("CONFIG_PATH")

	cfg := config.Load()
	require.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost:6379", cfg.Redis.Address)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, "./data/bruce.db", cfg.SQLite.DSN)
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
	assert.Equal(t, "localhost:6379", cfg.Redis.Address)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, "./data/bruce.db", cfg.SQLite.DSN)
}

func TestApplyDatabaseConfig(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE config_entries (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at DATETIME NOT NULL)`)
	require.NoError(t, err)

	repo := repository.NewConfigRepository(db)
	require.NoError(t, repo.Upsert("claude.api_key", "sk-ant-test"))
	require.NoError(t, repo.Upsert("claude.model", "claude-haiku-4-5"))
	require.NoError(t, repo.Upsert("claude.max_tokens", "2048"))
	require.NoError(t, repo.Upsert("connectors.discord.enabled", "true"))
	require.NoError(t, repo.Upsert("connectors.discord.bot_token", "test-bot-token"))
	require.NoError(t, repo.Upsert("app.timezone", "America/New_York"))
	require.NoError(t, repo.Upsert("llm.background_provider", "gemini"))
	require.NoError(t, repo.Upsert("llm.background_model", "gemini-2.0-flash"))

	cfg := &config.Config{}
	err = config.ApplyDatabaseConfig(cfg, repo)
	require.NoError(t, err)

	assert.Equal(t, "sk-ant-test", cfg.Claude.APIKey)
	assert.Equal(t, "claude-haiku-4-5", cfg.Claude.Model)
	assert.Equal(t, 2048, cfg.Claude.MaxTokens)
	assert.True(t, cfg.Connectors.Discord.Enabled)
	assert.Equal(t, "test-bot-token", cfg.Connectors.Discord.BotToken)
	assert.Equal(t, "America/New_York", cfg.App.Timezone)
	assert.Equal(t, "gemini", cfg.LLM.BackgroundProvider)
	assert.Equal(t, "gemini-2.0-flash", cfg.LLM.BackgroundModel)
}

func TestMigrateLegacyYamlToDB(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE config_entries (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at DATETIME NOT NULL)`)
	require.NoError(t, err)

	repo := repository.NewConfigRepository(db)
	legacyCfg := &config.Config{
		Claude: config.ClaudeConfig{
			APIKey: "sk-legacy-key",
			Model:  "claude-haiku-4-5",
		},
		App: config.AppConfig{
			Timezone: "America/Sao_Paulo",
		},
	}

	migrated, err := config.MigrateLegacyYamlToDB(legacyCfg, repo)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, migrated, 3)

	val, err := repo.Get("claude.api_key")
	require.NoError(t, err)
	assert.Equal(t, "sk-legacy-key", val)

	// Second run should migrate 0 entries since they exist
	migrated2, err := config.MigrateLegacyYamlToDB(legacyCfg, repo)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated2)
}
