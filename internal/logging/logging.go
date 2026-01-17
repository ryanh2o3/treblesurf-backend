package logging

import (
	"context"
	"log/slog"
	"os"

	"treblesurf-backend/internal/config"
)

type ctxKey struct{}

// Init initializes the global logger based on environment.
func Init(cfg *config.Config) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	var handler slog.Handler
	if cfg.IsDevelopment() {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// FromContext retrieves a logger from context, or returns default.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithLogger adds a logger to context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// WithRequestID returns a logger with request ID attached.
func WithRequestID(l *slog.Logger, requestID string) *slog.Logger {
	return l.With(slog.String("request_id", requestID))
}
package logging

import (
	"context"
	"log/slog"
	"os"

	"treblesurf-backend/internal/config"
)

type ctxKey struct{}

// Init initializes the global logger based on environment.
func Init(cfg *config.Config) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if cfg.IsDevelopment() {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// FromContext retrieves a logger from context, or returns default.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithLogger adds a logger to context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// WithRequestID returns a logger with request ID attached.
func WithRequestID(l *slog.Logger, requestID string) *slog.Logger {
	return l.With(slog.String("request_id", requestID))
}
