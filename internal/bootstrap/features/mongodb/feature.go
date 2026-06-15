package mongodb

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

func (feature) Name() string { return "mongodb" }
func (feature) Description() string {
	return "MongoDB client (document storage, aggregation) — wired into the composition root"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"go.mongodb.org/mongo-driver",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_persistence_mongodb_client.go.tmpl", OutputPath: "internal/adapters/persistence/mongodb/client.go"},
	}
}
