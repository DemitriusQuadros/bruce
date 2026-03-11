// Command bruce is the main entrypoint for the Bruce AI assistant server.

// @title           Bruce API
// @version         1.0
// @description     Personal AI assistant — REST API
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"database/sql"
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
	"bruce/internal/config"
	"bruce/internal/connectors/discord"
	"bruce/internal/database"
	"bruce/internal/repository"
	"bruce/internal/worker"
)

func main() {
	startTime := time.Now()

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

	if err := runSchema(db); err != nil {
		log.Fatalf("FATAL: run schema: %v", err)
	}

	// 3. Wire repositories, LLM service, and dispatcher.
	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	llmService := ai.NewClaudeService(cfg)
	dispatcherRegistry := worker.NewDispatcherRegistry()
	// Connector dispatchers (whatsapp, discord) are registered here when connectors are enabled.

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

	proc := worker.NewProcessor(sessionRepo, messageRepo, configRepo, llmService, dispatcherRegistry, cfg)
	muxHandler := asynq.NewServeMux()
	muxHandler.HandleFunc(worker.TaskProcessIncomingMessage, proc.HandleProcessIncomingMessageTask)

	go func() {
		if err := asynqServer.Run(muxHandler); err != nil {
			log.Printf("WARNING: asynq server stopped: %v", err)
		}
	}()

	// 4. Init HTTP server.
	mon := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitor",
		RedisConnOpt: redisOpt,
	})
	router := bruceapi.NewRouter(startTime, mon)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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

// runSchema executes the embedded DDL against the open SQLite database.
func runSchema(db *sql.DB) error {
	if _, err := db.Exec(database.Schema); err != nil {
		return fmt.Errorf("exec schema.sql: %w", err)
	}
	return nil
}

// resolveDiscordToken fetches the Discord bot token from the database (if configured)
// or falls back to the config file. Database config takes precedence.
func resolveDiscordToken(cfg *config.Config, repo repository.ConfigRepository) (string, error) {
	if t, err := repo.Get("connectors.discord.bot_token"); err == nil && t != "" {
		return t, nil
	}
	return cfg.Connectors.Discord.BotToken, nil
}
