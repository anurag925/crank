---
title: Navigating the Generated Application
---

# Navigating the generated application

<p align="center">
  After <code>crank init</code>, you have a fully wired Go backend service.
  <br/>
  This page explains every layer — where things live, why they are there,
  how they connect, and how to extend them.
</p>

<p align="center">
  <img alt="Go version" src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&amp;logoColor=white" />
  <img alt="Echo v5" src="https://img.shields.io/badge/Echo-v5-5C2D91?logo=go&amp;logoColor=white" />
  <img alt="DDD" src="https://img.shields.io/badge/Architecture-DDD-blueviolet" />
  <img alt="CQRS" src="https://img.shields.io/badge/Pattern-CQRS-brightgreen" />
  <img alt="License" src="https://img.shields.io/badge/License-MIT-yellow" />
</p>

---

## Architecture overview

The generated project follows a **Domain-Driven Design** layered architecture.

### Key principles

| Principle | How it's enforced |
|-----------|------------------|
| **Domain is pure Go** | Zero framework imports — no HTTP, no DB drivers, no serialization tags on aggregates |
| **Dependencies point inward** | Adapters implement interfaces defined by the domain |
| **Application coordinates** | Handlers load aggregates, call domain methods, route persistence through ports |
| **Composition root** | `cmd/server/main.go` is the only place concrete types meet interfaces |

### Directory layout

```
myapp/
├── cmd/server/                     ← Entry point + composition root
├── configs/                        ← Config defaults (committed)
├── internal/
│   ├── adapters/
│   │   ├── eventbus/               ← In-process event bus
│   │   ├── http/web/
│   │   │   ├── server.go           ← Echo server + EchoBinder + HTTP error handler
│   │   │   ├── v1/                 ← Versioned handlers at /api/v1
│   │   │   ├── api/                ← api.Error envelope
│   │   │   └── middleware/         ← Logging, auth, tracing
│   │   ├── persistence/
│   │   │   ├── memory/             ← In-memory repos (always)
│   │   │   └── gorm/               ← GORM repos (row DTO pattern)
│   │   ├── uow/                    ← In-memory UoW with TxRepositories
│   │   ├── outbox/                 ← Transactional UoW + worker (outbox feature)
│   │   └── auth/jwt/               ← JWT token service
│   ├── application/
│   │   ├── user/                   ← CQRS command/query handlers
│   │   └── uow/                    ← UnitOfWork + TxRepositories port
│   ├── config/                     ← Viper + env config loading
│   ├── domain/
│   │   ├── shared/                 ← DomainEvent interface + encode/decode
│   │   └── user/                   ← Aggregate, events, errors, repository port
│   ├── ports/                      ← Cross-cutting interfaces
│   └── validator/                  ← Request validation
├── pkg/
│   ├── logging/                    ← slog helpers + redaction + context enrichment
│   └── crypto/                     ← bcrypt hasher + AES-256-GCM cipher
├── migrations/                     ← SQL migration pairs
├── docs/                           ← Generated Swagger spec
├── .agents/skills/crank-project/   ← Agent skill file
├── .crank.yaml                     ← Project manifest (do not delete)
├── Dockerfile
└── Makefile
```

---

## Composition root

`cmd/server/main.go` is the single entry point and the only place where concrete types are wired together.

### What gets wired (feature-dependent)

| Client | Initialized when |
|--------|-----------------|
| `eventbus.NewInMemory()` | Always |
| `memory.NewUserRepository()` | No ORM feature |
| `gorm.NewUserRepository(gormDB)` | `gorm` feature |
| `uow.NewInMemoryUoW(bus, userRepo)` | No outbox |
| `outboxadapter.NewGormUoW(gormDB)` | `outbox` + ORM |
| `jwt.NewTokenService(cfg.JWT, denylist)` | `auth` feature |
| `auditapp.NewQueryHandler(auditStore)` | `audit` feature (spliced by `crank make scaffold`) |
| `redisclient.NewClient(cfg.Redis)` | `redis` feature |
| `qdrantclient.NewClient(ctx, cfg.Qdrant)` | `qdrant` feature |
| `temporal.NewClient(cfg.Temporal, logger)` | `temporal` feature |

