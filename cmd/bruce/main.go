// Command bruce is the main entrypoint for the Bruce AI assistant server.

// @title           Bruce API
// @version         1.0
// @description     Personal AI assistant — REST API
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"

	_ "bruce/docs"
	"bruce/internal/ai"
	bruceapi "bruce/internal/api"
	"bruce/internal/auth"
	"bruce/internal/config"
	"bruce/internal/connectors/discord"
	"bruce/internal/connectors/telegram"
	"bruce/internal/connectors/whatsapp"
	"bruce/internal/database"
	"bruce/internal/logging"
	"bruce/internal/repository"
	"bruce/internal/scheduler"
	"bruce/internal/tools"
	"bruce/internal/tools/bash"
	calendar "bruce/internal/tools/calendar"
	"bruce/internal/tools/docs"
	"bruce/internal/tools/files"
	git_local "bruce/internal/tools/git_local"
	"bruce/internal/tools/github"
	gmail "bruce/internal/tools/gmail"
	"bruce/internal/tools/httpclient"
	n8ntool "bruce/internal/tools/n8n"
	"bruce/internal/tools/notion"
	"bruce/internal/tools/proactive"
	"bruce/internal/tools/trello"
	"bruce/internal/worker"
)

func main() {
	startTime := time.Now()

	// 0. Initialize logger based on ENV variable.
	logging.Init()

	// 1. Load config.
	cfg := config.Load()

	// 2. Init SQLite — ensure data directory exists first.
	if err := os.MkdirAll("./data", 0o755); err != nil {
		log.Fatalf("FATAL: create data dir: %v", err)
	}

	db, err := database.NewSQLiteDB(cfg.SQLite.DSN)
	if err != nil {
		log.Fatalf("FATAL: open sqlite: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("FATAL: run migrations: %v", err)
	}

	// 3. Wire repositories and hydrate application config from database.
	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	toolExecutionRepo := repository.NewToolExecutionRepository(db)
	proactiveTaskRepo := repository.NewProactiveTaskRepository(db)
	sessionSummaryRepo := repository.NewSessionSummaryRepository(db)

	// Auto-migrate legacy application YAML settings into database if present.
	if migrated, err := config.MigrateLegacyYamlToDB(cfg, configRepo); err == nil && migrated > 0 {
		log.Printf("INFO: auto-migrated %d application config entries from YAML to database", migrated)
	}

	// Populate application configuration from database (single source of truth).
	if err := config.ApplyDatabaseConfig(cfg, configRepo); err != nil {
		log.Printf("WARNING: failed to apply database config: %v", err)
	}

	// Spec 13: Google OAuth handler (nil if credentials not set).
	var googleAuth *auth.GoogleHandler
	if cfg.Google.OAuthClientID != "" {
		googleAuth = auth.NewGoogleHandler(
			cfg.Google.OAuthClientID,
			cfg.Google.OAuthClientSecret,
			cfg.Google.OAuthRedirectURI,
			db,
		)
	}

	providers := ai.BuildProviders(cfg)
	llmService := ai.NewProviderRegistry(configRepo, sessionRepo, providers, cfg)
	dispatcherRegistry := worker.NewDispatcherRegistry()
	// Connector dispatchers (whatsapp, discord) are registered here when connectors are enabled.

	// 4. Init Asynq client + server.
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	// 4a. Init Discord connector if enabled.
	if cfg.Connectors.Discord.Enabled {
		token, err := resolveDiscordToken(cfg, configRepo)
		if err != nil || token == "" {
			log.Printf("discord: bot_token not configured — connector disabled")
		} else {
			dc, err := discord.New(token, asynqClient)
			if err != nil {
				log.Fatalf("discord init: %v", err)
			}
			if err := dc.Connect(); err != nil {
				log.Fatalf("discord connect: %v", err)
			}
			dispatcherRegistry.Register("discord", dc)
			defer dc.Disconnect()
		}
	}

	// 4b. Init WhatsApp connector if enabled.
	if resolveConnectorEnabled("connectors.whatsapp.enabled", cfg.Connectors.WhatsApp.Enabled, configRepo) {
		if cfg.Connectors.WhatsApp.DeviceStoreDSN == "" {
			cfg.Connectors.WhatsApp.DeviceStoreDSN = "./data/whatsapp.db"
		}
		wa, err := whatsapp.New(cfg, asynqClient)
		if err != nil {
			log.Printf("whatsapp: init failed: %v — connector disabled", err)
		} else if err := wa.Connect(); err != nil {
			log.Printf("whatsapp: connect failed: %v — connector disabled", err)
		} else {
			dispatcherRegistry.Register("whatsapp", wa)
			defer wa.Disconnect()
		}
	}

	// 4c. Init Telegram connector if enabled.
	if resolveConnectorEnabled("connectors.telegram.enabled", cfg.Connectors.Telegram.Enabled, configRepo) {
		token, err := resolveTelegramToken(cfg, configRepo)
		if err != nil || token == "" {
			log.Printf("telegram: bot_token not configured — connector disabled")
		} else {
			tg := telegram.New(token, asynqClient)
			if err := tg.Start(context.Background()); err != nil {
				log.Fatalf("telegram start: %v", err)
			}
			dispatcherRegistry.Register("telegram", tg)
			defer tg.Stop() //nolint:errcheck
		}
	}

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:    2, // 2 in-flight Claude requests; sufficient for single-user agent
		RetryDelayFunc: asynq.DefaultRetryDelayFunc,
		Queues:         map[string]int{"default": 1},
	})

	proc := worker.NewProcessor(
		sessionRepo, messageRepo, configRepo,
		llmService, dispatcherRegistry, cfg)

	// Spec 12: wire tool registry (no tools registered yet — Specs 13–31 will add them)
	toolRegistry := tools.NewRegistry(db)

	// Spec 16: Register bash execution tool if enabled.
	if cfg.Tools.Bash.Enabled {
		bashTool := bash.New(cfg)
		if err := toolRegistry.Register(bashTool); err != nil {
			log.Fatalf("FATAL: register bash tool: %v", err)
		}
	}

	// Spec 14: Gmail tools (requires Google OAuth).
	if cfg.Tools.Gmail.Enabled {
		if googleAuth == nil {
			log.Printf("WARNING: tools.gmail.enabled=true but google OAuth not configured — skipping")
		} else {
			toolRegistry.Register(gmail.NewReadTool(googleAuth))   //nolint:errcheck
			toolRegistry.Register(gmail.NewSearchTool(googleAuth)) //nolint:errcheck
			toolRegistry.Register(gmail.NewSendTool(googleAuth))   //nolint:errcheck
		}
	}

	// Spec 15: Calendar tools (requires Google OAuth).
	if cfg.Tools.Calendar.Enabled {
		if googleAuth == nil {
			log.Printf("WARNING: tools.calendar.enabled=true but google OAuth not configured — skipping")
		} else {
			toolRegistry.Register(calendar.NewReadTool(googleAuth))   //nolint:errcheck
			toolRegistry.Register(calendar.NewCreateTool(googleAuth)) //nolint:errcheck
		}
	}

	// Spec 18: Google Docs tools (requires Google OAuth).
	if cfg.Tools.Docs.Enabled {
		if googleAuth == nil {
			log.Printf("WARNING: tools.docs.enabled=true but google OAuth not configured — skipping")
		} else {
			toolRegistry.Register(docs.NewReadTool(googleAuth))   //nolint:errcheck
			toolRegistry.Register(docs.NewCreateTool(googleAuth)) //nolint:errcheck
			toolRegistry.Register(docs.NewAppendTool(googleAuth)) //nolint:errcheck
		}
	}

	// Spec 19: Notion tools.
	if cfg.Tools.Notion.Enabled {
		if cfg.Tools.Notion.APIToken == "" {
			log.Printf("WARNING: tools.notion.enabled=true but api_token is empty — skipping")
		} else {
			toolRegistry.Register(notion.NewReadTool(cfg.Tools.Notion.APIToken))   //nolint:errcheck
			toolRegistry.Register(notion.NewCreateTool(cfg.Tools.Notion.APIToken)) //nolint:errcheck
			toolRegistry.Register(notion.NewUpdateTool(cfg.Tools.Notion.APIToken)) //nolint:errcheck
		}
	}

	// Spec 20: Trello tools.
	if cfg.Tools.Trello.Enabled {
		if cfg.Tools.Trello.APIKey == "" || cfg.Tools.Trello.APIToken == "" {
			log.Printf("WARNING: tools.trello.enabled=true but api_key or api_token is empty — skipping")
		} else {
			toolRegistry.Register(trello.NewBoardListTool(cfg.Tools.Trello.APIKey, cfg.Tools.Trello.APIToken))  //nolint:errcheck
			toolRegistry.Register(trello.NewCardCreateTool(cfg.Tools.Trello.APIKey, cfg.Tools.Trello.APIToken)) //nolint:errcheck
			toolRegistry.Register(trello.NewCardMoveTool(cfg.Tools.Trello.APIKey, cfg.Tools.Trello.APIToken))   //nolint:errcheck
			toolRegistry.Register(trello.NewCardGetTool(cfg.Tools.Trello.APIKey, cfg.Tools.Trello.APIToken))    //nolint:errcheck
		}
	}
	// Spec 21: GitHub tools.
	if cfg.Tools.Github.Enabled {
		if cfg.Tools.Github.Token == "" {
			log.Printf("WARNING: tools.github.enabled=true but token is empty — skipping")
		} else {
			toolRegistry.Register(github.NewListBranchesTool(cfg.Tools.Github.Token, cfg.Tools.Github.DefaultOwner, cfg.Tools.Github.DefaultRepo)) //nolint:errcheck
			toolRegistry.Register(github.NewListIssuesTool(cfg.Tools.Github.Token, cfg.Tools.Github.DefaultOwner, cfg.Tools.Github.DefaultRepo))   //nolint:errcheck
			toolRegistry.Register(github.NewListPRsTool(cfg.Tools.Github.Token, cfg.Tools.Github.DefaultOwner, cfg.Tools.Github.DefaultRepo))      //nolint:errcheck
			toolRegistry.Register(github.NewCreateIssueTool(cfg.Tools.Github.Token, cfg.Tools.Github.DefaultOwner, cfg.Tools.Github.DefaultRepo))  //nolint:errcheck
		}
	}

	// Spec 22: Git local tools.
	if cfg.Tools.GitLocal.Enabled {
		toolRegistry.Register(git_local.NewStatusTool(cfg)) //nolint:errcheck
		toolRegistry.Register(git_local.NewCommitTool(cfg)) //nolint:errcheck
		toolRegistry.Register(git_local.NewPushTool(cfg))   //nolint:errcheck
		toolRegistry.Register(git_local.NewBranchTool(cfg)) //nolint:errcheck
	}

	// Spec 29: HTTP client tool.
	if cfg.Tools.HTTPClient.Enabled {
		toolRegistry.Register(httpclient.NewHTTPRequestTool(cfg.Tools.HTTPClient)) //nolint:errcheck
	}

	// Spec 29: n8n tools.
	if cfg.Tools.N8n.Enabled {
		if cfg.Tools.N8n.BaseURL == "" {
			log.Printf("WARNING: tools.n8n.enabled=true but base_url is empty — skipping n8n tools")
		} else {
			n8nClient := n8ntool.NewN8nClient(cfg.Tools.N8n)
			toolRegistry.Register(n8ntool.NewWebhookTool(n8nClient, cfg.Tools.N8n))    //nolint:errcheck
			toolRegistry.Register(n8ntool.NewAPITriggerTool(n8nClient, cfg.Tools.N8n)) //nolint:errcheck

			// MCP provider (non-fatal).
			if cfg.Tools.N8n.MCP.Enabled {
				mcpProvider := n8ntool.NewMCPProvider(cfg.Tools.N8n.MCP, toolRegistry)
				mcpProvider.Start(context.Background())
			}
		}
	}

	// Spec 17: File I/O tools.
	if cfg.Tools.Files.Enabled {
		if cfg.Tools.Files.HomeDir == "" {
			cfg.Tools.Files.HomeDir = os.Getenv("HOME")
		}
		if err := toolRegistry.Register(files.NewFileReadTool(cfg)); err != nil {
			log.Fatalf("FATAL: register file_read tool: %v", err)
		}
		if err := toolRegistry.Register(files.NewFileWriteTool(cfg)); err != nil {
			log.Fatalf("FATAL: register file_write tool: %v", err)
		}
	}

	// Spec 32: Proactive Conversational Tools.
	toolRegistry.Register(proactive.NewCreateTool(proactiveTaskRepo, sessionRepo, cfg)) //nolint:errcheck
	toolRegistry.Register(proactive.NewListTool(proactiveTaskRepo))                      //nolint:errcheck
	toolRegistry.Register(proactive.NewToggleTool(proactiveTaskRepo))                    //nolint:errcheck
	toolRegistry.Register(proactive.NewDeleteTool(proactiveTaskRepo))                    //nolint:errcheck

	proc.SetToolRegistry(toolRegistry)
	proc.SetProactiveRepo(proactiveTaskRepo)
	proc.SetProviderRegistry(llmService)
	proc.SetSummaryRepo(sessionSummaryRepo)
	proc.SetAsynqClient(asynqClient)

	muxHandler := asynq.NewServeMux()
	muxHandler.HandleFunc(worker.TaskProcessIncomingMessage, proc.HandleProcessIncomingMessageTask)
	muxHandler.HandleFunc(worker.TaskEvaluateWatch, proc.HandleEvaluateWatchTask)
	muxHandler.HandleFunc(worker.TaskExecuteScheduledReport, proc.HandleExecuteScheduledReportTask)
	muxHandler.HandleFunc(worker.TaskSummarizeSession, proc.HandleSummarizeSessionTask)

	go func() {
		if err := asynqServer.Run(muxHandler); err != nil {
			log.Printf("WARNING: asynq server stopped: %v", err)
		}
	}()

	// Spec 30: Start scheduler poller for due proactive tasks.
	poller := scheduler.NewPoller(proactiveTaskRepo, asynqClient, cfg)
	poller.Start(context.Background())
	defer poller.Stop()

	// 4. Init HTTP server.
	mon := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitor",
		RedisConnOpt: redisOpt,
	})
	router := bruceapi.NewRouter(startTime, mon, llmService, cfg, googleAuth, toolRegistry)

	// Wrap router with middleware to inject dependencies into request context.
	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "messageRepo", messageRepo)
		ctx = context.WithValue(ctx, "sessionSummaryRepo", sessionSummaryRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		ctx = context.WithValue(ctx, "toolExecutionRepo", toolExecutionRepo)
		ctx = context.WithValue(ctx, "proactiveTaskRepo", proactiveTaskRepo)
		ctx = context.WithValue(ctx, "asynqClient", asynqClient)
		ctx = context.WithValue(ctx, "dispatcherRegistry", dispatcherRegistry)
		ctx = context.WithValue(ctx, "appConfig", cfg)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      wrappedRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting HTTP server at %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("FATAL: HTTP server error: %v", err)
		}
	}()

	// 5. Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("WARNING: HTTP server shutdown error: %v", err)
	}

	asynqServer.Shutdown()
	// db.Close() handled by defer above.
}

