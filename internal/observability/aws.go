package observability

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentAWS uses SDK v1 lifecycle hooks: one span per logical operation,
// including retries. No params, keys, table names, URLs, or raw errors are read.
// Configure a client's handlers once, before sharing the client with requests.
func InstrumentAWS(handlers *request.Handlers) {
	handlers.Validate.PushFrontNamed(request.NamedHandler{Name: "treblesurf.trace.start", Fn: func(r *request.Request) {
		ctx, _ := otel.Tracer(ServiceName).Start(r.Context(), r.ClientInfo.ServiceName+"."+r.Operation.Name,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("rpc.system", "aws-api"),
				attribute.String("rpc.service", r.ClientInfo.ServiceName),
				attribute.String("rpc.method", r.Operation.Name),
			),
		)
		r.SetContext(ctx)
	}})
	handlers.Complete.PushBackNamed(request.NamedHandler{Name: "treblesurf.trace.end", Fn: func(r *request.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.Int("aws.retry_count", r.RetryCount))
		if r.HTTPResponse != nil {
			span.SetAttributes(attribute.Int("http.response.status_code", r.HTTPResponse.StatusCode))
		}
		if r.Error != nil {
			span.SetStatus(codes.Error, "AWS operation failed")
		}
		span.End()
	}})
}
