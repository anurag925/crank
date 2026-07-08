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
	return "JWT-based authentication: bcrypt password hashing, JWT issuance with revocation, /auth endpoints, /me protected route"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/golang-jwt/jwt/v5",
		"github.com/google/uuid",
		"golang.org/x/crypto",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Domain: extend the user aggregate with password and email value objects
		{TemplatePath: "templates/internal_domain_user_password.go.tmpl", OutputPath: "internal/domain/user/password.go"},
		{TemplatePath: "templates/internal_domain_user_email.go.tmpl", OutputPath: "internal/domain/user/email.go"},

		// Ports
		{TemplatePath: "templates/internal_ports_hasher.go.tmpl", OutputPath: "internal/ports/hasher.go"},
		{TemplatePath: "templates/internal_ports_tokenservice.go.tmpl", OutputPath: "internal/ports/tokenservice.go"},
		{TemplatePath: "templates/internal_ports_tokendenylist.go.tmpl", OutputPath: "internal/ports/tokendenylist.go"},

		// Adapters: bcrypt hasher lives in pkg/crypto (shared utility)
		{TemplatePath: "templates/pkg_crypto_bcrypt_hasher.go.tmpl", OutputPath: "pkg/crypto/bcrypt_hasher.go"},

		// Adapters: JWT token service
		{TemplatePath: "templates/internal_adapters_auth_jwt_token_service.go.tmpl", OutputPath: "internal/adapters/auth/jwt/token_service.go"},

		// Adapters: token denylist (GORM-backed, only when gorm is enabled)
		{TemplatePath: "templates/internal_adapters_persistence_gorm_token_denylist.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/token_denylist.go", Requires: "gorm"},

		// HTTP: auth handler + JWT middleware
		{TemplatePath: "templates/internal_adapters_http_web_auth_handler.go.tmpl", OutputPath: "internal/adapters/http/web/auth_handler.go"},
		{TemplatePath: "templates/internal_adapters_http_web_middleware_auth.go.tmpl", OutputPath: "internal/adapters/http/web/middleware/auth.go"},
	}
}
