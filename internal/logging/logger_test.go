package logging

import (
	"bytes"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLevel(t *testing.T) {
	tests := map[string]struct {
		env  string
		want Level
	}{
		"development": {env: "development", want: DebugLevel},
		"dev":         {env: "dev", want: DebugLevel},
		"staging":     {env: "staging", want: InfoLevel},
		"stage":       {env: "stage", want: InfoLevel},
		"production":  {env: "production", want: InfoLevel},
		"prod":        {env: "prod", want: InfoLevel},
		"empty":       {env: "", want: InfoLevel},
		"unknown":     {env: "unknown", want: InfoLevel},
		"mixed_case":  {env: "DEvElOpMeNt", want: DebugLevel},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseLevel(tc.env)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDebugLogDevelopmentMode(t *testing.T) {
	// In development mode, debug logs should appear.
	logger := &Logger{level: DebugLevel}

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger.Debug("test debug message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "test debug message")
}

func TestDebugLogProductionMode(t *testing.T) {
	// In production mode, debug logs should be suppressed.
	logger := &Logger{level: InfoLevel}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger.Debug("test debug message")

	output := buf.String()
	// Should be empty or very minimal (no DEBUG log)
	assert.NotContains(t, output, "DEBUG")
	assert.NotContains(t, output, "test debug message")
}

func TestInfoLogAlwaysShown(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"development", DebugLevel},
		{"production", InfoLevel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := &Logger{level: tc.level}

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			logger.Info("test info message")

			output := buf.String()
			assert.Contains(t, output, "INFO")
			assert.Contains(t, output, "test info message")
		})
	}
}

func TestWarnLogAlwaysShown(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"development", DebugLevel},
		{"production", InfoLevel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := &Logger{level: tc.level}

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			logger.Warn("test warn message")

			output := buf.String()
			assert.Contains(t, output, "WARN")
			assert.Contains(t, output, "test warn message")
		})
	}
}

func TestErrorLogAlwaysShown(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"development", DebugLevel},
		{"production", InfoLevel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := &Logger{level: tc.level}

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			logger.Error("test error message")

			output := buf.String()
			assert.Contains(t, output, "ERROR")
			assert.Contains(t, output, "test error message")
		})
	}
}

func TestFormattedLogs(t *testing.T) {
	logger := &Logger{level: DebugLevel}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logger.Debugf("formatted %s message with %d args", "debug", 2)

	output := buf.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "formatted")
	assert.Contains(t, output, "debug message with 2 args")
}

func TestPackageLevelFunctions(t *testing.T) {
	// Reset the global logger to use DebugLevel for this test
	defaultLogger = &Logger{level: DebugLevel}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	Debug("test debug")
	Info("test info")
	Warn("test warn")
	Error("test error")

	output := buf.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "WARN")
	assert.Contains(t, output, "ERROR")
}

func TestPackageLevelDebugSuppressed(t *testing.T) {
	// Reset the global logger to use InfoLevel (production mode)
	defaultLogger = &Logger{level: InfoLevel}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	Debug("this should not appear")
	Info("this should appear")

	output := buf.String()
	assert.NotContains(t, output, "this should not appear")
	assert.Contains(t, output, "this should appear")
}

func TestInitEnvironmentVariable(t *testing.T) {
	tests := map[string]struct {
		env         string
		expectedMin Level
	}{
		"development": {env: "development", expectedMin: DebugLevel},
		"production":  {env: "production", expectedMin: InfoLevel},
		"staging":     {env: "staging", expectedMin: InfoLevel},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set environment variable
			oldEnv := os.Getenv("ENV")
			defer os.Setenv("ENV", oldEnv)

			os.Setenv("ENV", tc.env)

			// Reset the once.Do to allow re-initialization
			once = sync.Once{}
			defaultLogger = nil

			logger := Init()
			assert.Equal(t, tc.expectedMin, logger.level)
		})
	}
}

func TestConcurrentLogging(t *testing.T) {
	logger := &Logger{level: InfoLevel}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Simulate concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Infof("concurrent message %d", id)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "concurrent message")
	// Ensure all 10 messages were logged
	count := strings.Count(output, "concurrent message")
	assert.Equal(t, 10, count)
}

func TestDefaultLoggerInitialization(t *testing.T) {
	// Note: We can't easily reset sync.Once, so we just test that
	// Get() returns an initialized logger with a valid level.
	logger := Get()
	assert.NotNil(t, logger)
	assert.True(t, logger.level >= DebugLevel && logger.level <= ErrorLevel)

	// Get should return the same instance
	logger2 := Get()
	assert.Equal(t, logger, logger2)
}
