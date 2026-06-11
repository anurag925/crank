package base

import (
	"embed"

	"github.com/anurag925/rev/internal/bootstrap"
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
	return "Echo v4 HTTP server, Viper configuration, structured slog logging"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/cmd_server_main.go.tmpl", OutputPath: "cmd/server/main.go"},
		{TemplatePath: "templates/internal_config_config.go.tmpl", OutputPath: "internal/config/config.go"},
		{TemplatePath: "templates/internal_handler_handler.go.tmpl", OutputPath: "internal/handler/handler.go"},
		{TemplatePath: "templates/internal_handler_handler_test.go.tmpl", OutputPath: "internal/handler/handler_test.go"},
		{TemplatePath: "templates/internal_handler_user.go.tmpl", OutputPath: "internal/handler/user.go"},
		{TemplatePath: "templates/internal_handler_user_test.go.tmpl", OutputPath: "internal/handler/user_test.go"},
		{TemplatePath: "templates/internal_middleware_logging.go.tmpl", OutputPath: "internal/middleware/logging.go"},
		{TemplatePath: "templates/internal_logging_logger.go.tmpl", OutputPath: "internal/logging/logger.go"},
		{TemplatePath: "templates/internal_logging_redactor.go.tmpl", OutputPath: "internal/logging/redactor.go"},
		{TemplatePath: "templates/internal_middleware_logging_test.go.tmpl", OutputPath: "internal/middleware/logging_test.go"},
		{TemplatePath: "templates/internal_validator_validator.go.tmpl", OutputPath: "internal/validator/validator.go"},
		{TemplatePath: "templates/internal_validator_errors.go.tmpl", OutputPath: "internal/validator/errors.go"},
		{TemplatePath: "templates/internal_validator_validator_test.go.tmpl", OutputPath: "internal/validator/validator_test.go"},
		{TemplatePath: "templates/internal_model_user.go.tmpl", OutputPath: "internal/model/user.go"},
		{TemplatePath: "templates/internal_repository_user.go.tmpl", OutputPath: "internal/repository/user.go"},
		{TemplatePath: "templates/internal_repository_user_test.go.tmpl", OutputPath: "internal/repository/user_test.go"},
		{TemplatePath: "templates/internal_service_user.go.tmpl", OutputPath: "internal/service/user.go"},
		{TemplatePath: "templates/internal_service_user_test.go.tmpl", OutputPath: "internal/service/user_test.go"},
		{TemplatePath: "templates/config.yaml.tmpl", OutputPath: "configs/config.yaml"},
		{TemplatePath: "templates/.env.example.tmpl", OutputPath: ".env.example"},
		{TemplatePath: "templates/Makefile.tmpl", OutputPath: "Makefile"},
		{TemplatePath: "templates/air.toml.tmpl", OutputPath: "air.toml"},
		{TemplatePath: "templates/Dockerfile.tmpl", OutputPath: "Dockerfile"},
		{TemplatePath: "templates/.gitignore.tmpl", OutputPath: ".gitignore"},
		{TemplatePath: "templates/go.mod.tmpl", OutputPath: "go.mod"},
		{TemplatePath: "templates/README.md.tmpl", OutputPath: "README.md"},
		{TemplatePath: "templates/.bootstrap.yaml.tmpl", OutputPath: ".bootstrap.yaml"},
	}
}
