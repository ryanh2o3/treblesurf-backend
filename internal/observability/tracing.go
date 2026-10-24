// Package observability provides opt-in, payload-free application telemetry.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const ServiceName = "treblesurf-api"
const FlushTimeout = 200 * time.Millisecond

type Config struct {
	Environment, Release string
	SampleRate           float64
	Enabled              bool
}

func LoadConfig() (Config, error) {
	c := Config{
		Environment: strings.TrimSpace(os.Getenv("OBSERVABILITY_ENVIRONMENT")),
		Release:     strings.TrimSpace(os.Getenv("SERVICE_VERSION")), SampleRate: 0.1,
	}
	raw := strings.TrimSpace(os.Getenv("OBSERVABILITY_ENABLED"))
	if raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return c, fmt.Errorf("OBSERVABILITY_ENABLED must be true or false")
		}
		c.Enabled = v
	}
	if !c.Enabled {
		return c, nil
	}
	if c.Environment == "" || c.Release == "" {
		return c, fmt.Errorf("enabled telemetry requires OBSERVABILITY_ENVIRONMENT and SERVICE_VERSION")
	}
	if raw = strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG")); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || !(v >= 0 && v <= 1) {
			return c, fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
		}
		c.SampleRate = v
	}
	return c, nil
}

type Runtime struct{ Provider *sdktrace.TracerProvider }

func Start(ctx context.Context, c Config) (*Runtime, error) {
	r := &Runtime{}
	if !c.Enabled {
		return r, nil
	}
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithTimeout(150*time.Millisecond),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{Enabled: false}),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize trace exporter: %w", err)
	}
	r.Provider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", ServiceName), attribute.String("service.version", c.Release),
			attribute.String("deployment.environment", c.Environment),
		)),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(c.SampleRate))),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxQueueSize(512), sdktrace.WithMaxExportBatchSize(128),
			sdktrace.WithBatchTimeout(time.Second), sdktrace.WithExportTimeout(150*time.Millisecond),
		),
	)
	otel.SetTracerProvider(r.Provider)
	otel.SetTextMapPropagator(xray.Propagator{})
	// Exporter failures must never include request payloads or change API responses.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) { slog.Warn("telemetry export failed") }))
	return r, nil
}

// Flush is bounded and detached from an expired request, but never destroys the
// reusable provider. Call after the request span ends, before Lambda returns.
func (r *Runtime) Flush() {
	if r == nil || r.Provider == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), FlushTimeout)
	defer cancel()
	if err := r.Provider.ForceFlush(ctx); err != nil {
		slog.Warn("telemetry flush failed")
	}
}
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || r.Provider == nil {
		return nil
	}
	return r.Provider.Shutdown(ctx)
}

// LambdaContext attaches the runtime parent. Gin extracts the HTTP parent only
// if this context has no parent, preserving Lambda's active-tracing segment.
func LambdaContext(ctx context.Context, header string) context.Context {
	return (xray.Propagator{}).Extract(ctx, propagation.MapCarrier{"X-Amzn-Trace-Id": header})
}
