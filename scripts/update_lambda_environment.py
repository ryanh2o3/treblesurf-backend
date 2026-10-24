#!/usr/bin/env python3
"""Merge explicitly supplied settings into Lambda without printing secrets.

Uses RevisionId to refuse overwriting a concurrent configuration change.
Empty CI values preserve existing settings; set explicit false to disable tracing.
"""
import argparse
import json
import os
import subprocess
import tempfile

BASE_KEYS = (
    "JWT_SECRET", "GOOGLE_CLIENT_ID", "GOOGLE_IOS_CLIENT_ID", "APPLE_CLIENT_IDS",
    "ADMIN_EMAILS", "GO_ENV", "WEBSOCKET_API_ENDPOINT", "SPOT_IMAGES_BASE_URL",
    "S3_BUCKET_NAME", "FORECAST_TABLE", "ALLOWED_ORIGINS",
)
TELEMETRY_KEYS = (
    "OBSERVABILITY_ENABLED", "OBSERVABILITY_ENVIRONMENT", "SERVICE_VERSION",
    "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_TRACES_SAMPLER_ARG",
    "OPENTELEMETRY_COLLECTOR_CONFIG_FILE",
)


def merge(existing, supplied, telemetry=False):
    result = dict(existing)
    for key in BASE_KEYS + (TELEMETRY_KEYS if telemetry else ()):
        if supplied.get(key):
            result[key] = supplied[key]
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("function")
    parser.add_argument("--telemetry", action="store_true")
    args = parser.parse_args()
    current = json.loads(subprocess.check_output([
        "aws", "lambda", "get-function-configuration", "--function-name", args.function,
        "--output", "json",
    ]))
    environment = current.get("Environment", {})
    if environment.get("Error"):
        raise SystemExit("Cannot read existing Lambda environment; refusing to replace it")
    values = merge(environment.get("Variables", {}), os.environ, args.telemetry)
    with tempfile.NamedTemporaryFile(mode="w", suffix=".json") as payload:
        json.dump({"Variables": values}, payload)
        payload.flush()
        subprocess.run([
            "aws", "lambda", "update-function-configuration", "--function-name", args.function,
            "--revision-id", current["RevisionId"], "--environment", "file://" + payload.name,
            "--query", "FunctionName", "--output", "text",
        ], check=True)
    subprocess.run(["aws", "lambda", "wait", "function-updated", "--function-name", args.function], check=True)


if __name__ == "__main__":
    main()
