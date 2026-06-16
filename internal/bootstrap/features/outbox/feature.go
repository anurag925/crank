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
	return "Transactional outbox: persists domain events to a postgres table in the same tx as the aggregate; a worker drains the table to the event bus"
}
func (feature) Templates() embed.FS { return tmpls }

// Requirements declares other features that must be installed for outbox
// to work. The outbox needs a transactional store, which today is provided
// by the bun feature (the outbox's postgres UoW talks to a *bun.DB). Without
// it, the outbox would silently degrade to the in-memory UnitOfWork the
// base feature already provides. GORM support is on the roadmap.
func (feature) Requirements() []string {
	return []string{"bun"}
}

func (feature) Dependencies() []string {
	return []string{
		"github.com/google/uuid",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Domain port: the outbox row
		{TemplatePath: "templates/internal_domain_outbox_event.go.tmpl", OutputPath: "internal/domain/outbox/event.go"},
		{TemplatePath: "templates/internal_domain_outbox_repository.go.tmpl", OutputPath: "internal/domain/outbox/repository.go"},

		// Adapters — bun-backed outbox repository + UoW + worker
		{TemplatePath: "templates/internal_adapters_outbox_bun_repository.go.tmpl", OutputPath: "internal/adapters/persistence/bun/outbox_repository.go"},
		{TemplatePath: "templates/internal_adapters_outbox_bun_uow.go.tmpl", OutputPath: "internal/adapters/outbox/bun_uow.go"},
		{TemplatePath: "templates/internal_adapters_outbox_worker.go.tmpl", OutputPath: "internal/adapters/outbox/worker.go"},

		// Migrations
		{TemplatePath: "templates/migrations_000002_add_outbox_events.up.sql.tmpl", OutputPath: "migrations/000002_add_outbox_events.up.sql"},
		{TemplatePath: "templates/migrations_000002_add_outbox_events.down.sql.tmpl", OutputPath: "migrations/000002_add_outbox_events.down.sql"},
	}
}
