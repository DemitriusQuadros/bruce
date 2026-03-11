// Package e2e_test contains end-to-end BDD tests for Bruce using godog.
// Tests require a running Bruce server and Redis instance.
package e2e_test

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
)

// newAsynqClient creates an Asynq client connected to the given Redis address.
// The caller is responsible for calling Close() when done.
func newAsynqClient(redisAddr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
}

// TestContext holds shared state for all E2E scenarios.
// It is reset between scenarios via the Before hook in common_steps.go.
type TestContext struct {
	// BaseURL is the root URL of the running Bruce API server.
	// Defaults to http://localhost:8080; overridden by BRUCE_BASE_URL env var.
	BaseURL string

	// HTTPClient is the shared HTTP client used for API health checks.
	HTTPClient *http.Client

	// DB is a direct connection to the Bruce SQLite database for DB assertions.
	// Uses database/sql — NOT gorm.
	DB *sql.DB

	// AsynqClient is used to enqueue tasks directly in tests.
	AsynqClient *asynq.Client

	// RedisAddr is the Redis address used by Asynq (e.g. "localhost:6379").
	RedisAddr string

	// LastResponse and LastBody hold the most recent HTTP response from the API.
	LastResponse *http.Response
	LastBody     []byte

	// ScenarioData is scenario-scoped scratch space. Reset before every scenario.
	ScenarioData map[string]interface{}
}

// NewTestContext constructs a TestContext with the given base URL, DB, and Asynq client.
func NewTestContext(baseURL string, db *sql.DB, asynqClient *asynq.Client, redisAddr string) *TestContext {
	return &TestContext{
		BaseURL:      baseURL,
		HTTPClient:   &http.Client{Timeout: 15 * time.Second},
		DB:           db,
		AsynqClient:  asynqClient,
		RedisAddr:    redisAddr,
		ScenarioData: make(map[string]interface{}),
	}
}
