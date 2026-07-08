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
	return "Redis Cache port + go-redis client (session storage, caching, rate limiting)"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/redis/go-redis/v9",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_ports_cache.go.tmpl", OutputPath: "internal/ports/cache.go"},
		{TemplatePath: "templates/internal_adapters_cache_redis_client.go.tmpl", OutputPath: "internal/adapters/cache/redis/client.go"},
		{TemplatePath: "templates/internal_adapters_platform_redis.go.tmpl", OutputPath: "internal/adapters/cache/redis/cache.go"},
	}
}
