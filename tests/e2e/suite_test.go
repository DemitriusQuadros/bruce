package e2e_test

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cucumber/godog"
	"github.com/hibiken/asynq"
	_ "github.com/mattn/go-sqlite3"

	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"bruce/internal/ai"
	"bruce/internal/api"
	"bruce/internal/config"
	"bruce/internal/database"
	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/tools"
	"bruce/internal/tools/proactive"
	"bruce/internal/worker"
)

// godogTags holds the value of the -godog.tags flag passed to the test binary.
// It is populated by TestMain before any test runs.
var godogTags = flag.String("godog.tags", "", "Godog tag expression to filter scenarios (e.g. @smoke)")

// TestMain registers godog flags and runs the test suite.
// It must call flag.Parse() before m.Run() so that -godog.tags (and any
// other flags registered above) are parsed before TestE2E reads them.
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// stubLLMService is an in-process LLM stub used for E2E testing.
// It always returns a fixed response so that worker pipeline scenarios
// exercise the full code path without a real Claude API key.
type stubLLMService struct{}

func (s *stubLLMService) GenerateResponse(_ context.Context, _ string, _ []domain.Message) (string, error) {
	return "Hello from test stub", nil
}

func (s *stubLLMService) GenerateWithTools(_ context.Context, _ string, _ []domain.Message, _ []ai.ToolDefinition) (*ai.ToolCallResponse, error) {
	return &ai.ToolCallResponse{Text: "Hello from test stub", Complete: true}, nil
}

// compile-time check: stubLLMService satisfies ai.LLMService.
var _ ai.LLMService = (*stubLLMService)(nil)

// TestE2E is the godog suite entry-point.
// It starts an in-process Asynq worker backed by a stub LLM so that all
// scenarios — including the previously @requires-claude ones — run without
// a real API key.
//
// Run with:
//
//	CGO_ENABLED=1 go test -v ./tests/e2e/ -args -godog.format=pretty
//
// Run only smoke tests:
//
//	CGO_ENABLED=1 go test -v ./tests/e2e/ -args -godog.tags="@smoke"
func TestE2E(t *testing.T) {
	cfg := loadTestConfig()

	db := openTestDB(t, cfg)
	defer db.Close()

	redisAddr := cfg.Redis.Address
	if redisAddr == "" || redisAddr == "localhost:6379" {
		redisAddr = "127.0.0.1:6379"
	}
	if override := os.Getenv("REDIS_ADDRESS"); override != "" {
		redisAddr = override
	}

	// Purge stale tasks from Redis queue to guarantee scenario isolation
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr})
	_, _ = inspector.DeleteAllPendingTasks("default")
	_, _ = inspector.DeleteAllScheduledTasks("default")
	_, _ = inspector.DeleteAllRetryTasks("default")
	_, _ = inspector.DeleteAllArchivedTasks("default")
	inspector.Close()

	asynqClient := newAsynqClient(redisAddr)
	defer asynqClient.Close()

	baseURL := os.Getenv("BRUCE_BASE_URL")
	var testHTTPSrv *httptest.Server
	if baseURL == "" {
		testPort := 8081
		if cfg.Server.Port > 0 {
			testPort = cfg.Server.Port
		}
		targetURL := fmt.Sprintf("http://localhost:%d", testPort)
		resp, err := http.Get(targetURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			baseURL = targetURL
		} else {
			testHTTPSrv = startStubAPIServer(t, db, cfg, asynqClient)
			defer testHTTPSrv.Close()
			baseURL = testHTTPSrv.URL
		}
	}

	// Start an in-process Asynq worker with the stub LLM so that all worker
	// pipeline scenarios exercise the full message→LLM→persist flow without
	// requiring a real Claude API key.
	asynqSrv := startStubWorker(t, db, cfg, redisAddr)
	defer asynqSrv.Shutdown()
	time.Sleep(200 * time.Millisecond)

	tc := NewTestContext(baseURL, db, asynqClient, redisAddr)

	// features/ is resolved relative to this file's directory so it works
	// regardless of the working directory when go test is invoked.
	_, thisFile, _, _ := runtime.Caller(0)
	featuresDir := filepath.Join(filepath.Dir(thisFile), "features")

	opts := &godog.Options{
		Format:   "pretty",
		Paths:    []string{featuresDir},
		TestingT: t,
	}
	if godogTags != nil && *godogTags != "" {
		opts.Tags = *godogTags
	}

	suite := godog.TestSuite{
		Name: "bruce-e2e",
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			RegisterCommonSteps(ctx, tc)
			RegisterWorkerSteps(ctx, tc)
			RegisterDiscordSteps(ctx, tc)
			RegisterWebChatSteps(ctx, tc)
			RegisterProactiveSteps(ctx, tc)
		},
		Options: opts,
	}

	if suite.Run() != 0 {
		t.Fatal("E2E suite reported failures")
	}
}

