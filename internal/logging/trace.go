package logging

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

// traceHandler uses the context at the log call, falling back to the context
// bound by FromContext for legacy logger.Info calls. No global request state.
type traceHandler struct {
	next  slog.Handler
	bound context.Context
}

func (h traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

//nolint:gocritic // slog.Handler requires passing Record by value.
func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() && h.bound != nil {
		sc = trace.SpanContextFromContext(h.bound)
	}
	if sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.next.Handle(ctx, r)
}
func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.next = h.next.WithAttrs(attrs)
	return h
}
func (h traceHandler) WithGroup(name string) slog.Handler { h.next = h.next.WithGroup(name); return h }
