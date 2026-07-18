package base

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

func (feature) Name() string { return "base" }
func (feature) Description() string {
	return "DDD layout: domain aggregates, application command/query handlers, HTTP and in-memory adapters"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/caarlos0/env/v11",
		"github.com/joho/godotenv",
		"github.com/labstack/echo/v5",
		"github.com/spf13/viper",
		"github.com/go-playground/validator/v10",
		"github.com/google/uuid",
		"github.com/stretchr/testify",
		"github.com/swaggo/echo-swagger/v2",
		"github.com/swaggo/files",
		"github.com/swaggo/swag",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Composition root + scaffolding metadata
		{TemplatePath: "templates/cmd_server_main.go.tmpl", OutputPath: "cmd/server/main.go"},
		{TemplatePath: "templates/docs_docs.go.tmpl", OutputPath: "docs/docs.go", SkipIfExists: true},

		// Configuration
		{TemplatePath: "templates/internal_config_config.go.tmpl", OutputPath: "internal/config/config.go"},

		// HTTP API envelope
		{TemplatePath: "templates/internal_adapters_http_web_api_error.go.tmpl", OutputPath: "internal/adapters/http/web/api/error.go"},

		// Domain layer — shared kernel
		{TemplatePath: "templates/internal_domain_shared_events.go.tmpl", OutputPath: "internal/domain/shared/events.go"},
		{TemplatePath: "templates/internal_domain_shared_registry.go.tmpl", OutputPath: "internal/domain/shared/registry.go"},

		// Application layer — Unit of Work abstraction
		{TemplatePath: "templates/internal_application_uow_uow.go.tmpl", OutputPath: "internal/application/uow/uow.go"},

		// Ports (cross-cutting interfaces)
		{TemplatePath: "templates/internal_ports_eventbus.go.tmpl", OutputPath: "internal/ports/eventbus.go"},

		// Adapters — eventbus, persistence, http, uow
		{TemplatePath: "templates/internal_adapters_eventbus_in_memory_eventbus.go.tmpl", OutputPath: "internal/adapters/eventbus/in_memory_eventbus.go"},
		{TemplatePath: "templates/internal_adapters_http_web_server.go.tmpl", OutputPath: "internal/adapters/http/web/server.go"},
		{TemplatePath: "templates/internal_adapters_uow_in_memory_uow.go.tmpl", OutputPath: "internal/adapters/uow/in_memory_uow.go"},
		{TemplatePath: "templates/internal_adapters_http_web_v1_routes.go.tmpl", OutputPath: "internal/adapters/http/web/v1/routes.go"},
		{TemplatePath: "templates/internal_adapters_http_web_middleware_logging.go.tmpl", OutputPath: "internal/adapters/http/web/middleware/logging.go"},

		// Validator
		{TemplatePath: "templates/internal_validator_validator.go.tmpl", OutputPath: "internal/validator/validator.go"},
		{TemplatePath: "templates/internal_validator_errors.go.tmpl", OutputPath: "internal/validator/errors.go"},
		{TemplatePath: "templates/internal_validator_validator_test.go.tmpl", OutputPath: "internal/validator/validator_test.go"},

		// Logging
		{TemplatePath: "templates/pkg_logging_logger.go.tmpl", OutputPath: "pkg/logging/logger.go"},
		{TemplatePath: "templates/pkg_logging_redactor.go.tmpl", OutputPath: "pkg/logging/redactor.go"},

		// Project metadata
		{TemplatePath: "templates/config.yaml.tmpl", OutputPath: "configs/config.yaml"},
		{TemplatePath: "templates/.env.example.tmpl", OutputPath: ".env.example"},
		{TemplatePath: "templates/Makefile.tmpl", OutputPath: "Makefile"},
		{TemplatePath: "templates/air.toml.tmpl", OutputPath: ".air.toml"},
		{TemplatePath: "templates/Dockerfile.tmpl", OutputPath: "Dockerfile"},
		{TemplatePath: "templates/.gitignore.tmpl", OutputPath: ".gitignore"},
		{TemplatePath: "templates/go.mod.tmpl", OutputPath: "go.mod"},
		{TemplatePath: "templates/README.md.tmpl", OutputPath: "README.md"},
		{TemplatePath: "templates/.crank.yaml.tmpl", OutputPath: ".crank.yaml"},
		{TemplatePath: "templates/AGENTS.md.tmpl", OutputPath: "AGENTS.md", SkipIfExists: true},
		{TemplatePath: "templates/agents_skills_crank_project_SKILL.md.tmpl", OutputPath: ".agents/skills/crank-project/SKILL.md", SkipIfExists: true},
	}
}