Optional services (redis, qdrant, temporal) gracefully degrade — the server starts without them if unavailable.

---

## Domain layer

The domain layer (`internal/domain/`) is pure Go with zero external dependencies.

### Package anatomy

| File | Purpose |
|------|---------|
| `user.go` | Aggregate root struct + constructor + `Rehydrate()` + behavior methods |
| `events.go` | `UserCreated`, `UserUpdated`, `UserDeleted` — all carry `ID uuid.UUID` |
| `errors.go` | Sentinel errors: `ErrUserNotFound`, `ErrInvalidUser`, `ErrInvalidUserID` |
| `repository.go` | Port (interface) — implementations live in adapters |

### Aggregate design

```go
type User struct {
    id        uuid.UUID      // uuid.UUID from google/uuid
    name      string         // private — accessed only through getters
    email     string
    password  string         // bcrypt hash (with auth feature)
    createdAt time.Time
    updatedAt time.Time
    events    []shared.DomainEvent  // gorm:"-" — ignored by GORM
}
```

### Key patterns

- **uuid.UUID IDs** — All aggregates use `uuid.UUID`. Commands carry `string` IDs, parsed by handlers. Generate new UUID on create when empty.
- **Private fields with getters** — `ID()`, `Name()`, `Email()`. No direct field access.
- **No struct tags on aggregates** — No `json`, `db`, or `gorm` tags. Each layer owns its DTOs.
- **Rehydrate() bypasses validation** — Used by persistence adapters for lossless round-trips.
- **Constructor validates** — `NewUser(id, name, email)` rejects `uuid.Nil` and empty strings.

---

## Application layer

Implements use cases with **CQRS** — Commands mutate, Queries read.

### Command handler — TxRepositories pattern

```go
type CommandHandler struct {
    repo user.Repository
    uow  uow.UnitOfWork
}

func (h *CommandHandler) HandleCreate(ctx context.Context, cmd CreateUserCommand) (*user.User, error) {
    if cmd.ID == "" { cmd.ID = uuid.New().String() }
    id, _ := uuid.Parse(cmd.ID)
    u, _ := user.NewUser(id, cmd.Name, cmd.Email)

    if err := h.uow.SaveAndPublish(ctx, func(ctx context.Context, repos uow.TxRepositories) error {
        return repos.Users().Save(ctx, u)   // tx-scoped repo, zero GORM imports
    }, u.PullEvents()); err != nil {
```

The save closure receives a `TxRepositories` that hands out transaction-scoped repos. Command handlers **never** import persistence adapters or `*gorm.DB`.

### Unit of Work + TxRepositories

Defined in `internal/application/uow/uow.go`:

```go
type TxRepositories interface {
    Users() user.Repository
    // crank:tx-repositories ← new repos spliced here by crank make scaffold
}

type UnitOfWork interface {
    SaveAndPublish(ctx context.Context, save func(ctx context.Context, repos TxRepositories) error, events []shared.DomainEvent) error
}
```

Two implementations:
- **In-memory** — `internal/adapters/uow/in_memory_uow.go` — non-transactional, save then publish
- **Outbox** — `internal/adapters/outbox/gorm_uow.go` — GORM transaction wraps save + outbox append

---

## HTTP adapter

### v1 handlers

Handlers live in `package v1` at `internal/adapters/http/web/v1/`. Routes mount at `e.Group("/api/v1")`.

### api.Error envelope

```go
// internal/adapters/http/web/api/error.go
package api

type Error struct {
    Error   string            `json:"error"`
    Details map[string]string `json:"details,omitempty"`
}
```

### Exported EchoBinder

```go
// internal/adapters/http/web/server.go
type EchoBinder struct {
    DefaultBinder *echo.DefaultBinder
    Logger        *slog.Logger
}
```

The binder auto-validates after `c.Bind()` — handlers never call the validator manually.

### HTTP error handler

The error handler is in `web.NewServer()`, not `main.go`. It maps:
- `*validator.ValidationError` → 422 with field details
- `*echo.HTTPError` → its status code
- Unhandled → 500

### Self-scoped user endpoints

