package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestConfigRejectsInvalidSampling(t *testing.T) {
	t.Setenv("OBSERVABILITY_ENABLED", "true")
	t.Setenv("OBSERVABILITY_ENVIRONMENT", "test")
	t.Setenv("SERVICE_VERSION", "test")
	for _, value := range []string{"NaN", "-1", "1.1", "abc"} {
		t.Setenv("OTEL_TRACES_SAMPLER_ARG", value)
		if _, err := LoadConfig(); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	t.Setenv("OBSERVABILITY_ENABLED", "false")
	if _, err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
}

func TestSDKSpanParentAndNoPayload(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(previous)
	defer provider.Shutdown(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		_, _ = w.Write([]byte(`{"Items":[]}`))
	}))
	defer server.Close()
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"), Endpoint: aws.String(server.URL),
		Credentials: credentials.NewStaticCredentials("test", "test", ""), MaxRetries: aws.Int(0),
	}))
	InstrumentAWS(&sess.Handlers)
	client := dynamodb.New(sess)
	ctx, parent := provider.Tracer("test").Start(context.Background(), "parent")
	_, err := client.QueryWithContext(ctx, &dynamodb.QueryInput{
		TableName: aws.String("private-table"), KeyConditionExpression: aws.String("secret-value"),
	})
	parent.End()
	if err != nil {
		t.Fatal(err)
	}
	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("spans=%d", len(spans))
	}
	if spans[0].Parent().SpanID() != parent.SpanContext().SpanID() || spans[0].SpanKind() != trace.SpanKindClient {
		t.Fatal("AWS span lost parent")
	}
	for _, a := range spans[0].Attributes() {
		if a.Value.AsString() == "private-table" || a.Value.AsString() == "secret-value" {
			t.Fatal("payload in span")
		}
	}
}

func TestExporterOutageFlushIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", server.URL+"/v1/traces")
	previous := otel.GetTracerProvider()
	defer otel.SetTracerProvider(previous)
	r, err := Start(context.Background(), Config{Enabled: true, Environment: "test", Release: "test", SampleRate: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Shutdown(context.Background())
	_, span := r.Provider.Tracer("test").Start(context.Background(), "outage")
	span.End()
	started := time.Now()
	r.Flush()
	if time.Since(started) > time.Second {
		t.Fatal("flush exceeded bounded allowance")
	}
}
