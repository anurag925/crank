package temporal

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

func (feature) Name() string { return "temporal" }
func (feature) Description() string {
	return "Temporal workflow engine: client, worker, slog-bridged logging and example workflow/activity under internal/adapters/temporal"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"go.temporal.io/sdk",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_temporal_logger.go.tmpl", OutputPath: "internal/adapters/temporal/logger.go"},
		{TemplatePath: "templates/internal_adapters_temporal_worker.go.tmpl", OutputPath: "internal/adapters/temporal/worker.go"},
		{TemplatePath: "templates/internal_adapters_temporal_workflow_greeting.go.tmpl", OutputPath: "internal/adapters/temporal/workflow/greeting.go"},
		{TemplatePath: "templates/internal_adapters_temporal_activity_greeting.go.tmpl", OutputPath: "internal/adapters/temporal/activity/greeting.go"},
		{TemplatePath: "templates/cmd_worker_main.go.tmpl", OutputPath: "cmd/worker/main.go"},
	}
}
