package logging

import (
	"bytes"
	"context"
	"encoding/json"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log/slog"
	"testing"
)

func TestRequestLoggerUsesCurrentChildSpan(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	defer provider.Shutdown(context.Background())
	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
	defer parent.End()
	var output bytes.Buffer
	logger := slog.New(traceHandler{next: slog.NewJSONHandler(&output, nil)})
	ctx = WithLogger(ctx, WithRequestID(logger, "request-1"))
	childCtx, child := provider.Tracer("test").Start(ctx, "child")
	defer child.End()
	FromContext(ctx).InfoContext(childCtx, "inside child")
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["span_id"] != child.SpanContext().SpanID().String() || record["request_id"] != "request-1" {
		t.Fatal(record)
	}
	output.Reset()
	FromContext(childCtx).Info("legacy logger call")
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["span_id"] != child.SpanContext().SpanID().String() {
		t.Fatal(record)
	}
}