// startStubAPIServer spins up an in-process HTTP server wrapping the full Bruce API router.
func startStubAPIServer(t *testing.T, db *sql.DB, cfg *config.Config, asynqClient *asynq.Client) *httptest.Server {
	t.Helper()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	toolExecutionRepo := repository.NewToolExecutionRepository(db)
	proactiveTaskRepo := repository.NewProactiveTaskRepository(db)
	dispatcherRegistry := worker.NewDispatcherRegistry()

	toolRegistry := tools.NewRegistry(db)
	toolRegistry.Register(proactive.NewCreateTool(proactiveTaskRepo, sessionRepo, cfg)) //nolint:errcheck
	toolRegistry.Register(proactive.NewListTool(proactiveTaskRepo))                      //nolint:errcheck
	toolRegistry.Register(proactive.NewToggleTool(proactiveTaskRepo))                    //nolint:errcheck
	toolRegistry.Register(proactive.NewDeleteTool(proactiveTaskRepo))                    //nolint:errcheck

	providers := map[ai.ProviderName]ai.LLMService{
		ai.ProviderClaude: &stubLLMService{},
		ai.ProviderGemini: &stubLLMService{},
		ai.ProviderOpenAI: &stubLLMService{},
	}
	llmService := ai.NewProviderRegistry(configRepo, sessionRepo, providers, cfg)

	router := api.NewRouter(time.Now(), nil, llmService, cfg, nil, toolRegistry)

	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "messageRepo", messageRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		ctx = context.WithValue(ctx, "proactiveTaskRepo", proactiveTaskRepo)
		ctx = context.WithValue(ctx, "toolExecutionRepo", toolExecutionRepo)
		ctx = context.WithValue(ctx, "toolRegistry", toolRegistry)
		ctx = context.WithValue(ctx, "dispatcherRegistry", dispatcherRegistry)
		ctx = context.WithValue(ctx, "asynqClient", asynqClient)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	return httptest.NewServer(wrapped)
}

// startStubWorker wires up repositories, a stub LLM, and an Asynq server in-process.
// The server runs in a background goroutine and is shut down via the returned *asynq.Server.
func startStubWorker(t *testing.T, db *sql.DB, cfg *config.Config, redisAddr string) *asynq.Server {
	t.Helper()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	proactiveTaskRepo := repository.NewProactiveTaskRepository(db)
	dispatcherRegistry := worker.NewDispatcherRegistry()
	// No real connector dispatchers are registered — dispatch errors are logged
	// but do not fail the task (ADR-004), so this is safe for E2E testing.

	toolRegistry := tools.NewRegistry(db)
	toolRegistry.Register(proactive.NewCreateTool(proactiveTaskRepo, sessionRepo, cfg)) //nolint:errcheck
	toolRegistry.Register(proactive.NewListTool(proactiveTaskRepo))                      //nolint:errcheck
	toolRegistry.Register(proactive.NewToggleTool(proactiveTaskRepo))                    //nolint:errcheck
	toolRegistry.Register(proactive.NewDeleteTool(proactiveTaskRepo))                    //nolint:errcheck

	proc := worker.NewProcessor(sessionRepo, messageRepo, configRepo,
		&stubLLMService{}, dispatcherRegistry, cfg)
	proc.SetToolRegistry(toolRegistry)
	proc.SetProactiveRepo(proactiveTaskRepo)

	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 2,
		Queues:      map[string]int{"default": 1},
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TaskProcessIncomingMessage, proc.HandleProcessIncomingMessageTask)
	mux.HandleFunc(worker.TaskEvaluateWatch, proc.HandleEvaluateWatchTask)
	mux.HandleFunc(worker.TaskExecuteScheduledReport, proc.HandleExecuteScheduledReportTask)

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Printf("E2E stub worker stopped: %v", err)
		}
	}()

	return srv
}

// loadTestConfig reads config.yml from the project root (two directories up from tests/e2e/).
// It falls back to sensible defaults if the file is missing, so the test binary compiles
// and runs even without infrastructure.
func loadTestConfig() *config.Config {
	// Allow explicit override for CI.
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		os.Setenv("CONFIG_PATH", path)
		return config.Load()
	}

	// Resolve the project root relative to this file.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Println("WARNING: cannot resolve caller path — using default config.yml")
		return config.Load()
	}
	// tests/e2e/suite_test.go → ../.. → project root
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	cfgPath := filepath.Join(projectRoot, "config.yml")

	os.Setenv("CONFIG_PATH", cfgPath)
	defer os.Unsetenv("CONFIG_PATH")

	return config.Load()
}

// openTestDB opens the Bruce SQLite database at the DSN from config and applies the schema.
// If the DSN is empty it falls back to a fresh in-memory database so that DB-only tests
// (FindOrCreate idempotency etc.) work without a persistent DB file.
func openTestDB(t *testing.T, cfg *config.Config) *sql.DB {
	t.Helper()

	dsn := cfg.SQLite.DSN
	if dsn == "" {
		dsn = ":memory:"
	}

	// Resolve relative DSNs against the project root (two directories up from tests/e2e/).
	// go test sets the working directory to the package directory, so relative paths
	// from config.yml (e.g. "./data/bruce.db") would otherwise resolve incorrectly.
	if dsn != ":memory:" && !filepath.IsAbs(dsn) {
		_, thisFile, _, ok := runtime.Caller(0)
		if ok {
			projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
			dsn = filepath.Join(projectRoot, dsn)
		}
	}

	db, err := database.NewSQLiteDB(dsn)
	if err != nil {
		t.Fatalf("openTestDB: cannot open %q: %v", dsn, err)
	}

	// Ensure the schema exists and apply migrations.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("openTestDB: run migrations: %v", err)
	}

	return db
}
