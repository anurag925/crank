package platform

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

func (feature) Name() string { return "platform" }
func (feature) Description() string {
	return "Platform client pattern: shared HTTP helper + typed port interfaces for external services"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/go-resty/resty/v2",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_platform_client.go.tmpl", OutputPath: "internal/adapters/platform/client.go"},
		{TemplatePath: "templates/internal_ports_platform_types.go.tmpl", OutputPath: "internal/ports/platform/types.go"},
	}
}
