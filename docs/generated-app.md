---
title: Navigating the Generated Application
---

# Navigating the generated application

After you run `crank init`, you have a fully wired Go backend service. This page explains every layer of the generated code — where things live, why they are there, how they connect, and how to extend them.

---

## Architecture overview

The generated project follows a **Domain-Driven Design (DDD)** layered architecture. Every layer depends only on the layer below it — never upward.

```text
┌────────────────────────────────────────────────────┐
│  🌐 Presentation Layer (HTTP Adapter)              │
│  internal/adapters/http/web/                       │
│  Echo handlers, DTOs, middleware, route mounting   │
└──────────────────────┬─────────────────────────────┘
                       │  depends on
┌──────────────────────▼─────────────────────────────┐
│  ⚙️ Application Layer (CQRS)                       │
│  internal/application/                              │
│  Command/Query handlers — use case orchestration    │
└──────────────────────┬─────────────────────────────┘
                       │  depends on
┌──────────────────────▼─────────────────────────────┐
│  🧠 Domain Layer                                   │
│  internal/domain/                                   │
│  Pure Go aggregates, value objects, domain events,  │
│  repository interfaces (ports)                     │
└──────────────────────┬─────────────────────────────┘
                       │  implemented by
┌──────────────────────▼─────────────────────────────┐
│  🔌 Infrastructure Layer (Adapters)                │
│  internal/adapters/persistence/                     │
│  internal/adapters/eventbus/                        │
│  internal/adapters/uow/                             │
│  Concrete implementations of domain ports          │
└────────────────────────────────────────────────────┘
```

**Key architectural rules:**

| Rule | Explanation |
|------|-------------|
| **Domain is pure Go** | The domain layer has zero framework imports — no HTTP, no database drivers, no serialization tags. |
| **Dependencies point inward** | Infrastructure adapters implement interfaces defined by the domain. The domain never imports adapters. |
| **Application coordinates** | Application handlers load aggregates from repositories, call domain methods, and save results — they orchestrate, not implement. |
| **Composition root** | `cmd/server/main.go` is the only place where concrete types (repository implementations, bus, etc.) are wired together. |

---

## Directory layout

```
myapp/
├── cmd/
│   └── server/
│       └── main.go                 Entry point + composition root
├── configs/
│   └── config.yaml                 Non-secret config defaults
├── internal/
│   ├── adapters/                   Infrastructure implementations
│   │   ├── eventbus/
│   │   ├── http/web/
│   │   ├── persistence/
│   │   ├── cache/                  (with redis feature)
│   │   └── uow/                    Unit of Work implementations
│   ├── application/                CQRS use cases
│   │   └── user/
│   ├── config/                     Viper + env config loading
│   ├── domain/                     Pure domain model
│   │   ├── shared/                 DomainEvent interface, encode/decode
│   │   └── user/                   User aggregate, ID, events, errors, repository port
│   ├── model/                      Shared DTOs (APIError)
│   ├── ports/                      Cross-cutting interfaces (EventBus, UnitOfWork)
│   └── validator/                  Request validation setup
├── pkg/
│   └── logging/                    slog helpers + redaction
├── docs/                           Generated Swagger spec
├── migrations/                     SQL migrations (with bun/gorm feature)
├── .crank.yaml                     Project manifest
├── .env.example                    Local env template
├── .air.toml                       Live-reload config
├── Dockerfile
└── Makefile                        Project-specific targets
```

---

## The composition root (`cmd/server/main.go`)

This is the application entry point and the **only place** where concrete types are wired together. It performs these steps in order:

1. **Load config** — calls `config.Load()` which reads `configs/config.yaml`, `.env`, and environment variables.
2. **Initialize logger** — creates a `slog.Logger` from the logging config.
3. **Connect infrastructure** — opens database connections (PostgreSQL via Bun/GORM), Redis, MongoDB, Qdrant, or Temporal client depending on enabled features.
4. **Wire the domain** — creates the in-memory event bus, selects the repository implementation (in-memory, Bun, or GORM), and constructs the Unit of Work.
5. **Create application services** — passes repository + UoW to command/query handler constructors.
6. **Create HTTP handlers** — passes application services to HTTP handler constructors.
7. **Mount routes** — calls `web.Mount(e, config)` to register all route groups on the Echo instance.
8. **Start the server** — begins listening and handles graceful shutdown on SIGINT/SIGTERM.

