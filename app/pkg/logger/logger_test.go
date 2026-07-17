package logger

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"drawo/config"
)

func TestLoggerInit(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "debug",
		Format: "json",
	}
	Init(cfg)
	assert.NotNil(t, L)

	cfg.Format = "text"
	cfg.Level = "info"
	Init(cfg)

	cfg.Level = "unknown"
	Init(cfg)
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithUserID(ctx, "user-456")

	logger := WithContext(ctx)
	assert.NotNil(t, logger)
	
	// Test without values (this hits the else branch)
	logger = WithContext(context.Background())
	assert.NotNil(t, logger)
}

func TestParseLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, parseLevel("debug"))
	assert.Equal(t, slog.LevelInfo, parseLevel("info"))
	assert.Equal(t, slog.LevelWarn, parseLevel("warn"))
	assert.Equal(t, slog.LevelError, parseLevel("error"))
	assert.Equal(t, slog.LevelInfo, parseLevel("unknown"))
}
