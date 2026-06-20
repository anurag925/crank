---
title: OpenTelemetry Feature
---

# OpenTelemetry feature

The `otel` feature adds **distributed tracing** using OpenTelemetry. It ships with a stdout span exporter (no external collector required) and Echo middleware that starts a span per incoming request.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/tracer.go` | TracerProvider interface |
| `internal/adapters/telemetry/otel.go` | OTel SDK setup with stdout exporter |
| `internal/adapters/http/web/middleware/tracing.go` | Echo middleware for per-request tracing spans |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go) | Tracing SDK, spans, context propagation | [opentelemetry.io/docs/languages/go](https://opentelemetry.io/docs/languages/go/) |
| [OTel stdout exporter](https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/stdout/stdouttrace) | Span export to stdout (no collector needed) | [pkg.go.dev](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/stdout/stdouttrace) |

## Learning resources

- **OpenTelemetry Go guide** — [opentelemetry.io/docs/languages/go](https://opentelemetry.io/docs/languages/go/) (getting started, instrumentation, exporters)
- **OpenTelemetry concepts** — [opentelemetry.io/docs/concepts](https://opentelemetry.io/docs/concepts/) (traces, metrics, logs)
- **Go OTel SDK** — [github.com/open-telemetry/opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go)
- **OpenTelemetry specification** — [opentelemetry.io/docs/specs/otel](https://opentelemetry.io/docs/specs/otel/)

## Notes

- By default, spans are written to stdout in JSON format — useful for local development without running a collector.
- Replace the stdout exporter with an OTLP exporter for production (e.g., to send traces to Jaeger, Grafana Tempo, or Datadog).
- Configure the service name and sampling rate in `configs/config.yaml` under the `telemetry` section.
- The tracing middleware is applied to all routes and creates a span named after the route pattern (e.g., `POST /users`).