This is also where you wire new feature clients. The generated code uses markers (`// crank:config-*`, `// crank:http-*`) so that `crank add` and `crank make handler` can inject new wiring without breaking existing code.

---

## Domain layer (`internal/domain/`)

The domain layer is **pure Go** — no HTTP, database, or serialization concerns. Each business concept lives in its own package.

### Package structure

Taking the generated `internal/domain/user/` package as an example:

| File | Purpose |
|------|---------|
| `user.go` | Aggregate root struct + constructor (`NewUser`) + behavior methods (`Update`, `MarkDeleted`) + event recording (`PullEvents`) |
| `user_id.go` | Typed value object wrapping a string — makes it impossible to accidentally pass a different resource's ID at compile time |
| `events.go` | Domain events: `UserCreated`, `UserUpdated`, `UserDeleted` |
| `errors.go` | Sentinel errors: `ErrUserNotFound`, `ErrInvalidUser` |
| `repository.go` | **Port** (interface) defining what the persistence layer must provide — implementations live in adapters |

### The aggregate

```go
type User struct {
    id        UserID
    name      string
    email     string
    password  string       // bcrypt hash (only with auth feature)
    createdAt time.Time
    updatedAt time.Time
    events    []shared.DomainEvent
}
```

Key design choices:

- **Unexported fields** — the aggregate controls its own state through methods. No one can set `user.name` directly.
- **No struct tags** — no `json`, `db`, `validate`, `gorm`, or `bun` tags on the domain type. Each layer defines its own DTOs.
- **Methods return errors** — constructors and mutators validate invariants (e.g., empty name → `ErrInvalidUser`).
- **Event sourcing ready** — behavior methods record events via `recordEvent`. Call `PullEvents()` to collect and dispatch them.

### Creating a new aggregate

```go
id, _ := user.NewUserID("usr_123")
u, err := user.NewUser(id, "Alice", "alice@example.com")
// u.PullEvents() → [UserCreated{...}]
```

### Shared domain event infrastructure

The `internal/domain/shared/` package defines:

- `DomainEvent` interface — `EventName() string`, `OccurredAt() time.Time`
- `EncodeEvent()` / `DecodeEvent()` — JSON serialize/deserialize events for transport (outbox, message queue)
- `EventEnvelope` — wire format with name, timestamp, and JSON body
- Event factory registry — maps event names to constructor functions for reliable deserialization

---

## Application layer (`internal/application/`)

The application layer implements use cases using the **CQRS pattern** (Command Query Responsibility Segregation). Commands mutate state, queries read state — they never mix.

### Package structure

Taking the generated `internal/application/user/` package:

| File | Purpose |
|------|---------|
| `commands.go` | Command structs: `CreateUserCommand`, `UpdateUserCommand`, `DeleteUserCommand` |
| `command_handler.go` | Handles each command — loads aggregate → calls domain method → saves + publishes through UnitOfWork |
| `queries.go` | Query structs: `GetUserQuery`, `ListUsersQuery` |
| `query_handler.go` | Handles each query — fetches data through repository port (read-only) |

### Command handler pattern

```go
func (h *CommandHandler) HandleCreate(ctx context.Context, cmd CreateUserCommand) (*user.User, error) {
    id, err := user.NewUserID(cmd.ID)
    u, err := user.NewUser(id, cmd.Name, cmd.Email)
    err := h.uow.SaveAndPublish(ctx,
        func(ctx context.Context) error { return h.repo.Save(ctx, u) },
        u.PullEvents(),
    )
    return u, nil
}
```

The command handler **never** calls `repo.Save` directly — it goes through the **Unit of Work** (see below). This ensures the aggregate save and its domain event publication are handled atomically.

### Why CQRS?

Even though commands and queries currently share the same repository, separating them at the method level means:

- **Queries can be optimized independently** — later you can swap in a read-optimized store (cached view, materialized table) without touching command logic.
- **Handlers stay focused** — command handlers handle mutations and events; query handlers do pure reads.
- **Clear intent at the HTTP layer** — a `POST` handler calls `cmd.HandleCreate`, a `GET` handler calls `qry.HandleGet`.

---

## HTTP adapter (`internal/adapters/http/web/`)

This is the **presentation layer** — it translates HTTP requests to application commands/queries and translates results back to JSON.

### Files at a glance

