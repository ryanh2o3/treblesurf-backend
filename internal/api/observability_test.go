package httphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"treblesurf-backend/internal/observability"
)

func TestTraceContinuityAndUnsampledMetrics(t *testing.T) {
	for _, sampled := range []string{"0", "1"} {
		t.Run(sampled, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(recorder),
				sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
			)
			previous := otel.GetTracerProvider()
			otel.SetTracerProvider(provider)
			defer otel.SetTracerProvider(previous)
			defer provider.Shutdown(context.Background())
			var output bytes.Buffer
			r := gin.New()
			r.Use(ObservabilityMiddleware(observability.Config{Enabled: true, Environment: "test"}, &output))
			r.Use(RequestLoggerMiddleware())
			r.GET("/api/items/:id", func(c *gin.Context) {
				_, child := otel.Tracer("test").Start(c.Request.Context(), "dependency")
				child.End()
				c.Status(503)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/items/private-id?token=secret", http.NoBody)
			req.Header.Set("X-Amzn-Trace-Id", "Root=1-65000000-123456789012345678901234;Parent=1234567890123456;Sampled="+sampled)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != 503 {
				t.Fatal(response.Code)
			}
			if strings.Contains(output.String(), "private-id") || strings.Contains(output.String(), "secret") {
				t.Fatal("private URL in metrics")
			}
			var metric map[string]any
			if err := json.Unmarshal(output.Bytes(), &metric); err != nil {
				t.Fatal(err)
			}
			if metric["Route"] != "/items/:id" || metric["Requests"] != float64(1) || metric["ServerErrors"] != float64(1) {
				t.Fatal(metric)
			}
			spans := recorder.Ended()
			if sampled == "0" {
				if len(spans) != 0 {
					t.Fatal("unsampled parent ignored")
				}
				return
			}
			if len(spans) != 2 {
				t.Fatalf("got %d spans", len(spans))
			}
			if spans[1].Parent().SpanID().String() != "1234567890123456" || spans[0].Parent().SpanID() != spans[1].SpanContext().SpanID() {
				t.Fatal("broken trace parent chain")
			}
		})
	}
}
