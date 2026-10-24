# Observability on the existing Treble Surf environment

Use the normal production environment and existing deployment workflows. No staging environment, separate AWS account, new database, or duplicate frontend is required. Telemetry is best-effort and off until enabled. These changes have not been deployed to AWS.

## Before deploying: AWS setup

### Recovery after deployment (2026-10-24)

The custom collector configuration caused `Extension.InitError` and API Gateway 502s: the attached Lambda collector has no `memory_limiter` processor and rejects the X-Ray exporter keys `sending_queue` and `retry_on_failure`. These unsupported settings have been removed from `collector.yaml` locally.

The collector layer was detached from the live `api` Lambda to restore service. Both a direct Lambda health invocation and `https://treblesurf.com/api/health` returned HTTP 200 afterward. Application spans cannot be exported while the layer is detached; platform X-Ray tracing remains enabled. The corrected collector file still needs deployment and runtime verification before the layer is reattached. The existing workflow does not attach layers automatically, so an ordinary deployment will leave the recovery in place.

The actual Lambda CloudWatch log group is `lambda`, not `/aws/lambda/api`.

### Applied through AWS CLI (2026-10-24)

In account `759663378274`, region `eu-west-1`:

- Attached `arn:aws:lambda:eu-west-1:901920570463:layer:aws-otel-collector-amd64-ver-0-156-0:4` to the existing `api` Lambda and enabled active tracing. The update completed successfully and a read-only `/health` invocation returned HTTP 200 with `{"status":"healthy"}`.
- Attached `AWSXRayDaemonWriteAccess` to `api-role-pydrwpue`.
- Enabled tracing on REST API `b6xei45o57`, existing stage `test`. Despite its name, this is the stage used by the live `treblesurf.com` CloudFront API origin. No environment or stage was created.
- Confirmed Lambda has no VPC configuration, so no endpoint or egress changes are needed.
- Confirmed the `/api/*` origin request policy forwards all viewer headers except `Host`, including the tracing headers; its cache policy is caching-disabled.

The extension currently uses its bundled default configuration. The application deployment will supply the custom collector file and its environment setting together. Application instrumentation and frontend RUM have not been deployed or enabled by this AWS setup. GitHub variables below remain outstanding. Compatibility of the custom collector configuration still needs verification before deployment.

RUM/Cognito setup was also completed on 2026-10-24: monitor `TrebleSurf` has ID `d7cb6787-9008-4a86-8792-bf95d6a7f75c`, and identity pool `eu-west-1:7f21c8b2-5081-4d9c-84ab-d6d6630865f4`. The monitor is in `CREATED` state for `treblesurf.com`, with X-Ray enabled, 10% sampling and cookies disabled. Guest role `TrebleSurfRUMGuest` is restricted to this pool's unauthenticated identities and `rum:PutRumEvents` on this monitor only.

### 1. Existing `api` Lambda in `eu-west-1`

- Add the ADOT **collector extension** layer matching the function's x86_64 architecture and region. Choose and pin the current compatible version from the [official ADOT Go Lambda instructions](https://aws-otel.github.io/docs/getting-started/lambda/lambda-go/). The workflow already builds linux/amd64 and packages `observability/collector.yaml` in the ZIP. This is manually instrumented Go; no language auto-instrumentation wrapper is needed.
- Give the existing Lambda execution role X-Ray write permissions, for example `AWSXRayDaemonWriteAccess`, alongside its existing application and CloudWatch Logs permissions.
- Enable active tracing on the existing Lambda and REST API Gateway stage. Apply/deploy the API stage settings as required. Preserve the existing API routing and integration.
- If Lambda runs in a VPC, ensure the collector can reach X-Ray through egress or an appropriate endpoint.

The deployment workflow supplies these settings automatically:

