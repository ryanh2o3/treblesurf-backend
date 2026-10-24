package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"treblesurf-backend/internal/config"
)

type ctxKey struct{}

// Init initializes the global logger based on environment.
func Init(cfg *config.Config) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if !cfg.Observability.Enabled {
				return a
			}
			switch strings.ToLower(a.Key) {
			case "email", "user_email", "user", "user_id", "apple_id", "admin",
				"authorization", "cookie", "body", "referer", "session_id",
				"csrf", "token", "image_key", "video_key", "media_key", "key", "error":
				return slog.String(a.Key, "[redacted]")
			}
			return a
		},
	}

	var handler slog.Handler
	if cfg.IsDevelopment() {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(traceHandler{next: handler})
	if cfg.Observability.Enabled {
		logger = logger.With("service", "treblesurf-api", "environment", cfg.Observability.Environment, "release", cfg.Observability.Release)
	}
	slog.SetDefault(logger)
}

// FromContext retrieves a logger from context, or returns default.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return bindContext(l, ctx)
	}
	return bindContext(slog.Default(), ctx)
}

// WithLogger adds a logger to context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// WithRequestID returns a logger with request ID attached.
func WithRequestID(l *slog.Logger, requestID string) *slog.Logger {
	return l.With(slog.String("request_id", requestID))
}

func bindContext(l *slog.Logger, ctx context.Context) *slog.Logger {
	if h, ok := l.Handler().(traceHandler); ok {
		h.bound = ctx
		return slog.New(h)
	}
	return slog.New(traceHandler{next: l.Handler(), bound: ctx})
}
