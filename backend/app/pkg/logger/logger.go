// Package logger provides structured, request-aware logging.
//
// Responsibility:
//   - Initialize a slog.Logger based on configuration.
//   - Provide helpers to attach context (request IDs, user IDs) to log records.
//
// Why slog?
//
//	slog is in the standard library since Go 1.21, supports structured JSON output,
//	and avoids a third-party dependency. It is also faster than logrus and simpler
//	than zap for our needs.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"drawo/config"
)

// contextKey is an unexported type so no external package can collide with our keys.
type contextKey string

const requestIDKey contextKey = "request_id"
const userIDKey contextKey = "user_id"

// L is the global logger. It defaults to slog.Default() and is reconfigured by Init(). Safe for concurrent use.
var L *slog.Logger = slog.Default()

// Init creates the global logger from configuration.
// It should be called once before the server starts.
func Init(cfg config.LogConfig) {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	L = slog.New(handler)
}

// parseLevel converts a string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger enriched with values from the context.
// Use it in HTTP handlers: logger.WithContext(c.Request.Context()).Info(...)
func WithContext(ctx context.Context) *slog.Logger {
	l := L
	if l == nil {
		l = slog.Default()
	}

	if reqID, ok := ctx.Value(requestIDKey).(string); ok && reqID != "" {
		l = l.With(slog.String("request_id", reqID))
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		l = l.With(slog.String("user_id", userID))
	}

	return l
}

// ContextWithRequestID returns a new context carrying a request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// ContextWithUserID returns a new context carrying a user ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
