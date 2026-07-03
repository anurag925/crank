package qdrant

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

func (feature) Name() string { return "qdrant" }
func (feature) Description() string {
	return "Qdrant vector database client (semantic search, embeddings) — wired into the composition root"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/qdrant/go-client",
		"github.com/go-resty/resty/v2",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_persistence_qdrant_client.go.tmpl", OutputPath: "internal/adapters/persistence/qdrant/client.go"},
		{TemplatePath: "templates/internal_ports_platform_qdrant.go.tmpl", OutputPath: "internal/ports/platform/qdrant.go"},
		{TemplatePath: "templates/internal_adapters_platform_qdrant.go.tmpl", OutputPath: "internal/adapters/platform/qdrant.go"},
	}
}