| File | Role |
|------|------|
| `server.go` | Creates Echo instance, wires the smart binder, exposes `GET /health` |
| `routes.go` | Central route aggregator — mounts each handler's route group. New handlers are spliced in via `// crank:http-*` markers |
| `user_handler.go` | CRUD handler for users — DTOs, Swagger annotations, error mapping |
| `middleware/logging.go` | Injects a request-scoped logger into the context |
| `middleware/auth.go` | JWT bearer token validation (with auth feature) |

### The smart binder

The generated `server.go` replaces Echo's default binder with a custom `echoBinder` that **automatically runs validation** after every `c.Bind()`:

```go
type echoBinder struct {
    defaultBinder echo.Binder
    logger        *slog.Logger
}

func (b *echoBinder) Bind(i any, c echo.Context) error {
    if err := b.defaultBinder.Bind(i, c); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    if err := validator.Struct(i); err != nil {
        return err  // automatically formatted as ValidationError
    }
    return nil
}
```

This means handlers never call the validator explicitly:

```go
func (h *UserHandler) Create(c echo.Context) error {
    var in userDTO
    if err := c.Bind(&in); err != nil {  // Bind + validate in one call
        return err
    }
    // `in` is guaranteed valid past this point
    out, err := h.cmd.HandleCreate(c.Request().Context(), in.toCreateCommand())
    // ...
}
```

### DTO pattern

HTTP handlers define their own request/response DTOs. These structs carry JSON and validation tags — they are **not** the domain aggregate:

```go
type userDTO struct {
    ID    string `json:"id"    validate:"required"`
    Name  string `json:"name"  validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}

func (d userDTO) toCreateCommand() appuser.CreateUserCommand {
    return appuser.CreateUserCommand{ID: d.ID, Name: d.Name, Email: d.Email}
}
```

### Route wiring

The `routes.go` file uses marker comments to keep `crank make handler` and `crank make scaffold` idempotent:

```go
type MountConfig struct {
    UserHandler *UserHandler
    // crank:http-fields (do not remove — crank make handler splices new fields here)
}

func Mount(e *echo.Echo, cfg MountConfig) {
    g := e.Group("/users")
    cfg.UserHandler.Register(g)
    // crank:http-register (do not remove — crank make handler splices new registrations here)
}
```

### Error handling

The composition root installs a custom `HTTPErrorHandler` that:

1. Catches `validator.ValidationError` → returns 422 with field-level error details
2. Catches `echo.HTTPError` → returns the appropriate HTTP status code
3. Falls through to 500 Internal Server Error for unhandled errors

Every handler also maps domain errors to HTTP status codes:

```go
if errors.Is(err, user.ErrUserNotFound) {
    return c.JSON(http.StatusNotFound, model.APIError{Error: "user not found"})
}
```

---

## Persistence adapters (`internal/adapters/persistence/`)

These implement the domain's `Repository` interface. The composition root decides which one to wire in.

### Available adapters

| Adapter | Location | Use Case |
|---------|----------|----------|
| **In-memory** | `persistence/memory/` | Unit tests, local dev without database |
| **GORM** (PostgreSQL) | `persistence/gorm/` | Production with GORM (with gorm feature) |
| **Bun** (PostgreSQL) | `persistence/bun/` | Production with Bun (with bun feature) |

The **in-memory repository** is a thread-safe `sync.RWMutex`-guarded map:

```go
type UserRepository struct {
    mu    sync.RWMutex
    byID  map[string]*user.User
    byEml map[string]*user.User
}
```

It is the default when no ORM feature is enabled and is always available for tests regardless of which ORM is active.

### Switching adapters

To switch from in-memory to PostgreSQL, change only the composition root (`cmd/server/main.go`):

```go
// In-memory (default, no ORM feature)
userRepo := memory.NewUserRepository()

// Bun (with bun feature)
userRepo := bun.NewUserRepository(db)

// GORM (with gorm feature)
userRepo := gorm.NewUserRepository(gormDB)
```

No application or HTTP code changes — they depend only on the domain's `Repository` interface.

---

## Unit of Work & Event bus

These two cross-cutting abstractions ensure atomicity between persisting an aggregate and publishing its domain events.

### The core flow

```text
Application Handler
      │
      ▼
UnitOfWork.SaveAndPublish(ctx, saveFn, events)
      │
      ├── 1. saveFn(ctx) → repo.Save(user)
      └── 2. bus.Publish(ctx, events...)
