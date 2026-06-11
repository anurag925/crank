package auth

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

func (feature) Name() string { return "auth" }
func (feature) Description() string {
	return "JWT-based authentication middleware, token issuance and refresh"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_middleware_auth.go.tmpl", OutputPath: "internal/middleware/auth.go"},
		{TemplatePath: "templates/internal_middleware_auth_test.go.tmpl", OutputPath: "internal/middleware/auth_test.go"},
		{TemplatePath: "templates/internal_service_auth.go.tmpl", OutputPath: "internal/service/auth.go"},
		{TemplatePath: "templates/internal_service_auth_test.go.tmpl", OutputPath: "internal/service/auth_test.go"},
		{TemplatePath: "templates/internal_handler_auth.go.tmpl", OutputPath: "internal/handler/auth.go"},
		{TemplatePath: "templates/internal_handler_auth_test.go.tmpl", OutputPath: "internal/handler/auth_test.go"},
		{TemplatePath: "templates/internal_model_user_auth.go.tmpl", OutputPath: "internal/model/user.go", SkipIfExists: false},
	}
}
