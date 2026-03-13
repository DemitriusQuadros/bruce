package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LogLevel represents a log message severity level.
type LogLevel string

const (
	DebugLevel LogLevel = "DEBUG"
	InfoLevel  LogLevel = "INFO"
	WarnLevel  LogLevel = "WARN"
	ErrorLevel LogLevel = "ERROR"
)

// LogEntry represents a single structured log entry persisted to the database.
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Logger    string                 `json:"logger"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
}

// StructuredLogger provides structured logging to a SQLite database.
type StructuredLogger struct {
	db      *sql.DB
	enabled bool
	enabledMu sync.RWMutex
}

// NewStructuredLogger creates a new StructuredLogger with the given database connection.
func NewStructuredLogger(db *sql.DB) *StructuredLogger {
	return &StructuredLogger{
		db:      db,
		enabled: true,
	}
}

// SetEnabled allows toggling logging on/off at runtime.
func (sl *StructuredLogger) SetEnabled(enabled bool) {
	sl.enabledMu.Lock()
	defer sl.enabledMu.Unlock()
	sl.enabled = enabled
}

// IsEnabled returns whether logging is currently enabled.
func (sl *StructuredLogger) IsEnabled() bool {
	sl.enabledMu.RLock()
	defer sl.enabledMu.RUnlock()
	return sl.enabled
}

// Debug logs a debug message.
func (sl *StructuredLogger) Debug(ctx context.Context, logger, message string, fields map[string]interface{}) {
	sl.logWithLevel(ctx, DebugLevel, logger, message, fields)
}

// Info logs an info message.
func (sl *StructuredLogger) Info(ctx context.Context, logger, message string, fields map[string]interface{}) {
	sl.logWithLevel(ctx, InfoLevel, logger, message, fields)
}

// Warn logs a warning message.
func (sl *StructuredLogger) Warn(ctx context.Context, logger, message string, fields map[string]interface{}) {
	sl.logWithLevel(ctx, WarnLevel, logger, message, fields)
}

// Error logs an error message.
func (sl *StructuredLogger) Error(ctx context.Context, logger, message string, fields map[string]interface{}) {
	sl.logWithLevel(ctx, ErrorLevel, logger, message, fields)
}

// logWithLevel is the internal method that performs the actual logging.
func (sl *StructuredLogger) logWithLevel(ctx context.Context, level LogLevel, logger, message string, fields map[string]interface{}) {
	if !sl.IsEnabled() {
		return
	}

	// Ensure fields is not nil
	if fields == nil {
		fields = make(map[string]interface{})
	}

	// Serialize context fields to JSON
	contextJSON := ""
	if len(fields) > 0 {
		b, err := json.Marshal(fields)
		if err == nil {
			contextJSON = string(b)
		}
	}

	// Insert log entry asynchronously (non-blocking)
	go func() {
		_, err := sl.db.ExecContext(ctx,
			`INSERT INTO log_entries (id, timestamp, level, logger, message, context, enabled)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(),
			time.Now().UTC(),
			level,
			logger,
			message,
			contextJSON,
			1, // enabled
		)
		if err != nil {
			// Silently fail — logging failure should not crash the app
			// Could add metrics here to track log write failures
		}
	}()
}