| Lambda variable | Value |
| --- | --- |
| `OBSERVABILITY_ENABLED` | GitHub production variable, default `false` |
| `OBSERVABILITY_ENVIRONMENT` | `production` |
| `SERVICE_VERSION` | deployed commit SHA |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | `http://127.0.0.1:4318/v1/traces` |
| `OTEL_TRACES_SAMPLER_ARG` | GitHub production variable, default `0.1` |
| `OPENTELEMETRY_COLLECTOR_CONFIG_FILE` | `/var/task/observability/collector.yaml` |

The collector listens on loopback and exports to X-Ray. Confirm the selected layer supports the supplied collector configuration. The environment-update script merges these values with existing Lambda settings using a revision check; application secrets and unrelated settings are preserved. It does not add the layer or change tracing settings for you.

### 2. CloudWatch RUM

Create a RUM app monitor in **eu-west-1** for the actual frontend domain (`treblesurf.com`; include any served `www`/other frontend domains as appropriate). Enable X-Ray integration. Let the console create a Cognito identity pool with unauthenticated identities and its guest role.

The guest role needs `rum:PutRumEvents` scoped to the app monitor ARN. It does not need access to application data. Record the **app monitor ID** and **identity pool ID**. These are public configuration values, not AWS credentials. See [AWS RUM authorization](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch-RUM-get-started-authorization.html).

The app remains opt-in: users enable **Share optional performance diagnostics** in Settings. Collection uses no tracking cookies or session replay. Revoking consent stops dispatch and reloads the page.

### 3. Existing CloudFront/API routing

Keep the current S3 bucket `treblesurffrontend`, distribution `E3NSJLNC72N931`, and API origin `https://treblesurf.com`.

Check that the `/api/*` behavior forwards `X-Amzn-Trace-Id` and `X-Request-Id` alongside the existing authentication/CSRF headers and cookies. API responses should not be cached or rewritten to the SPA HTML fallback. For cross-origin clients, the Gin CORS configuration now permits these tracing headers and exposes `X-Request-Id`; gateway/CloudFront responses must allow them too.

If the site uses CSP, allow the regional RUM data-plane and Cognito identity endpoints in `connect-src` (and STS if the chosen credential flow uses it). The SDK itself is bundled locally.

## GitHub production environment variables

Use the existing **production** environment in each repository, not a new environment.

Backend:

| Variable | Value |
| --- | --- |
| `OBSERVABILITY_ENABLED` | `true` once the layer/permissions are ready |
| `OTEL_TRACES_SAMPLER_ARG` | optional; defaults to `0.1` |

Frontend:

| Variable | Value |
| --- | --- |
| `VITE_RUM_ENABLED` | `true` once the monitor exists |
| `VITE_RUM_APP_MONITOR_ID` | monitor ID from CloudWatch |
| `VITE_RUM_IDENTITY_POOL_ID` | `eu-west-1:...` identity pool ID |
| `VITE_RUM_SESSION_SAMPLE_RATE` | optional; defaults to `0.1` |

The frontend workflow sets the API origin, region, `VITE_APP_ENVIRONMENT=production`, and release SHA. Existing Mapbox secrets and AWS deployment roles stay in use. Do not put access keys or server secrets in `VITE_*` variables. Changes to frontend variables require a rebuild.

## Deploy and check during ordinary use

1. Deploy through the usual **Deploy to AWS Lambda** workflow, then **Deploy Vue Frontend**, using the existing main/manual deployment paths.
2. Check collector startup in `/aws/lambda/api`. Open Settings, opt into diagnostics, and browse `/buoys` using a real buoy. Sampling means not every visit/request will have a trace.
3. Inspect RUM/X-Ray for browser → gateway/Lambda → Gin → `buoy.history` → DynamoDB spans. Browser propagation is limited to `/api/getLast24BuoyData` and `/api/user/preferences`. Authentication providers, maps and media URLs are excluded. The preferences endpoint can be tested with an existing signed-in session.
4. Find logs for a trace using Logs Insights. Convert X-Ray's `1-xxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxx` root to the 32-character OTel form by removing `1-` and the remaining dash:

   ```text
   fields @timestamp, msg, request_id, trace_id, span_id, route, status, duration_ms
   | filter trace_id = "REPLACE_WITH_32_HEX_TRACE_ID"
   | sort @timestamp asc
   ```

