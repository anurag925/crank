package postgres

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

func (feature) Name() string { return "postgres" }
func (feature) Description() string {
	return "PostgreSQL connection via Bun ORM and golang-migrate migrations"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/uptrace/bun",
		"github.com/uptrace/bun/dialect/pgdialect",
		"github.com/uptrace/bun/driver/pgdriver",
		"github.com/golang-migrate/migrate/v4",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_database_postgres.go.tmpl", OutputPath: "internal/database/postgres.go"},
		{TemplatePath: "templates/internal_database_migrate.go.tmpl", OutputPath: "internal/database/migrate.go"},
		{TemplatePath: "templates/internal_database_test.go.tmpl", OutputPath: "internal/database/database_test.go"},
		{TemplatePath: "templates/internal_model_user.go.tmpl", OutputPath: "internal/model/user.go"},
		{TemplatePath: "templates/internal_repository_user.go.tmpl", OutputPath: "internal/repository/user.go"},
		{TemplatePath: "templates/migrations_000001_init.up.sql.tmpl", OutputPath: "migrations/000001_init.up.sql"},
		{TemplatePath: "templates/migrations_000001_init.down.sql.tmpl", OutputPath: "migrations/000001_init.down.sql"},
	}
}
