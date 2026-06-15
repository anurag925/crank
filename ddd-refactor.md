# crank — DDD refactor of the generated project

## Overview

Refactor the **generated** project layout (produced by `crank init`, `crank add`, and `crank make`) to be Domain-Driven. Crank's own internal code (CLI, feature/tool registries, generators, the scaffold pipeline) is the mechanism — change only the *shape of the output*. The user-facing CLI surface stays identical: `crank init`, `crank add`, `crank make <kind>`, `crank migrate`, etc. all work as before.

This document is the single source of truth that every wave‑1/wave‑2 agent must follow. Wave‑3 (tests) and wave‑4 (validation) reference it too.

## Locked design decisions

1. **Commands + queries separated.** For each resource `<r>`, `application/<r>/` contains four files: `commands.go`, `command_handler.go`, `queries.go`, `query_handler.go`. Two handler structs (`CommandHandler`, `QueryHandler`) wired independently in the composition root.
2. **Aggregate is pure, no infra tags.** `domain/<r>/<r>.go` has zero `bun`, `json`, `db`, or `validate` tags. The postgres adapter owns a private `<r>Row` struct that maps to/from the aggregate. The HTTP adapter owns a separate `<r>DTO` struct for input/output (with `validate` tags).
3. **Events: aggregate field + `PullEvents()`.** Each behavior method on the aggregate appends a `DomainEvent` to an internal slice. The application service calls `aggregate.PullEvents()` after `Save`, then publishes them via the `EventBus` port.
4. **In-memory adapter ships by default.** `--features=base` (no postgres) projects get `internal/adapters/persistence/memory/<r>_repository.go`. The composition root in `cmd/server/main.go` wires the memory adapter. The project runs out of the box.
5. **No backward compat.** Hard breaking change. AGENTS.md/SPEC.md will note that existing generated projects must be re‑init'd.

## Target file tree (per generated project)

```
myapp/
├── cmd/server/main.go                       # composition root — explicit wiring
├── internal/
│   ├── domain/                              # pure Go, zero infra imports
│   │   ├── shared/events.go                 # DomainEvent interface
│   │   ├── <r>/
│   │   │   ├── <r>.go                       # aggregate root + behavior methods + PullEvents
│   │   │   ├── <r>_id.go                    # value object: <Resource>ID (typed string) — only if any field is uuid
│   │   │   ├── events.go                    # <Resource>Placed, <Resource>Cancelled, ...
│   │   │   ├── errors.go                    # Err<Resource>NotFound + other domain errors
│   │   │   └── repository.go                # Repository port (interface)
│   ├── application/                         # use cases
│   │   ├── <r>/
│   │   │   ├── commands.go                  # Create<Resource>Command, Update<Resource>Command, ...
│   │   │   ├── command_handler.go           # CommandHandler struct + Handle(ctx, cmd) methods
│   │   │   ├── queries.go                   # Get<Resource>Query, List<Resources>Query, ...
│   │   │   └── query_handler.go             # QueryHandler struct + Handle(ctx, q) methods
│   ├── ports/                               # cross-cutting ports (interfaces only)
│   │   ├── eventbus.go                      # EventBus port
│   │   ├── hasher.go                        # Hasher port (bcrypt) — auth
│   │   ├── tokenservice.go                  # TokenService port (JWT) — auth
│   │   ├── cipher.go                        # Cipher port (AES‑GCM) — crypto
│   │   └── cache.go                         # Cache port (redis)
│   ├── adapters/                            # infrastructure implementations
│   │   ├── http/web/                        # package web; Echo adapter
│   │   │   ├── server.go                    # Echo setup: binder, error handler, /health
│   │   │   ├── routes.go                    # MountConfig struct + Mount(e, cfg) — has crank:http-* markers
│   │   │   ├── <r>_handler.go               # HTTP handler for <r>
│   │   │   ├── auth_handler.go              # auth feature
│   │   │   └── middleware/
│   │   │       ├── auth.go                  # JWT verification
│   │   │       └── logging.go
│   │   ├── persistence/
│   │   │   ├── postgres/
│   │   │   │   ├── db.go                    # *bun.DB factory
│   │   │   │   ├── migrate.go               # migrate.New wrapper
│   │   │   │   └── <r>_repository.go        # implements domain/<r>.Repository with private <r>Row
│   │   │   └── memory/
│   │   │       └── <r>_repository.go        # in-memory implementation of the same port
│   │   ├── crypto/
│   │   │   ├── bcrypt_hasher.go             # implements ports.Hasher
│   │   │   ├── jwt_token_service.go         # implements ports.TokenService
│   │   │   └── aesgcm_cipher.go             # implements ports.Cipher
│   │   ├── cache/redis/client.go            # implements ports.Cache
│   │   ├── persistence/mongodb/client.go    # mongodb client
│   │   ├── eventbus/in_memory_eventbus.go   # implements ports.EventBus
│   │   └── temporal/
│   │       ├── worker.go                    # worker setup, marker comments for register-workflow/activity
│   │       ├── workflow/<r>.go              # temporal workflow per resource
│   │       └── activity/<r>.go              # temporal activity per resource
│   ├── config/config.go
│   └── validator/                           # shared request validation
│       ├── validator.go
│       └── errors.go
├── pkg/logging/
│   ├── logger.go
│   └── redactor.go
├── configs/config.yaml
├── .env.example
├── Makefile
├── Dockerfile
├── .air.toml
├── .gitignore
├── go.mod
├── README.md
├── .crank.yaml
├── docs/docs.go                             # swagger (SkipIfExists)
└── migrations/                              # create-table files
```

