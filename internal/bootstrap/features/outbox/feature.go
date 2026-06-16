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
	return "Transactional outbox: persists domain events to a postgres table in the same tx as the aggregate; a worker drains the table to the event bus. Requires bun or gorm."
}
func (feature) Templates() embed.FS { return tmpls }

// Requirements returns both bun and gorm as valid ORM backends. The
// generator treats this pair as alternatives — at least one must be
// present but both are not required.
func (feature) Requirements() []string {
	return []string{"bun", "gorm"}
}

func (feature) Dependencies() []string {
	return []string{
		"github.com/google/uuid",
		// Both ORM deps are listed so the feature works with either. The
		// unused deps are harmless (they get added to go.sum but not imported).
		"github.com/uptrace/bun",
		"github.com/uptrace/bun/dialect/pgdialect",
		"github.com/uptrace/bun/driver/pgdriver",
		"gorm.io/gorm",
		"gorm.io/driver/postgres",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Domain port: the outbox row (ORM-agnostic)
		{TemplatePath: "templates/internal_domain_outbox_event.go.tmpl", OutputPath: "internal/domain/outbox/event.go"},
		{TemplatePath: "templates/internal_domain_outbox_repository.go.tmpl", OutputPath: "internal/domain/outbox/repository.go"},

		// Adapters — bun-backed outbox repository + UoW (only when bun is enabled)
		{TemplatePath: "templates/internal_adapters_outbox_bun_repository.go.tmpl", OutputPath: "internal/adapters/persistence/bun/outbox_repository.go", Requires: "bun"},
		{TemplatePath: "templates/internal_adapters_outbox_bun_uow.go.tmpl", OutputPath: "internal/adapters/outbox/bun_uow.go", Requires: "bun"},

		// Adapters — gorm-backed outbox repository + UoW (only when gorm is enabled)
		{TemplatePath: "templates/internal_adapters_outbox_gorm_repository.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/outbox_repository.go", Requires: "gorm"},
		{TemplatePath: "templates/internal_adapters_outbox_gorm_uow.go.tmpl", OutputPath: "internal/adapters/outbox/gorm_uow.go", Requires: "gorm"},

		// Adapters — worker (ORM-agnostic)
		{TemplatePath: "templates/internal_adapters_outbox_worker.go.tmpl", OutputPath: "internal/adapters/outbox/worker.go"},

		// Migrations
		{TemplatePath: "templates/migrations_000002_add_outbox_events.up.sql.tmpl", OutputPath: "migrations/000002_add_outbox_events.up.sql"},
		{TemplatePath: "templates/migrations_000002_add_outbox_events.down.sql.tmpl", OutputPath: "migrations/000002_add_outbox_events.down.sql"},
	}
}
