package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents the logging level.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Logger provides structured logging with environment-aware level filtering.
type Logger struct {
	level Level
	mu    sync.Mutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the global logger based on the ENV environment variable.
// Call this once at application startup.
func Init() *Logger {
	once.Do(func() {
		env := os.Getenv("ENV")
		if env == "" {
			env = "development"
		}
		level := parseLevel(env)
		defaultLogger = &Logger{level: level}
	})
	return defaultLogger
}

// Get returns the global logger instance.
func Get() *Logger {
	if defaultLogger == nil {
		return Init()
	}
	return defaultLogger
}

// parseLevel converts an environment name to a log level.
// - "development" → DebugLevel (show DEBUG and higher)
// - "staging" → InfoLevel (show INFO and higher, skip DEBUG)
// - "production" → InfoLevel (show INFO and higher, skip DEBUG)
// - unknown/empty → InfoLevel
func parseLevel(env string) Level {
	switch strings.ToLower(env) {
	case "development", "dev":
		return DebugLevel
	case "staging", "stage":
		return InfoLevel
	case "production", "prod":
		return InfoLevel
	default:
		return InfoLevel
	}
}

// Debug logs a debug message (only shown in development mode).
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.log("DEBUG", msg, args...)
	}
}

// Debugf logs a formatted debug message (only shown in development mode).
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.logf("DEBUG", format, args...)
	}
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.log("INFO", msg, args...)
	}
}

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.logf("INFO", format, args...)
	}
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.log("WARN", msg, args...)
	}
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.logf("WARN", format, args...)
	}
}

// Error logs an error message (always shown).
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log("ERROR", msg, args...)
}

// Errorf logs a formatted error message (always shown).
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logf("ERROR", format, args...)
}

// log writes a message with the given level prefix.
func (l *Logger) log(level, msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Simple key=value concatenation for args
	var extra string
	for i := 0; i < len(args); i++ {
		if i > 0 {
			extra += ", "
		}
		extra += formatArg(args[i])
	}

	if extra != "" {
		log.Printf("%s: %s (%s)", level, msg, extra)
	} else {
		log.Printf("%s: %s", level, msg)
	}
}

// logf writes a formatted message with the given level prefix.
func (l *Logger) logf(level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("%s: %s", level, fmt.Sprintf(format, args...))
}

// formatArg converts a single argument to string.
func formatArg(arg interface{}) string {
	return fmt.Sprintf("%v", arg)
}

// Package-level convenience functions that use the global logger.

// Debug logs a debug message using the global logger.
func Debug(msg string, args ...interface{}) {
	Get().Debug(msg, args...)
}

// Debugf logs a formatted debug message using the global logger.
func Debugf(format string, args ...interface{}) {
	Get().Debugf(format, args...)
}

// Info logs an info message using the global logger.
func Info(msg string, args ...interface{}) {
	Get().Info(msg, args...)
}

// Infof logs a formatted info message using the global logger.
func Infof(format string, args ...interface{}) {
	Get().Infof(format, args...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, args ...interface{}) {
	Get().Warn(msg, args...)
}

// Warnf logs a formatted warning message using the global logger.
func Warnf(format string, args ...interface{}) {
	Get().Warnf(format, args...)
}

// Error logs an error message using the global logger (always shown).
func Error(msg string, args ...interface{}) {
	Get().Error(msg, args...)
}

// Errorf logs a formatted error message using the global logger (always shown).
func Errorf(format string, args ...interface{}) {
	Get().Errorf(format, args...)
}