## Per-resource file mapping (for `crank make`)

For a resource named `<Resource>` (e.g. `Order`), the lowercase snake form `<r>` (`order`) is used for directory + file names; `<Resource>` for type names.

| Generated file | Source template | Notes |
|---|---|---|
| `internal/domain/<r>/<r>.go` | `domain_<r>.go.tmpl` | Aggregate + behavior |
| `internal/domain/<r>/<r>_id.go` | `domain_<r>_id.go.tmpl` | Only emitted when any field is `uuid` |
| `internal/domain/<r>/events.go` | `domain_<r>_events.go.tmpl` | Domain events |
| `internal/domain/<r>/errors.go` | `domain_<r>_errors.go.tmpl` | `Err<Resource>NotFound` |
| `internal/domain/<r>/repository.go` | `domain_<r>_repository.go.tmpl` | Port interface |
| `internal/application/<r>/commands.go` | `application_<r>_commands.go.tmpl` | |
| `internal/application/<r>/command_handler.go` | `application_<r>_command_handler.go.tmpl` | |
| `internal/application/<r>/queries.go` | `application_<r>_queries.go.tmpl` | |
| `internal/application/<r>/query_handler.go` | `application_<r>_query_handler.go.tmpl` | |
| `internal/adapters/persistence/postgres/<r>_repository.go` | `adapter_persistence_postgres_<r>_repository.go.tmpl` | Only when `--features=postgres` |
| `internal/adapters/persistence/memory/<r>_repository.go` | `adapter_persistence_memory_<r>_repository.go.tmpl` | Always |
| `internal/adapters/http/web/<r>_handler.go` | `adapter_http_<r>_handler.go.tmpl` | |
| `<r>_test.go` variants for each above | `<name>_test.go.tmpl` | When `--tests` is set; never primary; never errored if present |

`crank make` kinds map to subsets of these:
- `KindModel` → domain files only (aggregate, value objects, events, port, errors)
- `KindRepository` → persistence adapters only (port is re-emitted by KindModel if missing; otherwise skipped)
- `KindService` → application files only
- `KindHandler` → HTTP adapter only
- `KindScaffold` → everything
- `KindWorkflow` / `KindActivity` → `internal/adapters/temporal/workflow/<r>.go` and `.../activity/<r>.go` (unchanged paths, but updated to call application services)

## Code patterns

### Domain event interface (`internal/domain/shared/events.go`)

```go
package events

import "time"

type DomainEvent interface {
    EventName() string
    OccurredAt() time.Time
}
```

### Value object (typed ID)

