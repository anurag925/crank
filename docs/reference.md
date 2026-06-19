---
title: Reference
---

# Reference

Quick reference for commands, features, and field types.

## Command cheat sheet

```bash
crank init myapp --features=base,auth
crank init myapp --features=base,auth --use-bun
crank add redis --project ./myapp
crank list
crank tools

crank make model Order customer:string total:float
crank make repository Order
crank make service Order
crank make handler Order
crank make scaffold Order customer:string total:float --tests
crank make workflow OrderFulfillment order_id:uuid
crank make activity ChargeCard amount:float --tests
crank make migration create_orders

crank run --project ./myapp
crank dev --project ./myapp
crank build --project ./myapp
crank test -v --project ./myapp
crank gofmt --project ./myapp
crank vet --project ./myapp
crank tidy --project ./myapp
crank swag --project ./myapp
crank migrate up --project ./myapp
crank doctor --project ./myapp
```

## Features

| Feature | Summary |
| --- | --- |
| `base` | Core DDD service layout and HTTP/config/logging/validation foundation. |
| `gorm` | PostgreSQL persistence with GORM. Default ORM. |
| `bun` | PostgreSQL persistence with Bun. |
| `auth` | JWT auth and bcrypt password hashing. |
| `crypto` | AES-256-GCM helper. |
| `redis` | Redis client and cache port. |
| `mongodb` | MongoDB client. |
| `qdrant` | Qdrant vector database client. |
| `temporal` | Temporal client, worker, workflows, and activities. |
| `otel` | OpenTelemetry tracing. |
| `outbox` | Transactional outbox for domain events. |
| `views` | React SPA with Vite, embedded by the Go binary. |

## Field types

| Field type | Example | Go type |
| --- | --- | --- |
| `string` | `title:string` | `string` |
| `text` | `body:text` | `string` |
| `int` | `count:int` | `int` |
| `int64` | `total:int64` | `int64` |
| `float` | `price:float` | `float64` |
| `float64` | `amount:float64` | `float64` |
| `bool` | `active:bool` | `bool` |
| `time` | `published_at:time` | `time.Time` |
| `uuid` | `customer_id:uuid` | `string` with UUID validation |
| `email` | `email:email` | `string` with email validation |

## Tool requirements

| Tool | Requires feature |
| --- | --- |
| `migrate` | `gorm` or `bun` |
| `swag` | none |
| `build` | none |
| `run` | none |
| `dev` | none |
| `test` | none |
| `gofmt` | none |
| `vet` | none |
| `tidy` | none |
| `doctor` | none |

## Important files in generated projects

| Path | Purpose |
| --- | --- |
| `.crank.yaml` | Manifest used by `crank`. |
| `configs/config.yaml` | Safe configuration defaults. |
| `.env.example` | Local environment template. |
| `.env` | Local secrets; git-ignored. |
| `cmd/server/main.go` | Composition root and server startup. |
| `internal/domain` | Pure domain model. |
| `internal/application` | Command/query use cases. |
| `internal/adapters` | HTTP, persistence, event bus, unit of work, and other infrastructure. |
| `internal/config` | Runtime config loading. |
| `internal/validator` | Request validation. |
| `migrations` | SQL migration files. |
| `docs` | Generated Swagger output for the application. |
