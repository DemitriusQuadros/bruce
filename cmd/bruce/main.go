// Command bruce is the main entrypoint for the Bruce AI assistant server.
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

	bruceapi "bruce/internal/api"
	"bruce/internal/config"
	"bruce/internal/database"
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

	// 3. Init Asynq client + server.
	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Address}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 4,
		Queues: map[string]int{
			"default": 1,
		},
	})

	muxHandler := asynq.NewServeMux()
	// Task handlers will be registered here in later specs.
	go func() {
		if err := asynqServer.Run(muxHandler); err != nil {
			log.Printf("WARNING: asynq server stopped: %v", err)
		}
	}()

	// 4. Init HTTP server.
	router := bruceapi.NewRouter(startTime)
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
