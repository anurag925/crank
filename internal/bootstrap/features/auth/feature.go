package auth

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

func (feature) Name() string { return "auth" }
func (feature) Description() string {
	return "JWT-based authentication: bcrypt password hashing, JWT issuance, /auth endpoints, /me protected route"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/golang-jwt/jwt/v5",
		"github.com/google/uuid",
		"golang.org/x/crypto",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Domain: extend the user aggregate with password and email value objects
		{TemplatePath: "templates/internal_domain_user_password.go.tmpl", OutputPath: "internal/domain/user/password.go"},
		{TemplatePath: "templates/internal_domain_user_email.go.tmpl", OutputPath: "internal/domain/user/email.go"},

		// Ports
		{TemplatePath: "templates/internal_ports_hasher.go.tmpl", OutputPath: "internal/ports/hasher.go"},
		{TemplatePath: "templates/internal_ports_tokenservice.go.tmpl", OutputPath: "internal/ports/tokenservice.go"},

		// Adapters: crypto + http
		{TemplatePath: "templates/internal_adapters_crypto_bcrypt_hasher.go.tmpl", OutputPath: "internal/adapters/crypto/bcrypt_hasher.go"},
		{TemplatePath: "templates/internal_adapters_crypto_jwt_token_service.go.tmpl", OutputPath: "internal/adapters/crypto/jwt_token_service.go"},
		{TemplatePath: "templates/internal_adapters_http_web_auth_handler.go.tmpl", OutputPath: "internal/adapters/http/web/auth_handler.go"},
		{TemplatePath: "templates/internal_adapters_http_web_middleware_auth.go.tmpl", OutputPath: "internal/adapters/http/web/middleware/auth.go"},
	}
}