```

### In-memory Unit of Work

The default implementation (`internal/adapters/uow/`) runs save and publish sequentially:

```go
func (u *InMemoryUoW) SaveAndPublish(ctx context.Context, save func(ctx context.Context) error, events []shared.DomainEvent) error {
    if err := save(ctx); err != nil {
        return err
    }
    if u.bus != nil && len(events) > 0 {
        _ = u.bus.Publish(ctx, events...)
    }
    return nil
}
```

- If save fails, events are **not** published (short-circuit).
- If publish fails, the save stands (best-effort — log and continue).

### Outbox Unit of Work

With the `outbox` feature enabled, both save and publish share a **database transaction**:

1. The aggregate is saved to its table.
2. Domain events are written to an `outbox_events` table in the same transaction.
3. A background worker polls the `outbox_events` table and relays events to the bus.
4. Events are deleted from the table once published (at-least-once delivery).

This eliminates the window between "save succeeded" and "publish failed" that the in-memory UoW has.

### The Event Bus

The generated in-memory bus (`internal/adapters/eventbus/`) is a simple pub/sub:

```go
bus := eventbus.NewInMemory()
bus.Subscribe(func(ctx context.Context, ev shared.DomainEvent) error {
    slog.Default().Info("event received", "event", ev.EventName())
    return nil
})
```

Subscribers are called synchronously in registration order. A failing subscriber does not block other subscribers — the error is logged and the next subscriber runs.

---

## Cross-cutting packages

### Configuration (`internal/config/`)

Uses **Viper** for YAML config and **caarlos0/env** for secret overlay:

```text
Priority:  env vars  >  .env file  >  configs/config.yaml
```

- Non-secret fields (host, port, log level) go in `configs/config.yaml`.
- Secret fields are tagged with `env:"SECRET_NAME"` — they are loaded from `.env` or environment variables.
- Feature configs are injected via marker comments (`// crank:config-fields`, `// crank:config-defaults`).

### Validation (`internal/validator/`)

Preconfigured `go-playground/validator` singleton:

- Uses JSON tag names in error messages (API consumers see `"name"` not `"Name"`).
- Automatically called by the smart Echo binder after every `c.Bind()`.
- Extensible — add custom validators before the closing comment in `init()`:

```go
validate.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
    return strings.TrimSpace(fl.Field().String()) != ""
})
```

### Model (`internal/model/`)

Shared response types like `APIError` that every HTTP handler uses:

```go
type APIError struct {
    Error   string   `json:"error"`
    Details any      `json:"details,omitempty"`
}
```

### Logging (`pkg/logging/`)

Helpers built on `log/slog`:

- `logging.New(level, addSource)` — creates a configured logger.
- `logging.FromContext(ctx)` — retrieves the request-scoped logger injected by middleware.
- `logging.Redacted(key, value)` — redacts sensitive values (passwords, tokens) from log output.

### Ports (`internal/ports/`)

Cross-cutting interfaces that adapters implement:

| Interface | Purpose | Default implementation |
|-----------|---------|-----------------------|
| `EventBus` | Publish domain events | In-memory bus (or outbox with outbox feature) |
| `UnitOfWork` | Atomic save + publish | In-memory UoW (or outbox UoW with outbox feature) |
| `Hasher` | Hash and compare passwords | BCrypt hasher (with auth feature) |
| `TokenService` | Issue, refresh, validate JWT tokens | JWT service (with auth feature) |
| `Cache` | Get/Set/Delete cache entries | (with redis feature) |
| `Cipher` | Encrypt/Decrypt data | AES-256-GCM (with crypto feature) |
| `TracerProvider` | Create OpenTelemetry tracers | OTel stdout exporter (with otel feature) |

---

## Request lifecycle

This is the complete path of a `POST /users` request through the system:

```text
Client
  │
  │  POST /users  { "name": "Alice", "email": "a@b.com" }
  ▼
Echo Router
  │
  ▼
echoBinder.Bind(&userDTO)
  │
  ├── DefaultBinder.Bind() → parse JSON into userDTO
  └── validator.Struct()   → validate tags (required, email, etc.)
  │
  ▼ (if validation fails → 422 ValidationError)
UserHandler.Create(c)
  │
  ├── dto.toCreateCommand() → user.CreateUserCommand{...}
  └── cmd.HandleCreate(ctx, cmd)
        │
        ├── user.NewUser(id, name, email) → aggregate + UserCreated event
        └── uow.SaveAndPublish(ctx, saveFn, events)
              │
              ├── repo.Save(ctx, user)
              └── bus.Publish(ctx, UserCreated{...})
  │
  ▼
toUserDTO(result) → 201 Created + JSON
```

