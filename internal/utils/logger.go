package utils

import (
	"log/slog"
	"os"
	"path/filepath"
)

// Logger is the global logger instance.
var Logger *slog.Logger

// logFile stores the file handle for cleanup
var logFile *os.File

// LogLevel represents logging severity.
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// InitLogger initializes the global logger.
func InitLogger(logPath string, level LogLevel) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	logFile = file

	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	Logger = slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slogLevel,
	}))

	return nil
}

// SetVerbose enables debug logging.
func SetVerbose() {
	if Logger == nil {
		return
	}
	Logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	if Logger == nil {
		return
	}
	Logger.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	if Logger == nil {
		return
	}
	Logger.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	if Logger == nil {
		return
	}
	Logger.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	if Logger == nil {
		return
	}
	Logger.Error(msg, args...)
}

// With returns a logger with additional context.
func With(args ...any) *slog.Logger {
	if Logger == nil {
		return nil
	}
	return Logger.With(args...)
}

// CloseLogger closes the log file handle.
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}