```go
package <r>

import "errors"

type <Resource>ID string

var ErrInvalid<Resource>ID = errors.New("invalid <r> id")

func New<Resource>ID(s string) (<Resource>ID, error) {
    if s == "" {
        return "", ErrInvalid<Resource>ID
    }
    return <Resource>ID(s), nil
}

func (id <Resource>ID) String() string { return string(id) }
func (id <Resource>ID) IsZero() bool   { return id == "" }
```

### Aggregate

```go
package <r>

import (
    "time"
    "<module>/internal/domain/shared/events"
)

type <Resource> struct {
    id        <Resource>ID
    // ... other fields
    createdAt time.Time
    events    []events.DomainEvent
}

func New<Resource>(...) (*<Resource>, error) {
    if /* invariant */ { return nil, ErrInvalid<Resource> }
    x := &<Resource>{ id: id, /* ... */ createdAt: time.Now().UTC() }
    x.recordEvent(<Resource>Placed{ ... OccurredAt: x.createdAt })
    return x, nil
}

func (x *<Resource>) ID() <Resource>ID        { return x.id }
func (x *<Resource>) CreatedAt() time.Time   { return x.createdAt }
// ... other getters

func (x *<Resource>) PullEvents() []events.DomainEvent {
    out := x.events
    x.events = nil
    return out
}

func (x *<Resource>) recordEvent(e events.DomainEvent) { x.events = append(x.events, e) }
```

Aggregate has **no** `json`, `bun`, `db`, `validate` tags.

### Domain events

```go
package <r>

import "time"

type <Resource>Placed struct {
    <Resource>ID <Resource>ID
    OccurredAt   time.Time
}

func (<Resource>Placed) EventName() string  { return "<r>.placed" }
func (e <Resource>Placed) OccurredAt() time.Time { return e.OccurredAt }
```

### Repository port

```go
package <r>

import "context"

type Repository interface {
    Save(ctx context.Context, x *<Resource>) error
    Get(ctx context.Context, id <Resource>ID) (*<Resource>, error)
    List(ctx context.Context) ([]*<Resource>, error)
    Delete(ctx context.Context, id <Resource>ID) error
}
```

### Application command handler

```go
package <r>

import (
    "context"
    "fmt"
    "<module>/internal/domain/<r>"
    "<module>/internal/domain/shared/events"
    "<module>/internal/ports"
)

type Create<Resource>Command struct {
    // ... input fields (stringly; typed conversion happens in Handle)
}

type CommandHandler struct {
    repo   <r>.Repository
    events ports.EventBus
}

func NewCommandHandler(repo <r>.Repository, events ports.EventBus) *CommandHandler {
    return &CommandHandler{repo: repo, events: events}
}

func (h *CommandHandler) Handle(ctx context.Context, cmd Create<Resource>Command) (*<r>.<Resource>, error) {
    // 1. typed-ID conversion
    // 2. New<Resource>(...) — may record events
    // 3. h.repo.Save(ctx, x)
    // 4. h.events.Publish(ctx, x.PullEvents()...)
    // 5. return x, nil
}
```

### Application query handler

```go
package <r>

import (
    "context"
    "<module>/internal/domain/<r>"
)

type Get<Resource>Query struct {
    ID string
}

type QueryHandler struct {
    repo <r>.Repository
}

func NewQueryHandler(repo <r>.Repository) *QueryHandler {
    return &QueryHandler{repo: repo}
}

func (h *QueryHandler) Handle(ctx context.Context, q Get<Resource>Query) (*<r>.<Resource>, error) {
    id, err := <r>.New<Resource>ID(q.ID)
    if err != nil { return nil, fmt.Errorf("invalid id: %w", err) }
    return h.repo.Get(ctx, id)
}
```

### Postgres adapter (with private row DTO)

