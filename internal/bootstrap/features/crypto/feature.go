package crypto

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

func (feature) Name() string { return "crypto" }
func (feature) Description() string {
	return "AES-256-GCM encryption helpers (Encrypt/Decrypt) backed by a config-driven secret"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return nil // uses only stdlib
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_crypto_crypto.go.tmpl", OutputPath: "internal/crypto/crypto.go"},
		{TemplatePath: "templates/internal_crypto_crypto_test.go.tmpl", OutputPath: "internal/crypto/crypto_test.go"},
	}
}
