---
title: Redis Feature
---

# Redis feature

The `redis` feature adds a **Redis cache** client wired into the composition root. Use it for session storage, caching, rate limiting, or distributed coordination.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/cache.go` | `Get` / `Set` / `Delete` cache interface |
| `internal/adapters/cache/redis/client.go` | Redis client connection using go-redis |

The client is wired in the composition root as `rdb` and can be accessed via the `Cache` port interface.

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [go-redis v9](https://github.com/redis/go-redis) | Redis client for Go | [redis.uptrace.dev](https://redis.uptrace.dev) |
| Redis | In-memory data structure store | [redis.io/docs](https://redis.io/docs) |

## Learning resources

- **go-redis documentation** — [redis.uptrace.dev](https://redis.uptrace.dev) (commands, pipelines, pub/sub, sentinel, cluster)
- **Redis commands reference** — [redis.io/commands](https://redis.io/commands)
- **Redis official docs** — [redis.io/docs](https://redis.io/docs) (data types, persistence, replication)

## Notes

- Redis connection settings are configurable in `configs/config.yaml` under the `redis` section.
- The `Cache` port interface allows swapping in a different cache backend without changing application code.
- For local development, start Redis with Docker: `docker run -p 6379:6379 redis` or use the memory adapter.
