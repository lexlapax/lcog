package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/lexlapax/cogmem/pkg/entity"
)

// Level represents log levels
type Level string

// Log levels
const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
)

// Format represents log output format
type Format string

// Log formats
const (
	TextFormat Format = "text"
	JSONFormat Format = "json"
)

// Config holds configuration for the logger
type Config struct {
	// Level is the minimum log level that will be output
	Level Level `yaml:"level" mapstructure:"level"`
	
	// Format specifies the output format (text or json)
	Format Format `yaml:"format" mapstructure:"format"`
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() Config {
	return Config{
		Level:  InfoLevel,
		Format: TextFormat,
	}
}

// contextKey is a private type for context keys
type contextKey int

const (
	// loggerKey is the key for storing the logger in a context
	loggerKey contextKey = iota
)

// Setup initializes the global logger with the given configuration
func Setup(cfg Config) *slog.Logger {
	var level slog.Level

	// Parse log level
	switch strings.ToLower(string(cfg.Level)) {
	case string(DebugLevel):
		level = slog.LevelDebug
	case string(InfoLevel):
		level = slog.LevelInfo
	case string(WarnLevel):
		level = slog.LevelWarn
	case string(ErrorLevel):
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	switch strings.ToLower(string(cfg.Format)) {
	case string(JSONFormat):
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	
	return logger
}

// SetupWithOutput is like Setup but allows specifying the output destination
func SetupWithOutput(cfg Config, w io.Writer) *slog.Logger {
	var level slog.Level

	// Parse log level
	switch strings.ToLower(string(cfg.Level)) {
	case string(DebugLevel):
		level = slog.LevelDebug
	case string(InfoLevel):
		level = slog.LevelInfo
	case string(WarnLevel):
		level = slog.LevelWarn
	case string(ErrorLevel):
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	switch strings.ToLower(string(cfg.Format)) {
	case string(JSONFormat):
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	logger := slog.New(handler)
	return logger
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context
// If no logger is found, it returns the default logger
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithEntityContext returns a logger with entity context fields added
func WithEntityContext(logger *slog.Logger, entityCtx entity.Context) *slog.Logger {
	return logger.With(
		slog.String("entity_id", string(entityCtx.EntityID)),
		slog.String("user_id", entityCtx.UserID),
	)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// DebugContext logs a debug message with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// InfoContext logs an info message with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// WarnContext logs a warning message with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// ErrorContext logs an error message with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}