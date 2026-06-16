package gorm

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

func (feature) Name() string { return "gorm" }
func (feature) Description() string {
	return "PostgreSQL persistence backed by GORM (gorm.io): user repository, db factory, golang-migrate migrations. This is the default ORM."
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"gorm.io/gorm",
		"gorm.io/driver/postgres",
		"github.com/golang-migrate/migrate/v4",
		"github.com/google/uuid",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Postgres persistence adapters
		{TemplatePath: "templates/internal_adapters_persistence_gorm_db.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/db.go"},
		{TemplatePath: "templates/internal_adapters_persistence_gorm_migrate.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/migrate.go"},
		{TemplatePath: "templates/internal_adapters_persistence_gorm_user_repository.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/user_repository.go"},
		// Migrations
		{TemplatePath: "templates/migrations_000001_init.up.sql.tmpl", OutputPath: "migrations/000001_init.up.sql"},
		{TemplatePath: "templates/migrations_000001_init.down.sql.tmpl", OutputPath: "migrations/000001_init.down.sql"},
	}
}
