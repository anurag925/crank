package outbox

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

func (feature) Name() string { return "outbox" }
func (feature) Description() string {
	return "Transactional outbox: persists domain events to a postgres table in the same tx as the aggregate; a worker drains the table to the event bus. Requires gorm."
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Requirements() []string {
	return []string{"gorm"}
}

func (feature) Dependencies() []string {
	return []string{
		"github.com/google/uuid",
		"gorm.io/gorm",
		"gorm.io/driver/postgres",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Domain port: the outbox row (ORM-agnostic)
		{TemplatePath: "templates/internal_domain_outbox_event.go.tmpl", OutputPath: "internal/domain/outbox/event.go"},
		{TemplatePath: "templates/internal_domain_outbox_repository.go.tmpl", OutputPath: "internal/domain/outbox/repository.go"},

		// Adapters — gorm-backed outbox repository + UoW
		{TemplatePath: "templates/internal_adapters_outbox_gorm_repository.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/outbox_repository.go", Requires: "gorm"},
		{TemplatePath: "templates/internal_adapters_outbox_gorm_uow.go.tmpl", OutputPath: "internal/adapters/outbox/gorm_uow.go", Requires: "gorm"},

		// Adapters — worker (ORM-agnostic)
		{TemplatePath: "templates/internal_adapters_outbox_worker.go.tmpl", OutputPath: "internal/adapters/outbox/worker.go"},

		// Migrations
		{TemplatePath: "templates/migrations_000002_add_outbox_events.up.sql.tmpl", OutputPath: "db/migrations/000002_add_outbox_events.up.sql"},
		{TemplatePath: "templates/migrations_000002_add_outbox_events.down.sql.tmpl", OutputPath: "db/migrations/000002_add_outbox_events.down.sql"},
	}
}
