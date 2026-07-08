# crank — Golang Backend CLI - Specification

## Concept & Vision

A modular CLI tool that scaffolds production-ready Go backend applications using DDD patterns and wraps common development tools as subcommands. Generated projects follow a strict layered architecture: domain aggregates (uuid.UUID IDs, private fields, row DTOs), CQRS application handlers, v1 HTTP handlers with self-scoping, and a TxRepositories-based Unit of Work that never leaks ORM types into the application layer.

## Architecture Overview

```
crank/
├── cmd/
│   └── crank/
│       └── main.go                  # CLI entry point, feature/tool imports
├── internal/
│   ├── bootstrap/
│   │   ├── generator.go             # Core code generation (Generate, Add, Update)
│   │   ├── feature.go               # Feature interface, FileMapping, Registry
│   │   ├── tool.go                  # Tool interface, InProcessTool, ToolRegistry
│   │   ├── context.go               # Template context with Has()/Require()
│   │   ├── manifest.go              # .crank.yaml read/write
│   │   ├── project.go               # LoadProjectInfo() — public manifest reader
│   │   ├── commands/                # Cobra commands (init, add, update, update-skill, list, make, tools)
│   │   ├── scaffold/                # crank make code generators + templates
│   │   ├── tools/                   # Tool wrappers (build, run, test, dev, swag, migrate, doctor, etc.)
│   │   └── features/                # Feature modules (self-registering via init())
│   │       ├── base/                # Foundation: Echo v5, Viper, slog, validator, DDD layout
│   │       ├── auth/                # JWT + bcrypt + token denylist + revocation
│   │       ├── crypto/              # AES-256-GCM cipher
│   │       ├── bun/                 # Bun ORM + migrations
│   │       ├── gorm/                # GORM (default ORM) + migrations
│   │       ├── redis/               # Redis cache adapter
│   │       ├── mongodb/             # MongoDB client
│   │       ├── qdrant/              # Qdrant vector DB (gRPC + HTTP clients)
│   │       ├── temporal/            # Temporal workflows, activities, worker
│   │       ├── otel/                # OpenTelemetry tracing
│   │       ├── outbox/              # Transactional outbox (requires bun or gorm)
│   │       ├── views/               # React SPA with Vite
│   │       └── audit/               # Audit trail (domain event persistence)
│   └── utils/
│       ├── fileutil.go              # EnsureDir, WriteFile, PathExists
│       └── exec.go                  # FindBinary, RunExternal
└── go.mod
```

## Generated Project Architecture

```
myapp/
├── cmd/server/main.go               # Composition root — DDD wiring
├── configs/config.yaml              # Non-secret defaults
├── internal/
│   ├── adapters/
│   │   ├── eventbus/                # In-memory event bus
│   │   ├── http/web/
│   │   │   ├── server.go            # Echo server + EchoBinder + HTTP error handler
│   │   │   ├── api/error.go         # api.Error envelope
│   │   │   ├── v1/                  # Versioned handlers at /api/v1
│   │   │   └── middleware/          # Logging, JWT auth, tracing
│   │   ├── persistence/
│   │   │   ├── memory/              # In-memory repos (always generated)
│   │   │   ├── gorm/                # GORM repos — row DTO pattern
│   │   │   └── bun/                 # Bun repos — row DTO pattern
│   │   ├── uow/                     # In-memory UoW with TxRepositories
│   │   ├── outbox/                  # Transactional UoW + worker
│   │   ├── auth/jwt/                # JWT token service
│   │   ├── cache/redis/             # Redis cache adapter
│   │   ├── audit/                   # Audit logger
│   │   └── telemetry/               # OpenTelemetry setup
│   ├── application/
│   │   ├── user/                    # CQRS command/query handlers
│   │   └── uow/                     # UnitOfWork + TxRepositories port
│   ├── config/                      # Viper + caarlos0/env loading
│   ├── domain/                      # Pure domain (uuid.UUID IDs, private fields)
│   ├── ports/                       # Cross-cutting interfaces
│   └── validator/                   # go-playground/validator with EchoBinder
├── pkg/
│   ├── logging/                     # slog + redaction + context enrichment
│   └── crypto/                      # bcrypt hasher + AES-256-GCM cipher
├── migrations/                      # SQL migration pairs
├── .crank.yaml                      # Project manifest
├── .agents/skills/crank-project/    # Agent skill file
├── AGENTS.md
├── Dockerfile
└── Makefile
```

## Key Design Decisions

| Decision | Implementation |
|---|---|
| **ID type** | `uuid.UUID` from `github.com/google/uuid` — all aggregates, no custom types |
| **Aggregate purity** | Private fields with getters, zero ORM/JSON tags. `Rehydrate()` for persistence |
| **Row DTO pattern** | Private `{name}Row` struct with `toAggregate()` / `rowFromAggregate()` — ORM tags never touch aggregates |
| **Unit of Work** | `uow.UnitOfWork` + `uow.TxRepositories` in `application/uow/`. Save closures call `repos.Users().Save(ctx, u)` — zero GORM imports |
| **HTTP layout** | Versioned `web/v1/` package at `/api/v1`, `api.Error` envelope, self-scoped user endpoints |
| **Auth** | JWT at `adapters/auth/jwt/`, bcrypt at `pkg/crypto/`, token denylist with revocation |
| **Graceful degradation** | Optional services (redis, qdrant, temporal) warn + skip if unavailable |
| **Configuration** | env vars > .env > configs/config.yaml. Secrets tagged `env:"..."` |

## Technical Stack

| Component | Choice |
|---|---|
| HTTP Framework | Echo v5 |
| ORM | GORM (default) or Bun |
| Config | Viper + caarlos0/env |
| Validation | go-playground/validator v10 |
| API Docs | swaggo/swag + echo-swagger v2 |
| Logging | log/slog with three-layer handler stack |
| Migrations | golang-migrate v4 |
| Live Reload | Air |
| IDs | github.com/google/uuid |
| JWT | golang-jwt/jwt v5 |
| Password | golang.org/x/crypto (bcrypt) |
| Encryption | stdlib crypto/aes, crypto/cipher (AES-256-GCM) |

## CLI Commands

```bash
# Project lifecycle
crank init myapp --features=base,auth
crank add redis --project ./myapp
crank update-skill --project ./myapp    # refresh agent SKILL.md
crank list
crank tools

# Code generation
crank make scaffold Book title:string author:string price:float --tests
crank make handler Product --only
crank make migration create_orders

# Tool wrappers
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
