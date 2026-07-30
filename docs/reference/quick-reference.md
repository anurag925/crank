---
title: Quick reference
type: reference
---

# Quick Reference

## Command cheat sheet

```bash
crank init myapp --features=base,auth
crank add redis --project ./myapp
crank make skill --project ./myapp
crank list
crank tools

crank make model Order customer:string total:float
crank make repository Order
crank make service Order
crank make handler Order
crank make scaffold Order customer:string total:float --tests
crank make workflow OrderFulfillment order_id:uuid
crank make activity ChargeCard amount:float --tests

crank run --project ./myapp
crank dev --project ./myapp
crank build --project ./myapp
crank test -v --project ./myapp
crank gofmt --project ./myapp
crank vet --project ./myapp
crank tidy --project ./myapp
crank make swag --project ./myapp
crank migrate up --project ./myapp
crank doctor --project ./myapp
crank make seed User --count 20
crank make seed up
```

## Features

| Feature | Summary |
| --- | --- |
| `base` | Core DDD service layout, Echo v5, config, validation, logging, `uuid.UUID` domain IDs, `TxRepositories` UoW |
| `gorm` | PostgreSQL with GORM — row DTO pattern, database factory, migrations |
| `auth` | JWT auth with token revocation, bcrypt in `pkg/crypto/`, `/auth/logout` |
| `audit` | Audit trail: persists domain events to DB, queryable by entity |
| `crypto` | AES-256-GCM cipher in `pkg/crypto/` |
| `redis` | Redis cache adapter in `internal/adapters/cache/redis/` |
| `mongodb` | MongoDB client in `internal/adapters/persistence/mongodb/` |
| `qdrant` | Qdrant port + gRPC/HTTP clients in `internal/adapters/persistence/qdrant/` |
| `temporal` | Temporal client, worker, workflows, activities |
| `otel` | OpenTelemetry tracing |
| `outbox` | Transactional outbox with TxRepositories-backed UoW |
| `views` | React SPA with Vite, embedded by Go binary |

## Important file paths

| Path | Purpose |
| --- | --- |
| `.crank.yaml` | Project manifest |
| `configs/config.yaml` | Safe config defaults |
| `cmd/server/main.go` | Composition root |
| `internal/domain/{resource}/` | Aggregates, events, repository ports |
| `internal/application/{resource}/` | CQRS command/query handlers |
| `internal/application/uow/` | UnitOfWork + TxRepositories port |
| `internal/adapters/http/web/v1/` | Versioned HTTP handlers |
| `internal/adapters/http/web/api/` | `api.Error` envelope |
| `internal/adapters/http/web/server.go` | Echo server with HTTP error handler and exported `EchoBinder` |
| `internal/adapters/persistence/gorm/` | GORM repos (row DTO pattern) |
| `internal/adapters/outbox/` | Transactional UoW + worker |
| `internal/adapters/auth/jwt/` | JWT token service |
| `pkg/crypto/` | bcrypt hasher + AES-256-GCM cipher |
| `pkg/logging/` | slog helpers with redaction + context enrichment |
| `migrations/` | SQL migration pairs |
