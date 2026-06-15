package e2e

// Tests for the `otel` feature. OTel adds OpenTelemetry tracing to a
// generated project — stdout exporter by default, Echo middleware for
// inbound requests, and a TelemetryConfig block in the project config.

import (
	"strings"
	"testing"
)

// TestE2E_Otel_Init runs the init happy path with base+otel and asserts
// the generated project compiles, vets, and contains the otel wiring.
func TestE2E_Otel_Init(t *testing.T) {
	dir := scaffold(t, "otel_init", []string{"base", "otel"})

	// OTel-specific files exist.
	assertExists(t, dir, "internal/adapters/telemetry/otel.go")
	assertExists(t, dir, "internal/ports/tracer.go")
	assertExists(t, dir, "internal/adapters/http/web/middleware/tracing.go")

	// The composition root references the tracer provider and installs
	// the tracing middleware.
	main := readFile(t, dir, "cmd/server/main.go")
	if !strings.Contains(main, "telemetry.NewProvider") {
		t.Errorf("main.go should construct the tracer provider:\n%s", main)
	}
	if !strings.Contains(main, "middleware.Tracing(cfg.Telemetry.ServiceName)") {
		t.Errorf("main.go should install the tracing middleware:\n%s", main)
	}

	// TelemetryConfig is in the config struct.
	cfgGo := readFile(t, dir, "internal/config/config.go")
	if !strings.Contains(cfgGo, "TelemetryConfig") {
		t.Errorf("config.go should contain TelemetryConfig")
	}
	if !strings.Contains(cfgGo, `telemetry.service_name`) {
		t.Errorf("config.go should set telemetry.service_name default")
	}

	// Project must compile and vet cleanly.
	compileProject(t, dir)
}