5. Optionally install the supplied dashboard:

   ```sh
   aws cloudwatch put-dashboard --region eu-west-1 --dashboard-name TrebleSurf-production --dashboard-body file://observability/dashboard.json
   ```

   It is already configured for `production`, the `api` function, and `eu-west-1`. Route metrics appear after traffic. RUM provides its own browser dashboard.
6. Watch ordinary API/Lambda errors, duration, memory and telemetry volume after enabling. Set log retention and a budget alert appropriate to your account. No load campaign, dependency outage, staging sign-off or production rollout gate is required for this experiment.

An optional small read-only check is `BUOY_NAME=M4 k6 run observability/load.js`, choosing an existing buoy. It makes ten requests with one virtual user and a one-second pause. This script is not run by deployments. Failure injection remains in local tests; do not disrupt the live database to validate diagnostics.

## Best-effort behavior and disabling

- Malformed telemetry configuration disables diagnostics with a warning instead of blocking API startup. Exporter initialization failure also lets the application start. Normal application configuration errors still behave normally.
- Export queues/timeouts are bounded. Lambda flushes have a 200 ms budget and do not change API responses; exporter failure can lose telemetry. The provider is reused between invocations.
- Browser SDK loading has a one-second startup allowance; the app can mount while a slow SDK finishes. Missing RUM settings or initialization failures leave the application usable.
- Backend root sampling defaults to 10%; supported incoming parent sampling decisions take precedence. Browser sessions default to 10% of opted-in visits. Gateway/Lambda sampling also affects actual trace retention. Route metrics are unsampled and independent of the collector.
- Set GitHub production `OBSERVABILITY_ENABLED=false` and redeploy the backend to turn off application telemetry. Set `VITE_RUM_ENABLED=false` and rebuild/redeploy the frontend to turn off RUM. A direct Lambda flag change takes effect immediately but the next deployment uses GitHub's value.
- If the collector extension itself causes startup/resource problems, remove that layer while preserving existing layers. Disable platform active tracing separately if desired. An extension runs outside the Go error handling, so the feature flag alone does not unload it.

## Coverage and verification

Go includes Gin spans, the buoy-history operation, shared AWS repository client spans, context-aware log correlation, request-scoped session adapters, and unsampled route EMF metrics. Legacy storage wrappers, context-free global slog calls, outbound Google HTTP, and WebSocket tracing are not fully covered. Known sensitive diagnostic attributes are redacted; this does not rewrite historical logs or replace an audit-log policy.

Vue records router page views, navigation/web vitals, generic errors, and the two approved API journeys. Query values, arbitrary error text, stack traces and DOM attribution strings are redacted. Browser IDs, technical metadata and temporary Cognito credentials still exist; cookie-free collection is not zero-data collection.

The full Go suite, focused race tests, Linux Lambda build, frontend policy/browser tests, and enabled/disabled builds passed during implementation. Changed-code lint passes; older Go tests have existing repository-wide lint findings. Browser tests mock AWS and do not prove managed AWS trace continuity. Live cost and overhead have not been measured. Local commands:

```sh
go test ./...
go test -race ./internal/observability ./internal/logging ./internal/api ./internal/auth/... ./internal/service
python3 -m unittest discover -s scripts -p 'test_update_lambda_environment.py'
```

In the frontend repository:

```sh
npm run test:observability
npm run test:observability:browser
npm run build
```

For browser tests, install Chromium using `npx playwright install chromium`, or use `PLAYWRIGHT_CHANNEL=chrome` with installed Chrome.
