---
title: Generated Project Structure
---

# Generated project structure

A generated project follows a layered, Domain-Driven layout.

```text
myapp/
├── cmd/
│   └── server/
│       └── main.go
├── configs/
│   └── config.yaml
├── internal/
│   ├── adapters/
│   │   ├── eventbus/
│   │   ├── http/web/
│   │   │   ├── server.go
│   │   │   ├── api/error.go
│   │   │   ├── v1/
│   │   │   │   ├── routes.go
│   │   │   │   └── user_handler.go
│   │   │   └── middleware/
│   │   ├── persistence/
│   │   │   ├── memory/
│   │   │   └── gorm/
│   │   └── uow/
│   ├── application/
│   │   ├── user/
│   │   └── uow/
│   ├── config/
│   ├── domain/
│   │   ├── shared/
│   │   └── user/
│   ├── ports/
│   └── validator/
├── pkg/
│   ├── logging/
│   └── crypto/
├── migrations/
├── docs/
├── .agents/
│   └── skills/
│       └── crank-project/
│           └── SKILL.md
├── .crank.yaml
├── AGENTS.md
├── .env.example
├── Dockerfile
└── Makefile
```

## `cmd/server`

Application entry point and composition root. Wires all concrete types — repos, UoW, event bus, token service, handlers. Mounts v1 routes at `/api/v1`, configures CORS, sets up graceful shutdown.

## `internal/domain`

Pure domain code. Each resource lives in its own package:

```text
internal/domain/book/
├── book.go           # Aggregate root with uuid.UUID ID, getters, Rehydrate()
├── events.go         # Domain events
├── errors.go         # Sentinel errors
└── repository.go     # Repository port interface
```

All aggregates use `uuid.UUID` for identity. Fields are private with exported getters. No ORM/JSON tags on aggregates.

## `internal/application`

CQRS use cases per resource plus the UnitOfWork port:

```text
internal/application/book/
├── commands.go
├── command_handler.go   # Uses repos.Users().Save() through TxRepositories
├── queries.go
└── query_handler.go     # Uses uuid.Parse for ID conversion

internal/application/uow/
└── uow.go               # UnitOfWork + TxRepositories port
```

## `internal/adapters/http/web/v1/`

Versioned HTTP handlers using `api.Error` with self-scoped user endpoints.

## `internal/adapters/persistence/gorm/`

GORM repositories with row DTO pattern. Private `{name}Row` struct with `toAggregate()` via `Rehydrate()` and `rowFromAggregate()`.

## `internal/ports/`

Cross-cutting interfaces: EventBus, TokenService, TokenDenylist, Hasher, Cache, Cipher, etc.

## `pkg/logging/`

Three-layer slog handler stack: JSON → redaction → context enrichment. Auto-injects `request_id` and `user_id`.

## `pkg/crypto/`

Shared bcrypt hasher (auth) and AES-256-GCM cipher (crypto).

## Agent guidance

Use `crank update-skill` to refresh `.agents/skills/crank-project/SKILL.md` with the latest conventions.
