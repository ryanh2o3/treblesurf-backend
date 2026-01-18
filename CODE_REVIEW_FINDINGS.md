# Treble Surf Backend - Code Review Findings

Review date: 2026-01-17  
Scope: Go backend codebase (`cmd/`, `internal/`, `local/`)

This document focuses on actionable findings that would move the codebase
toward more idiomatic, reliable, and maintainable Go. It highlights gaps,
risks, and opportunities for improvement with file references for context.

---

## High Priority Findings

1. **Session store setup is a no-op.**  
   `internal/auth/store/dynamodb.go` implements `EnsureSessionsTable` and
   `EnableTTL` as empty functions returning `nil`. `internal/auth/service.go`
   expects these to initialize and configure TTL on sessions. Today those
   operations silently never happen, which is misleading in production.

2. **Storage constructors can panic despite returning errors.**  
   `internal/storage/s3.go` and `internal/storage/dynamodb.go` call
   `session.Must(session.NewSession(...))` but also return `(*Client, error)`.
   This can panic instead of returning a normal error, and the signature
   implies it should not. Prefer `session.NewSession` and return the error.

3. **Missing functionality behind TODOs affects production behavior.**  
   - `internal/service/report_submit.go` stubs out subscriber lookup and
     broadcasting (`getSpotSubscribers`, `broadcastToUsers`).  
   - `internal/service/report_media.go` skips real content type detection.  
   - `internal/service/tide_service.go` returns sample data and includes a
     TODO for DynamoDB integration.  
   These should be implemented or gated to prevent misleading outputs.

4. **Context cancellation is bypassed in request flows.**  
   - `internal/websocket/handler.go` uses `context.Background()` for storage
     operations instead of a request-scoped context.  
   - `internal/auth/store/dynamodb.go` and
     `internal/auth/service_helpers.go` similarly use `context.Background()`.  
   Use `context.WithTimeout` or pass the caller’s context so work is canceled
   on request termination.

5. **WebSocket API client creation is per-request and can panic.**  
   `internal/service/websocket_service.go` creates a new AWS session on every
   `SendToConnection` call via `session.Must`, which is expensive and can
   panic. Reuse a single client (or a small pool) and return errors instead.

---

## Medium Priority Findings

1. **Rate limiter goroutine has no stop mechanism.**  
   `internal/api/middleware.go` starts a ticker goroutine in `newRateLimiter`
   without a way to stop it. This leaks goroutines in tests and on shutdown.
   Add a `Stop()` method and call it from server shutdown or `t.Cleanup`.

2. **Rate limiter accepts zero or negative RPS.**  
   `RateLimitMiddleware` doesn’t validate `requestsPerSecond`. A zero or
   negative value leads to unexpected behavior (first request passes, then
   permanent denial). Validate and default to a sensible minimum.

3. **Heavy use of `map[string]interface{}` across domain logic.**  
   Multiple services and repositories exchange untyped maps:
   - `internal/service/report_similarity.go`
   - `internal/service/swell_prediction_service.go`
   - `internal/service/tide_service.go`
   - `internal/controller/buoy_controller.go`
   - `internal/repository/dynamodb/forecast_repo.go`
   This makes the code fragile (runtime type assertions, hidden schema).
   Prefer typed DTO structs and explicit JSON marshaling.

4. **Repository error contracts aren’t enforced.**  
   `internal/repository/interfaces.go` documents sentinel errors like
   `ErrNotFound`, but implementations rarely return them. Downstream
   error handling cannot rely on `errors.Is`. Standardize error mapping
   in repository implementations.

5. **Error wrapping loses cause chain in many places.**  
   Several functions use `fmt.Errorf("...: %v", err)` which loses the
   original error for `errors.Is/As`. Use `%w` instead:
   - `internal/storage/s3.go`
   - `internal/service/report_media.go`
   - `internal/service/report_similarity.go`
   - `internal/service/websocket_service.go`

---

## Low Priority / Style Findings

1. **Duplicate CORS middleware.**  
   `internal/api/middleware.go` contains `CorsMiddleware`, but routing
   uses `buildCORSMiddleware` in `internal/api/router.go`. One is likely
   unused. Consider deleting or consolidating to avoid drift.

2. **Large, complex file with multiple responsibilities.**  
   `internal/service/report_similarity.go` is very large and disables
   cyclomatic complexity lint checks. Consider splitting into smaller,
   testable helpers and using typed result structures.

3. **Inefficient media existence checks.**  
   `internal/service/report_media.go` calls `mediaRepo.Download` to verify
   existence in `GenerateVideoViewURL` and `ValidateImageKeyExists`. This
   can download full files. Prefer metadata or `HeadObject` calls.

4. **Time usage can be inconsistent within a single request.**  
   `internal/service/tide_service.go` calls `time.Now()` repeatedly within
   a single operation. Using a single `now := time.Now()` yields consistent
   comparisons and makes tests more deterministic.

---

## Suggested Next Steps (High Impact)

- Implement missing TODOs in reporting/tides or explicitly disable those
  endpoints until complete.
- Replace `session.Must` usages with error-returning constructors.
- Introduce typed DTOs for forecast, swell prediction, tides, and report
  similarity outputs to eliminate `map[string]interface{}` usage.
- Add request-scoped contexts and timeouts in WebSocket handlers and auth
  helpers.
- Provide a shutdown hook for `rateLimiter` cleanup goroutines.

