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
	"bruce/internal/database"
	"bruce/internal/logging"
	"bruce/internal/monitoring"
	"bruce/internal/repository"
	"bruce/internal/tools"
	"bruce/internal/tools/bash"
	calendar "bruce/internal/tools/calendar"
	"bruce/internal/tools/docs"
	"bruce/internal/tools/files"
	gmail "bruce/internal/tools/gmail"
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

	// 3. Wire repositories, LLM providers and registry, and dispatcher.
	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	monitoringRepo := repository.NewMonitoringRepository(db)
	providers := ai.BuildProviders(cfg)
	llmService := ai.NewProviderRegistry(configRepo, sessionRepo, providers, cfg)
	dispatcherRegistry := worker.NewDispatcherRegistry()
	// Connector dispatchers (whatsapp, discord) are registered here when connectors are enabled.

	// 3a. Initialize monitoring (metrics collector and structured logger).
	metricsCollector := monitoring.NewCollector(db)
	structuredLogger := monitoring.NewStructuredLogger(db)

	// 4. Init Asynq client + server.
	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Address}
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

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:    2, // 2 in-flight Claude requests; sufficient for single-user agent
		RetryDelayFunc: asynq.DefaultRetryDelayFunc,
		Queues:         map[string]int{"default": 1},
	})

	proc := worker.NewProcessor(
		sessionRepo, messageRepo, configRepo, monitoringRepo,
		llmService, dispatcherRegistry, metricsCollector, structuredLogger, cfg)

	// Spec 12: wire tool registry (no tools registered yet — Specs 13–31 will add them)
	toolRegistry := tools.NewRegistry(db)

	// Spec 16: Register bash execution tool if enabled.
	if cfg.Tools.Bash.Enabled {
		bashTool := bash.New(cfg.Tools.Bash)
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

	// Spec 17: File I/O tools.
	if cfg.Tools.Files.Enabled {
		if cfg.Tools.Files.HomeDir == "" {
			cfg.Tools.Files.HomeDir = os.Getenv("HOME")
		}
		if err := toolRegistry.Register(files.NewFileReadTool(cfg.Tools.Files)); err != nil {
			log.Fatalf("FATAL: register file_read tool: %v", err)
		}
		if err := toolRegistry.Register(files.NewFileWriteTool(cfg.Tools.Files)); err != nil {
			log.Fatalf("FATAL: register file_write tool: %v", err)
		}
	}

	proc.SetToolRegistry(toolRegistry)

	muxHandler := asynq.NewServeMux()
	muxHandler.HandleFunc(worker.TaskProcessIncomingMessage, proc.HandleProcessIncomingMessageTask)
	muxHandler.HandleFunc("monitoring:flush_metrics", proc.HandleFlushMetricsTask)
	muxHandler.HandleFunc("monitoring:cleanup_logs", proc.HandleCleanupLogsTask)

	go func() {
		if err := asynqServer.Run(muxHandler); err != nil {
			log.Printf("WARNING: asynq server stopped: %v", err)
		}
	}()

	// Schedule periodic tasks
	go func() {
		// Give the asynq server a moment to start
		time.Sleep(2 * time.Second)

		// Enqueue FlushMetrics task to run every 60 seconds
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				_, err := asynqClient.Enqueue(
					asynq.NewTask("monitoring:flush_metrics", []byte("{}")),
					asynq.Queue("default"),
				)
				if err != nil {
					log.Printf("WARNING: failed to enqueue flush metrics task: %v", err)
				}
			}
		}()

		// Enqueue CleanupLogs task to run daily at 2am UTC
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()

			// Calculate time until next 2am UTC
			now := time.Now().UTC()
			nextCleanup := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)
			if nextCleanup.Before(now) {
				nextCleanup = nextCleanup.AddDate(0, 0, 1)
			}

			time.Sleep(nextCleanup.Sub(now))

			for {
				_, err := asynqClient.Enqueue(
					asynq.NewTask("monitoring:cleanup_logs", []byte("{\"retention_days\": 30}")),
					asynq.Queue("default"),
				)
				if err != nil {
					log.Printf("WARNING: failed to enqueue cleanup logs task: %v", err)
				}
				<-ticker.C
			}
		}()
	}()

	// 4. Init HTTP server.
	mon := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitor",
		RedisConnOpt: redisOpt,
	})
	router := bruceapi.NewRouter(startTime, mon, llmService, cfg, googleAuth)

	// Wrap router with middleware to inject dependencies into request context.
	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "messageRepo", messageRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		ctx = context.WithValue(ctx, "monitoringRepo", monitoringRepo)
		ctx = context.WithValue(ctx, "dispatcherRegistry", dispatcherRegistry)
		ctx = context.WithValue(ctx, "metricsCollector", metricsCollector)
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