```go
package postgres

import (
    "context"
    "database/sql"
    "errors"
    "time"
    "github.com/uptrace/bun"
    "<module>/internal/domain/<r>"
)

type <r>Row struct {
    ID        string    `bun:"id,pk,type:uuid"`
    // ... other columns
    CreatedAt time.Time `bun:"created_at,nullzero"`
}

func (row *<r>Row) toAggregate() *<r>.<Resource> {
    id, _ := <r>.New<Resource>ID(row.ID)
    return &<r>.<Resource>{id: id, /* ... */ createdAt: row.CreatedAt}
}

func rowFromAggregate(x *<r>.<Resource>) *<r>Row {
    return &<r>Row{ID: x.ID().String(), /* ... */ CreatedAt: x.CreatedAt()}
}

type <Resource>Repository struct{ db *bun.DB }

func New<Resource>Repository(db *bun.DB) *<Resource>Repository {
    return &<Resource>Repository{db: db}
}

func (r *<Resource>Repository) Save(ctx context.Context, x *<r>.<Resource>) error {
    _, err := r.db.NewInsert().Model(rowFromAggregate(x)).
        On("CONFLICT (id) DO UPDATE SET /* ... */").Exec(ctx)
    return err
}

// Get, List, Delete follow same pattern; map sql.ErrNoRows -> <r>.Err<Resource>NotFound
```

### In-memory adapter

```go
package memory

import (
    "context"
    "sync"
    "<module>/internal/domain/<r>"
)

type <Resource>Repository struct {
    mu    sync.RWMutex
    items map[string]*<r>.<Resource>
}

func New<Resource>Repository() *<Resource>Repository {
    return &<Resource>Repository{items: make(map[string]*<r>.<Resource>)}
}

func (r *<Resource>Repository) Save(ctx context.Context, x *<r>.<Resource>) error {
    r.mu.Lock(); defer r.mu.Unlock()
    r.items[x.ID().String()] = x
    return nil
}
// ...
```

### HTTP adapter (with input/output DTO)

Package name is `web`. Import the framework as `echo`; no alias needed because the package is named `web`.

```go
package web

import (
    "errors"
    "net/http"
    "strconv"
    "github.com/labstack/echo/v4"
    "<module>/internal/application/<r>"
    "<module>/internal/domain/<r>"
    "<module>/internal/model"  // for APIError envelope
)

type <resource>DTO struct {
    ID         string `json:"id" validate:"required,uuid"`
    // ... fields
}

func (d <resource>DTO) toCreateCommand() <r>.Create<Resource>Command {
    return <r>.Create<Resource>Command{ID: d.ID, /* ... */}
}

type <Resource>Handler struct {
    cmd *<r>.CommandHandler
    qry *<r>.QueryHandler
}

func New<Resource>Handler(cmd *<r>.CommandHandler, qry *<r>.QueryHandler) *<Resource>Handler {
    return &<Resource>Handler{cmd: cmd, qry: qry}
}

func (h *<Resource>Handler) Register(g *echo.Group) {
    g.POST("", h.Create)
    g.GET("/:id", h.Get)
    g.GET("", h.List)
    g.PUT("/:id", h.Update)
    g.DELETE("/:id", h.Delete)
}

func (h *<Resource>Handler) Create(c echo.Context) error {
    var in <resource>DTO
    if err := c.Bind(&in); err != nil { return err }
    out, err := h.cmd.Handle(c.Request().Context(), in.toCreateCommand())
    if errors.Is(err, <r>.Err<Resource>NotFound) {
        return c.JSON(http.StatusNotFound, model.APIError{Error: "<r> not found"})
    }
    if err != nil {
        return c.JSON(http.StatusUnprocessableEntity, model.APIError{Error: err.Error()})
    }
    return c.JSON(http.StatusCreated, toDTO(out))
}
// Get / List / Update / Delete follow the same shape
```

## HTTP package naming

Use **`web`** as the package name. Directory: `internal/adapters/http/web/`. Avoids the name collision with `github.com/labstack/echo/v4` and with `net/http` (the framework is imported as `echo`).

## Wire marker conventions

| Old (current) | New (DDD) |
|---|---|
| `// crank:handler-fields` | `// crank:http-fields` |
| `// crank:handler-init` | *(removed — composition root in main.go handles init)* |
| `// crank:handler-register` | `// crank:http-register` |
| Target file: `internal/handler/handler.go` | Target file: `internal/adapters/http/web/routes.go` |

