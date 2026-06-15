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
		"github.com/labstack/echo/v4",
		"github.com/spf13/viper",
		"github.com/go-playground/validator/v10",
		"github.com/stretchr/testify",
		"github.com/swaggo/echo-swagger",
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

		// Cross-layer transport envelope
		{TemplatePath: "templates/internal_model_model.go.tmpl", OutputPath: "internal/model/model.go"},

		// Domain layer — shared kernel + seed aggregate
		{TemplatePath: "templates/internal_domain_shared_events.go.tmpl", OutputPath: "internal/domain/shared/events.go"},
		{TemplatePath: "templates/internal_domain_shared_registry.go.tmpl", OutputPath: "internal/domain/shared/registry.go"},
		{TemplatePath: "templates/internal_domain_user_user.go.tmpl", OutputPath: "internal/domain/user/user.go"},
		{TemplatePath: "templates/internal_domain_user_id.go.tmpl", OutputPath: "internal/domain/user/user_id.go"},
		{TemplatePath: "templates/internal_domain_user_events.go.tmpl", OutputPath: "internal/domain/user/events.go"},
		{TemplatePath: "templates/internal_domain_user_errors.go.tmpl", OutputPath: "internal/domain/user/errors.go"},
		{TemplatePath: "templates/internal_domain_user_repository.go.tmpl", OutputPath: "internal/domain/user/repository.go"},

		// Application layer
		{TemplatePath: "templates/internal_application_user_commands.go.tmpl", OutputPath: "internal/application/user/commands.go"},
		{TemplatePath: "templates/internal_application_user_command_handler.go.tmpl", OutputPath: "internal/application/user/command_handler.go"},
		{TemplatePath: "templates/internal_application_user_queries.go.tmpl", OutputPath: "internal/application/user/queries.go"},
		{TemplatePath: "templates/internal_application_user_query_handler.go.tmpl", OutputPath: "internal/application/user/query_handler.go"},

		// Ports (cross-cutting interfaces)
		{TemplatePath: "templates/internal_ports_eventbus.go.tmpl", OutputPath: "internal/ports/eventbus.go"},
		{TemplatePath: "templates/internal_ports_uow.go.tmpl", OutputPath: "internal/ports/uow.go"},

		// Adapters — eventbus, persistence, http, uow
		{TemplatePath: "templates/internal_adapters_eventbus_in_memory_eventbus.go.tmpl", OutputPath: "internal/adapters/eventbus/in_memory_eventbus.go"},
		{TemplatePath: "templates/internal_adapters_persistence_memory_user_repository.go.tmpl", OutputPath: "internal/adapters/persistence/memory/user_repository.go"},
		{TemplatePath: "templates/internal_adapters_http_web_server.go.tmpl", OutputPath: "internal/adapters/http/web/server.go"},
		{TemplatePath: "templates/internal_adapters_uow_in_memory_uow.go.tmpl", OutputPath: "internal/adapters/uow/in_memory_uow.go"},
		{TemplatePath: "templates/internal_adapters_http_web_routes.go.tmpl", OutputPath: "internal/adapters/http/web/routes.go"},
		{TemplatePath: "templates/internal_adapters_http_web_user_handler.go.tmpl", OutputPath: "internal/adapters/http/web/user_handler.go"},
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
	}
}
