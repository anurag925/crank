package redis

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

func (feature) Name() string { return "redis" }
func (feature) Description() string {
	return "Redis client (session storage, caching, rate limiting) — placeholder"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/redis/go-redis/v9",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_redis_client.go.tmpl", OutputPath: "internal/redis/client.go"},
		{TemplatePath: "templates/internal_redis_client_test.go.tmpl", OutputPath: "internal/redis/client_test.go"},
	}
}