---

## Feature modules

Each optional feature adds files to the project. Here is what every feature contributes and where.

### Auth (`auth`)

| File | Purpose |
|------|---------|
| `internal/ports/hasher.go` | Password hashing interface |
| `internal/ports/tokenservice.go` | Token issue/refresh/validation interface |
| `internal/adapters/crypto/bcrypt_hasher.go` | BCrypt hashing implementation |
| `internal/adapters/crypto/jwt_token_service.go` | JWT access + refresh token management |
| `internal/adapters/http/web/auth_handler.go` | `/auth/register`, `/auth/login`, `/auth/refresh`, `/me` endpoints |
| `internal/adapters/http/web/middleware/auth.go` | `JWTAuth()` middleware for protecting routes |
| `internal/domain/user/email.go` | Email value object with validation |
| `internal/domain/user/password.go` | Password value object with hashing |

**Auth endpoints:**

| Method | Path | Auth required |
|--------|------|---------------|
| POST | `/auth/register` | No — creates account + returns tokens |
| POST | `/auth/login` | No — returns tokens on valid credentials |
| POST | `/auth/refresh` | No — exchange refresh token for new pair |
| GET | `/me` | Yes — returns current user ID from JWT |

**Protecting custom routes:**

```go
e.GET("/admin", adminHandler, middleware.JWTAuth(tokens))
```

### Bun ORM (`bun`)

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/bun/db.go` | PostgreSQL connection via `pgdriver` with connection pooling |
| `internal/adapters/persistence/bun/migrate.go` | golang-migrate wrapper for Bun |
| `internal/adapters/persistence/bun/user_repository.go` | Bun-backed user repository |
| `migrations/000001_init.up.sql` | Initial schema |
| `migrations/000001_init.down.sql` | Schema rollback |

See [bun.uptrace.dev](https://bun.uptrace.dev) for Bun documentation.

### GORM (`gorm`)

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/gorm/db.go` | PostgreSQL connection via GORM with connection pooling |
| `internal/adapters/persistence/gorm/migrate.go` | golang-migrate wrapper for GORM |
| `internal/adapters/persistence/gorm/user_repository.go` | GORM-backed user repository |
| `migrations/000001_init.up.sql` | Initial schema |
| `migrations/000001_init.down.sql` | Schema rollback |

