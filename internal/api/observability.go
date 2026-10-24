package httphandler

import (
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"treblesurf-backend/internal/logging"
	"treblesurf-backend/internal/observability"
)

// Middleware emits only matched route templates and numeric measurements. It
// runs outside recovery so recovered panics are measured as HTTP 500 responses.
func ObservabilityMiddleware(cfg observability.Config, output io.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		started := time.Now()
		route := strings.TrimPrefix(c.FullPath(), "/api")
		if route == "" {
			route = "unmatched"
		}
		method := c.Request.Method
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		default:
			method = "OTHER"
		}
		ctx := c.Request.Context()
		if !trace.SpanContextFromContext(ctx).IsValid() {
			ctx = (xray.Propagator{}).Extract(ctx, propagation.HeaderCarrier(c.Request.Header))
		}
		ctx, span := otel.Tracer(observability.ServiceName).Start(ctx, method+" "+route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attribute.String("http.request.method", method), attribute.String("http.route", route)),
		)
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		status := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.response.status_code", status))
		serverError, clientError := 0, 0
		if status >= 500 {
			serverError = 1
			span.SetStatus(codes.Error, "HTTP server error")
		} else if status >= 400 {
			clientError = 1
		}
		latency := float64(time.Since(started).Microseconds()) / 1000
		logging.FromContext(c.Request.Context()).InfoContext(c.Request.Context(), "http request completed",
			slog.String("route", route), slog.Int("status", status), slog.Float64("duration_ms", latency),
		)
		event := requestMetric(cfg.Environment, route, serverError, clientError, latency)
		if err := json.NewEncoder(output).Encode(event); err != nil {
			slog.Warn("request metric write failed")
		}
	}
}

// requestMetric emits bounded route dimensions, independently of trace sampling.
func requestMetric(environment, route string, serverError, clientError int, latency float64) map[string]any {
	return map[string]any{
		"_aws": map[string]any{
			"Timestamp": time.Now().UnixMilli(),
			"CloudWatchMetrics": []any{map[string]any{
				"Namespace":  "TrebleSurf/API",
				"Dimensions": [][]string{{"Service", "Environment", "Route"}},
				"Metrics": []any{
					map[string]string{"Name": "Requests", "Unit": "Count"},
					map[string]string{"Name": "ServerErrors", "Unit": "Count"},
					map[string]string{"Name": "ClientErrors", "Unit": "Count"},
					map[string]string{"Name": "Duration", "Unit": "Milliseconds"},
				},
			}},
		},
		"Service": observability.ServiceName, "Environment": environment, "Route": route,
		"Requests": 1, "ServerErrors": serverError, "ClientErrors": clientError, "Duration": latency,
	}
}
