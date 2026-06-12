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
	return "Temporal workflow engine: client, worker, slog-bridged logging and example workflow/activity"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"go.temporal.io/sdk",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_temporal_logger.go.tmpl", OutputPath: "internal/temporal/logger.go"},
		{TemplatePath: "templates/internal_temporal_client.go.tmpl", OutputPath: "internal/temporal/client.go"},
		{TemplatePath: "templates/internal_temporal_worker.go.tmpl", OutputPath: "internal/temporal/worker.go"},
		{TemplatePath: "templates/internal_workflow_greeting.go.tmpl", OutputPath: "internal/workflow/greeting.go"},
		{TemplatePath: "templates/internal_workflow_greeting_test.go.tmpl", OutputPath: "internal/workflow/greeting_test.go"},
		{TemplatePath: "templates/internal_activity_greeting.go.tmpl", OutputPath: "internal/activity/greeting.go"},
		{TemplatePath: "templates/internal_activity_greeting_test.go.tmpl", OutputPath: "internal/activity/greeting_test.go"},
		{TemplatePath: "templates/cmd_worker_main.go.tmpl", OutputPath: "cmd/worker/main.go"},
	}
}