See [gorm.io/docs](https://gorm.io/docs) for GORM documentation.

### Redis (`redis`)

| File | Purpose |
|------|---------|
| `internal/ports/cache.go` | Cache interface |
| `internal/adapters/cache/redis/client.go` | `redis.Client` connection with ping validation |

Wired in the composition root as `rdb` — use it directly or wrap with your own cache abstraction. See [redis.uptrace.dev](https://redis.uptrace.dev) for go-redis docs.

### MongoDB (`mongodb`)

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/mongodb/client.go` | `mongo.Client` connection |

Wired as `mdb` — access collections via `mdb.Client().Database("name").Collection("coll")`. See [mongodb.com/docs](https://www.mongodb.com/docs/drivers/go/) for the Go driver docs.

### Qdrant (`qdrant`)

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/qdrant/client.go` | Qdrant gRPC client connection |

See [qdrant.tech/documentation](https://qdrant.tech/documentation/) for API docs.

### Temporal (`temporal`)

| File | Purpose |
|------|---------|
| `internal/adapters/temporal/client.go` | Temporal client + slog bridge |
| `internal/adapters/temporal/worker.go` | Worker with workflow + activity registration (marker-based) |
| `internal/adapters/temporal/workflow/greeting.go` | Example workflow |
| `internal/adapters/temporal/activity/greeting.go` | Example activity |
| `cmd/worker/main.go` | Standalone worker entry point |

Generate new workflows and activities:

```bash
crank make workflow OrderFulfillment order_id:uuid
crank make activity ChargeCard amount:float --tests
```

They are auto-wired into the worker via `// crank:workflow-register` / `// crank:activity-register` markers.

See [docs.temporal.io](https://docs.temporal.io/dev-guide/go) for the Temporal Go SDK docs.

### OpenTelemetry (`otel`)

| File | Purpose |
|------|---------|
| `internal/ports/tracer.go` | TracerProvider interface |
| `internal/adapters/telemetry/otel.go` | OTel SDK setup (stdout exporter by default) |
| `internal/adapters/http/web/middleware/tracing.go` | HTTP tracing middleware |

Currently emits spans to stdout. Replace `stdouttrace` exporter with an OTLP exporter for production (Jaeger, Datadog, Grafana Tempo, etc.).

See [opentelemetry.io/docs/languages/go](https://opentelemetry.io/docs/languages/go/) for the Go OTel SDK docs.

### Crypto (`crypto`)

| File | Purpose |
|------|---------|
| `internal/ports/cipher.go` | Encrypt/Decrypt interface |
| `internal/adapters/crypto/aesgcm/cipher.go` | AES-256-GCM implementation |

Generate a strong key with `openssl rand -base64 32` and set it in `crypto.secret` in config.

### Views (`views`)

| File | Purpose |
|------|---------|
| `internal/adapter/http/web/views.go` | SPA serving + Vite proxy for HMR |
| `static/embed.go` | Embedded static assets |
| `static/dist/index.html` | Built SPA entry point |
| `src/App.jsx` | React application entry |
| `vite.config.js` | Vite dev server config |

Toggle with `views.enabled` in config. Set `views.dev_server` to your Vite URL for HMR during frontend development.

### Outbox (`outbox`)

| File | Purpose |
|------|---------|
| `internal/domain/outbox/event.go` | Outbox event domain type |
| `internal/domain/outbox/repository.go` | Outbox repository port |
| `internal/adapter/outbox/bun_repository.go` | Bun-backed outbox implementation |
| `internal/adapter/outbox/gorm_repository.go` | GORM-backed outbox implementation |
| `internal/adapter/outbox/bun_uow.go` | Bun-backed transactional Unit of Work |
| `internal/adapter/outbox/gorm_uow.go` | GORM-backed transactional Unit of Work |
| `internal/adapter/outbox/worker.go` | Background worker that polls + publishes |
| `migrations/000002_add_outbox_events.up.sql` | Outbox table schema |
| `migrations/000002_add_outbox_events.down.sql` | Outbox table rollback |

Requires `gorm` or `bun`. Replaces the default in-memory Unit of Work with a transaction-backed implementation.

---

## Code generation output (`crank make`)

### What `crank make scaffold` produces

```bash
crank make scaffold Product title:string price:float --tests
```

| Layer | Files generated |
|-------|----------------|
| **Domain** | `internal/domain/product/product.go` (aggregate), `product_id.go`, `events.go`, `errors.go`, `repository.go` |
| **Application** | `internal/application/product/commands.go`, `command_handler.go`, `queries.go`, `query_handler.go` |
| **Persistence** | `internal/adapters/persistence/memory/product_repository.go` (always), plus `bun/` or `gorm/` variant if enabled |
| **HTTP** | `internal/adapters/http/web/product_handler.go` + route wiring in `routes.go` |
| **Tests** | `_test.go` for each layer (with `--tests` flag) |
| **Migration** | `migrations/<timestamp>_create_products.{up,down}.sql` (with bun feature) |

**Idempotency rules:**

- Dependency files (model, service, repository) are **skipped** if they already exist.
- The primary artifact (the handler for `crank make handler`, the domain model for `crank make model`) **errors** if it exists unless `--force` is passed.
- Route wiring is **idempotent** — running `crank make scaffold Product` twice will not duplicate the route registration.

### What each kind generates

| Kind | Primary artifact | Dependencies generated |
|------|-----------------|----------------------|
| `model` | Domain aggregate, ID, events, errors, repository port | — |
| `repository` | Repository implementation | Domain model (if missing) |
| `service` | Commands, queries, handlers | Domain model, repository (if missing) |
| `handler` | HTTP handler + route wiring | Domain model, service, repository (skipped with `--only`) |
| `scaffold` | Everything above | Everything above |
| `workflow` | Temporal workflow + worker registration | — |
| `activity` | Temporal activity + worker registration | — |
| `migration` | SQL up/down pair | — |

---

## Testing patterns

The in-memory adapters make it possible to test every layer without a database.

### Testing the domain

Domain tests need no mocks — they test pure Go types:

```go
func TestNewUser(t *testing.T) {
    id, _ := user.NewUserID("test-1")
    u, err := user.NewUser(id, "Alice", "alice@example.com")
    assert.NoError(t, err)
    assert.Equal(t, "Alice", u.Name())
    events := u.PullEvents()
    assert.Len(t, events, 1)
    assert.Equal(t, "user.created", events[0].EventName())
}
```

### Testing application handlers

Wire the in-memory repository, event bus, and Unit of Work directly:

```go
func TestCreateUser_CommandHandler(t *testing.T) {
    repo := memory.NewUserRepository()
    bus := eventbus.NewInMemory()
    uow := uow.NewInMemoryUoW(bus)
    handler := userapp.NewCommandHandler(repo, uow)

    user, err := handler.HandleCreate(context.Background(), user.CreateUserCommand{
        ID: "test-1", Name: "Alice", Email: "alice@example.com",
    })
    assert.NoError(t, err)
    assert.Equal(t, "Alice", user.Name())
}
```

### Testing HTTP handlers

Use Echo's test helpers (`httptest`):

```go
func TestUserHandler_Create(t *testing.T) {
    // Wire in-memory adapters
    repo := memory.NewUserRepository()
    bus := eventbus.NewInMemory()
    uow := uow.NewInMemoryUoW(bus)
    cmd := userapp.NewCommandHandler(repo, uow)
    qry := userapp.NewQueryHandler(repo)
    handler := web.NewUserHandler(cmd, qry)

    // Echo test
    e := echo.New()
    body := `{"name":"Alice","email":"a@b.com"}`
    req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    err := handler.Create(c)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

---

## Configuration deep dive

### How config loads

```text
godotenv.Load()              ← reads .env into os.Environ
      │
viper.ReadInConfig()         ← reads configs/config.yaml
      │
env.ParseWithOptions(&cfg)   ← overlays env vars on tagged fields
      ▼
Final *Config
```

- **Viper** handles all non-secret fields — app name, host, port, database host/port, log level, etc.
- **caarlos0/env** overlays only fields tagged with `env:"..."` — typically secrets like `JWT_SECRET`, `DATABASE_URL`.
- **`.env` is optional** — if it doesn't exist, `godotenv.Load()` silently skips it.

### Adding a new config field

1. Add the field to the `Config` struct in `internal/config/config.go`.
2. Add a Viper default in the `setDefaults()` function.
3. Add the YAML key to `configs/config.yaml`.
4. If it's a secret, add an `env:"VAR_NAME"` tag and document it in `.env.example`.

### Feature config injection

When you run `crank add redis`, new config fields, structs, and defaults are injected into `internal/config/config.go` at marker comments:

```go
type Config struct {
    App AppConfig `mapstructure:"app"`
    // crank:config-fields       ← RedisConfig field is inserted here
    Logging LoggingConfig `mapstructure:"logging"`
}
```

The injection is **idempotent** — adding the same feature twice is a no-op.

---

## External framework documentation

| Framework | Official docs |
|-----------|---------------|
| Echo v4 (HTTP) | [echo.labstack.com](https://echo.labstack.com) |
| Viper (config) | [github.com/spf13/viper](https://github.com/spf13/viper) |
| caarlos0/env (env config) | [github.com/caarlos0/env](https://github.com/caarlos0/env) |
| go-playground/validator | [pkg.go.dev](https://pkg.go.dev/github.com/go-playground/validator/v10) |
| log/slog | [pkg.go.dev/log/slog](https://pkg.go.dev/log/slog) |
| swaggo/swag (OpenAPI) | [github.com/swaggo/swag](https://github.com/swaggo/swag) |
| golang-migrate | [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate) |
| GORM | [gorm.io/docs](https://gorm.io/docs) |
| Bun ORM | [bun.uptrace.dev](https://bun.uptrace.dev) |
| go-redis | [redis.uptrace.dev](https://redis.uptrace.dev) |
| MongoDB Go driver | [mongodb.com/docs/drivers/go](https://www.mongodb.com/docs/drivers/go/) |
| Qdrant | [qdrant.tech/documentation](https://qdrant.tech/documentation/) |
| Temporal Go SDK | [docs.temporal.io/dev-guide/go](https://docs.temporal.io/dev-guide/go) |
| OpenTelemetry Go | [opentelemetry.io/docs/languages/go](https://opentelemetry.io/docs/languages/go/) |
| Domain-Driven Design | [dddcommunity.org](https://www.dddcommunity.org/) |
| CQRS pattern | [martinfowler.com/bliki/CQRS.html](https://martinfowler.com/bliki/CQRS.html) |