`routes.go` (generated by base) looks like:

```go
package web

import "github.com/labstack/echo/v4"

type MountConfig struct {
    UserHandler *UserHandler
    // crank:http-fields (do not remove — `crank make handler` inserts fields here)
}

func Mount(e *echo.Echo, cfg MountConfig) {
    g := e.Group("/users")
    cfg.UserHandler.Register(g)
    // crank:http-register (do not remove — `crank make handler` inserts route registrations here)
}
```

`crank make handler` / `crank make scaffold` splices:
- A field line `OrderHandler *OrderHandler` before `// crank:http-fields`
- A register block `g2 := e.Group("/orders"); cfg.OrderHandler.Register(g2)` before `// crank:http-register`

## Field type → value object mapping

| User type | Aggregate field type | VO file? | Notes |
|---|---|---|---|
| `string` | `string` | no | primitive |
| `text` | `string` | no | primitive |
| `int` | `int` | no | primitive |
| `int64` | `int64` | no | primitive |
| `float` | `float64` | no | primitive |
| `bool` | `bool` | no | primitive |
| `time` | `time.Time` | no | primitive |
| `uuid` | `<Resource>ID` (typed) | yes (`<resource>_id.go`) | Always emitted when at least one field is `uuid` |
| `email` | `string` (base) → `Email` (auth) | only when resource=user + auth feature | Special case |

`validate` tags live on the **DTO** in the HTTP adapter, never on the aggregate.

## Feature-to-files mapping

### `base` (foundation, always present)
- `cmd/server/main.go` (composition root)
- `internal/domain/shared/events.go`
- `internal/domain/user/{user.go,email.go,events.go,errors.go,repository.go}` (the seed aggregate; auth feature extends it)
- `internal/application/user/{commands.go,command_handler.go,queries.go,query_handler.go}`
- `internal/ports/eventbus.go`
- `internal/adapters/eventbus/in_memory_eventbus.go`
- `internal/adapters/persistence/memory/user_repository.go`
- `internal/adapters/http/web/{server.go,routes.go,user_handler.go,middleware/logging.go}`
- `internal/config/config.go`
- `internal/validator/{validator.go,errors.go}`
- `pkg/logging/{logger.go,redactor.go}`
- `configs/config.yaml`
- `.env.example`, `Makefile`, `Dockerfile`, `.air.toml`, `.gitignore`, `go.mod`, `README.md`, `.crank.yaml`
- `docs/docs.go` (skip-if-exists)