User handlers verify `user_id == c.Param("id")` on Get/Update/Delete, returning 404 on mismatch (IDOR protection). Users can only access their own records.

### CORS

CORS middleware is configured in `main.go` with configurable origins from `app.cors_origins`.

---

## Persistence adapters

### Row DTO pattern

Persistence adapters use a **private row DTO** with `toAggregate()` / `rowFromAggregate()`:

```go
// internal/adapters/persistence/gorm/user_repository.go
type userRow struct {
    ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
    Name      string    `gorm:"column:name;not null"`
    Email     string    `gorm:"column:email;not null;uniqueIndex"`
    Password  string    `gorm:"column:password;not null"`
    CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
    UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (row *userRow) toAggregate() *user.User {
    return user.Rehydrate(row.ID, row.Name, row.Email, row.Password, row.CreatedAt, row.UpdatedAt)
}

func rowFromAggregate(u *user.User) *userRow {
    return &userRow{ID: u.ID(), Name: u.Name(), Email: u.Email(), Password: u.PasswordHash(), ...}
}
```

Domain aggregates have zero ORM tags. The row DTO owns the column mapping.

### In-memory repository

Thread-safe `sync.RWMutex`-guarded `map[uuid.UUID]*user.User` — no row DTO needed since there's no ORM layer.

Both in-memory and ORM adapters are always generated. The composition root selects which to use.

---

## Auth

- JWT service at `internal/adapters/auth/jwt/` with `Issue`, `Refresh`, `Subject`, `Revoke`
- Token denylist at `internal/ports/tokendenylist.go` with GORM adapter
- bcrypt hasher at `pkg/crypto/bcrypt_hasher.go`
- Middleware at `internal/adapters/http/web/middleware/auth.go` stores `user_id` in Echo context
- `/auth/logout` revokes refresh tokens

---

## Logging

`pkg/logging/` provides:
- **Three-layer handler stack**: JSONHandler → redactionHandler → ContextHandler
- **Redaction** scrubs passwords, secrets, tokens, API keys from both keys and inline values
- **Context enrichment** auto-injects `request_id` and `user_id` from Go context
- **Source compression** shortens file paths to `package/file.go:line`

---

## Configuration

```
Priority:  env vars  >  .env file  >  configs/config.yaml
```

Secret fields tagged `env:"..."` are overlaid from `.env` / environment. Non-secrets stay in YAML.

Feature config injection uses marker comments (`// crank:config-fields`, `# crank:config-section`). Adding the same feature twice is idempotent.

---

## Testing

In-memory adapters enable testing every layer without a database.

```go
// Domain: pure unit tests — no mocks needed
func TestNewUser(t *testing.T) {
    u, _ := user.NewUser(uuid.New(), "Alice", "alice@example.com")
    assert.Equal(t, "Alice", u.Name())
}

// Application: in-memory repo + bus
func TestCreateUser(t *testing.T) {
    repo := memory.NewUserRepository()
    bus := eventbus.NewInMemory()
    uow := uow.NewInMemoryUoW(bus, repo)
    handler := userapp.NewCommandHandler(repo, uow)
    // ...
}
```

---

## Code generation

`crank make scaffold` generates the full stack and auto-wires:
- **HTTP handlers** into `internal/adapters/http/web/v1/routes.go`
- **TxRepositories accessors** into `internal/application/uow/uow.go` and all UoW implementations

Generated handlers use `package v1`, `api.Error`, and the `Register(g *echo.Group)` pattern.

---

## External links

| Framework | Docs |
|-----------|------|
| [Echo v5](https://echo.labstack.com) | HTTP framework |
| [Viper](https://github.com/spf13/viper) | Config management |
| [GORM](https://gorm.io/docs) | GORM ORM |
| [golang-jwt](https://github.com/golang-jwt/jwt) | JWT library |
| [go-redis](https://redis.uptrace.dev) | Redis client |
| [swaggo/swag](https://github.com/swaggo/swag) | Swagger/OpenAPI |
| [golang-migrate](https://github.com/golang-migrate/migrate) | SQL migrations |
| [Qdrant](https://qdrant.tech/documentation/) | Vector database |
| [Temporal](https://docs.temporal.io/dev-guide/go) | Workflow orchestration |
