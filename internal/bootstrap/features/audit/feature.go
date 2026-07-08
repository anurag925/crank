package audit

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

func (feature) Name() string { return "audit" }
func (feature) Description() string {
	return "Audit trail: persists domain events to a database table, queryable by entity type and ID"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/google/uuid",
	}
}

func (feature) Requirements() []string { return []string{"bun", "gorm"} }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_domain_audit_event.go.tmpl", OutputPath: "internal/domain/audit/event.go"},
		{TemplatePath: "templates/internal_domain_audit_repository.go.tmpl", OutputPath: "internal/domain/audit/repository.go"},
		{TemplatePath: "templates/internal_ports_audit.go.tmpl", OutputPath: "internal/ports/audit.go"},
		{TemplatePath: "templates/internal_adapters_persistence_bun_audit_repository.go.tmpl", OutputPath: "internal/adapters/persistence/bun/audit_repository.go", Requires: "bun"},
		{TemplatePath: "templates/internal_adapters_persistence_gorm_audit_repository.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/audit_repository.go", Requires: "gorm"},
		{TemplatePath: "templates/internal_adapters_audit_logger.go.tmpl", OutputPath: "internal/adapters/audit/logger.go"},
		{TemplatePath: "templates/internal_application_audit_query_handler.go.tmpl", OutputPath: "internal/application/audit/query_handler.go"},
		{TemplatePath: "templates/internal_adapters_http_web_audit_handler.go.tmpl", OutputPath: "internal/adapters/http/web/v1/audit_handler.go"},
		{TemplatePath: "templates/migrations_000003_add_audit_events.up.sql.tmpl", OutputPath: "migrations/000003_add_audit_events.up.sql"},
		{TemplatePath: "templates/migrations_000003_add_audit_events.down.sql.tmpl", OutputPath: "migrations/000003_add_audit_events.down.sql"},
	}
}
