---
title: Redis Feature
---

# Redis feature

The `redis` feature adds a **Redis cache** adapter wired into the composition root. Use it for session storage, caching, rate limiting, or distributed coordination.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/cache.go` | `Get` / `Set` / `Del` cache interface |
| `internal/adapters/cache/redis/client.go` | Redis client connection using go-redis |
| `internal/adapters/cache/redis/cache.go` | `CacheAdapter` implementing `ports.Cache` |

The adapter is nil-safe — graceful behavior when Redis is unavailable.

## Notes

- Connection settings in `configs/config.yaml` under the `redis` section.
- For local dev: `docker run -p 6379:6379 redis`.
- Gracefully degrades if unreachable.
