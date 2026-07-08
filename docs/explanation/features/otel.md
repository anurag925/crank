---
title: OpenTelemetry Feature
---

# OpenTelemetry feature

The `otel` feature adds **distributed tracing** with a stdout span exporter and Echo middleware for per-request spans.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/tracer.go` | TracerProvider interface |
| `internal/adapters/telemetry/otel.go` | OTel SDK setup with stdout exporter |
| `internal/adapters/http/web/middleware/tracing.go` | Per-request tracing spans |

Spans are written to stdout in JSON format. Swap to OTLP exporter for production. Configurable under the `telemetry` section in `configs/config.yaml`.
