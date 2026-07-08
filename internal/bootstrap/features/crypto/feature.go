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
	return "AES-256-GCM authenticated encryption (Encrypt/Decrypt) backed by a config-driven secret"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string { return nil }

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_ports_cipher.go.tmpl", OutputPath: "internal/ports/cipher.go"},
		{TemplatePath: "templates/pkg_crypto_aesgcm_cipher.go.tmpl", OutputPath: "pkg/crypto/aesgcm_cipher.go"},
	}
}