// resolveConnectorEnabled returns true if the DB value for key is "true",
// falling back to the YAML default.
func resolveConnectorEnabled(key string, yamlVal bool, repo repository.ConfigRepository) bool {
	v, err := repo.Get(key)
	if err != nil || v == "" {
		return yamlVal
	}
	return v == "true"
}

// resolveTelegramToken fetches the Telegram bot token from DB first, then YAML.
func resolveTelegramToken(cfg *config.Config, repo repository.ConfigRepository) (string, error) {
	t, err := repo.Get("connectors.telegram.bot_token")
	if err != nil {
		logging.Debugf("resolveTelegramToken: DB lookup failed (falling back to config): %v", err)
		return cfg.Connectors.Telegram.BotToken, nil
	}
	if t != "" {
		return t, nil
	}
	return cfg.Connectors.Telegram.BotToken, nil
}

// resolveDiscordToken fetches the Discord bot token from the database (if configured)
// or falls back to the config file. Database config takes precedence.
func resolveDiscordToken(cfg *config.Config, repo repository.ConfigRepository) (string, error) {
	t, err := repo.Get("connectors.discord.bot_token")
	if err != nil {
		logging.Debugf("resolveDiscordToken: database lookup failed (falling back to config): %v", err)
		return cfg.Connectors.Discord.BotToken, nil
	}
	if t != "" {
		logging.Debug("resolveDiscordToken: using token from database")
		return t, nil
	}
	logging.Debug("resolveDiscordToken: database value is empty, falling back to config")
	return cfg.Connectors.Discord.BotToken, nil
}