### `auth`
- `internal/domain/user/password.go` (Password value object with bcrypt helpers)
- `internal/domain/user/email.go` (overwrite base's `string` to be `Email` value object — or add a new value object if the base kept it as string)
- Update `internal/domain/user/user.go` (add `password` accessor, `Authenticate(cmd)` method, etc.)
- Update `internal/application/user/{commands.go,command_handler.go,queries.go,query_handler.go}` (Register, Authenticate, Refresh commands + handlers)
- `internal/ports/hasher.go`
- `internal/ports/tokenservice.go`
- `internal/adapters/crypto/bcrypt_hasher.go`
- `internal/adapters/crypto/jwt_token_service.go`
- `internal/adapters/http/web/auth_handler.go`
- `internal/adapters/http/web/middleware/auth.go`
- Update `cmd/server/main.go` (wire auth handler + middleware)
- Update `internal/adapters/http/web/routes.go` (add AuthHandler to MountConfig + register it)

### `postgres`
- `internal/adapters/persistence/postgres/db.go` (Bun + pgdriver setup)
- `internal/adapters/persistence/postgres/migrate.go` (migrate.New wrapper)
- `internal/adapters/persistence/postgres/user_repository.go` (overwrites in-memory in the composition root)
- Update `cmd/server/main.go` (constructs DB, swaps userRepo to postgres impl)
- `migrations/<timestamp>_create_users.up.sql` + `.down.sql`

### `redis`
- `internal/ports/cache.go`
- `internal/adapters/cache/redis/client.go`
- Update `cmd/server/main.go` (constructs redis client, exposes via DI in composition root)

### `mongodb`
- `internal/adapters/persistence/mongodb/client.go`
- Update `cmd/server/main.go` (constructs mongo client; not necessarily wired to any aggregate by default)

### `temporal`
- `internal/adapters/temporal/worker.go` (replaces the current `internal/temporal/worker.go`)
- `internal/adapters/temporal/workflow/<r>.go` (per workflow; emitted by `crank make workflow`)
- `internal/adapters/temporal/activity/<r>.go` (per activity; emitted by `crank make activity`)
- Update `cmd/server/main.go` (starts worker)

### `crypto`
- `internal/ports/cipher.go`
- `internal/adapters/crypto/aesgcm_cipher.go`
- (No wiring in `cmd/server/main.go` by default — application services that want to use the cipher inject it.)

## What does NOT change in crank's own code

- `cmd/crank/main.go` (CLI root; only registers features/commands)
- `internal/bootstrap/feature.go`, `tool.go`
- `internal/bootstrap/registry.go`, `tool_registry.go`
- `internal/bootstrap/manifest.go`, `project.go`
- `internal/bootstrap/context.go` (template context — no new fields needed for DDD)
- `internal/bootstrap/result.go`
- `internal/bootstrap/gomod.go`
- `internal/bootstrap/commands/{list,tools,makedelegate}.go`
- `internal/bootstrap/commands/make.go` (just calls `scaffold.Generate` — minimal/no change)
- `internal/bootstrap/commands/init.go` (no change — uses feature files)
- `internal/bootstrap/commands/add.go` (no change — uses feature files)
- `internal/bootstrap/config_inject.go` (config still in `configs/`)
- `internal/bootstrap/tools/**` (no change)
- `internal/utils/**` (no change)
- `crank/internal/bootstrap/registry_test.go`, `context_test.go`, `manifest_test.go`, `result_test.go` (no change)
- `crank/internal/bootstrap/scaffold/names_test.go` (no change)

## What DOES change in crank's own code

- `crank/internal/bootstrap/scaffold/scaffold.go` (new `tmplData`, `buildPlan`, `Generate`)
- `crank/internal/bootstrap/scaffold/wire.go` (new marker names, new target file)
- `crank/internal/bootstrap/scaffold/fields.go` (uuid → value object auto-emission)
- `crank/internal/bootstrap/scaffold/names.go` (only if path conventions need it)
- `crank/internal/bootstrap/scaffold/templates/**` (full template rewrite)
- `crank/internal/bootstrap/features/base/**` (full template rewrite)
- `crank/internal/bootstrap/features/{auth,postgres,redis,mongodb,temporal,crypto}/**` (full template rewrite)

## Test impact (for wave-3 reference, do NOT touch in waves 1–2)

- `crank/internal/bootstrap/scaffold/scaffold_test.go` — path assertions
- `crank/internal/bootstrap/scaffold/scaffold_temporal_test.go` — path assertions
- `crank/internal/bootstrap/integration_test.go` — extensive content assertions, all referenced paths change
- `crank/e2e/e2e_test.go` — `compileCases` and `allFeatureNames` lists still apply; the assertions on generated file content need updates

## Validation strategy

After Wave 1:
1. `cd crank && go build ./...` — must succeed
2. Smoke: `cd crank && go test ./internal/bootstrap/scaffold/...` (expect failures in path assertions; note them; do not fix)
3. Smoke: scaffold a base project (manually or via test) and verify the project compiles + the composition root is well-formed
4. Smoke: run `crank make handler Order` on a base project; verify the new handler file is generated AND wired into `routes.go`

After Wave 2:
1. `cd crank && go build ./...`
2. For each feature, scaffold a project with that feature and verify the generated project compiles
3. Verify the composition root in main.go correctly wires all features together

After Wave 3:
1. `cd crank && go test ./internal/...` — must pass
2. `cd crank && go test -tags e2e ./e2e/...` — must pass
3. End-to-end smoke: `crank init test --features=base,auth,postgres`, then `cd test && go build ./... && go vet ./...`
