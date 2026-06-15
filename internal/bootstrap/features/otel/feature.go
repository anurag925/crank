// Package otel adds OpenTelemetry distributed tracing to a generated
// project. It ships a stdout SpanExporter (no external collector required)
// and an Echo middleware that starts a span per incoming request.
package otel

import (
	"embed"

	"github.com/anurag925/crank/internal/bootstrap"
)

//go:embed templates/*.tmpl
var tmpls embed.FS

type feature struct{}

func init() {
	if err := bootstrap.GlobalRegistry.Register(feature{}); err != nil {
		panic(err)
	}
}

func (feature) Name() string { return "otel" }

func (feature) Description() string {
	return "OpenTelemetry tracing: stdout exporter by default; Echo middleware starts a span per request"
}

func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"go.opentelemetry.io/otel",
		"go.opentelemetry.io/otel/sdk",
		"go.opentelemetry.io/otel/sdk/resource",
		"go.opentelemetry.io/otel/sdk/trace",
		"go.opentelemetry.io/otel/semconv/v1.26.0",
		"go.opentelemetry.io/otel/exporters/stdout/stdouttrace",
		"github.com/labstack/echo/v4", // echoed middleware is the same package
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_telemetry_otel.go.tmpl", OutputPath: "internal/adapters/telemetry/otel.go"},
		{TemplatePath: "templates/internal_ports_tracer.go.tmpl", OutputPath: "internal/ports/tracer.go"},
		{TemplatePath: "templates/internal_adapters_http_web_middleware_tracing.go.tmpl", OutputPath: "internal/adapters/http/web/middleware/tracing.go"},
	}
}
