# PRD: crank Framework Upgrades from surge

> Backport patterns proven in surge into crank's code generation templates.

---

## Feature 1: Two-Process Architecture (Temporal)

### Problem

When `temporal` is enabled, `cmd/server/main.go` starts a worker in a goroutine
that registers every workflow and activity — the same set as the dedicated
`cmd/worker/main.go` process. This means:

- The HTTP server imports heavy activity dependencies (DB repos, platform
  clients) just for the in-process worker.
- Activity failures can crash the API process.
- The server blocks on worker initialization before serving traffic.
- No clear separation between server-appropriate and worker-appropriate deps.

### Solution

Split the worker contract into two modes:

| Process | Worker | Activities |
|---------|--------|------------|
| `cmd/server/main.go` | In-process goroutine | `nil` (none) |
| `cmd/worker/main.go` | Standalone binary | Full set via `activity.NewActivities(...)` |

### Template Changes

**`cmd/server/main.go.tmpl`** (when `temporal` enabled):
```go
// In-process worker with NO activities — runs only sample workflows.
w := temporal.NewWorker(tc, cfg.Temporal, nil)
```

**`cmd/worker/main.go.tmpl`** (already exists, update signature):
```go
// Worker process: connects DB, builds activity deps, runs real activities.
gormDB, _ := gorm.NewDB(cfg.Database)
agentRepo := gorm.NewAgentRepository(gormDB)
runRepo := gorm.NewAgentRunRepository(gormDB)
acts := activity.NewActivities(agentRepo, runRepo)
w := temporal.NewWorker(tc, cfg.Temporal, acts)
```

**`internal/adapters/temporal/worker.go.tmpl`** — Change `NewWorker` to accept
an `*activity.Activities` parameter instead of always registering all activities:
```go
func NewWorker(c client.Client, cfg config.TemporalConfig, acts *activity.Activities) worker.Worker {
    w := worker.New(c, cfg.TaskQueue, worker.Options{})
    w.RegisterWorkflow(workflow.GreetingWorkflow)
    // crank:workflow-register
    if acts != nil {
        acts.Register(w)
    }
    return w
}
```

**`internal/adapters/temporal/activity/`** — New `activities.go` wrapper:
```go
type Activities struct {
    AgentRepo domain.AgentRepository
    RunRepo   domain.RunRepository
}
func NewActivities(agentRepo, runRepo) *Activities { ... }
func (a *Activities) Register(w worker.Worker) {
    w.RegisterActivity(a.CheckRunStatus)
    w.RegisterActivity(a.ExecuteAgentRun)
    // crank:activity-register
}
```

### Docker Changes

- `Dockerfile` — single-stage build for `cmd/server` (unchanged)
- `Dockerfile.worker` — new multi-stage build for `cmd/worker`
- `docker-compose.yml` with profiles — add `worker` service when temporal
  enabled (example in PRD appendix)

### Migration

Existing projects: `crank update` re-renders `cmd/server/main.go` and
`cmd/worker/main.go`. The `NewWorker` signature change is a compile error —
forces the dev to update. The `activity.Activities` struct is backward-compat:
generated `crank make activity` splices into `activities.go` instead of
`worker.go`.

---

## Feature 2: Platform Client Pattern

### Problem

Crank generates raw external-service clients directly in their adapter
packages (e.g., `internal/adapters/cache/redis/client.go`,
`internal/adapters/persistence/qdrant/client.go`). Each client reinvents:
HTTP setup, timeout handling, error wrapping, health checks. No shared
pattern for nil-guarding, mocking, or interface-based testing.

### Solution

Introduce a `internal/adapters/platform/` package (shared HTTP helper) and
`internal/ports/platform/` interfaces — exactly as surge does. This is a
**refactoring of existing client generation**, not a new feature flag.

### Template Changes

**New: `internal/adapters/platform/client.go.tmpl`** — Base HTTP helper:
```go
type Client struct {
    BaseURL string
    Resty   *resty.Client  // or HTTPClient interface for mocking
}
func NewClient(baseURL string) *Client {
    return &Client{
        BaseURL: strings.TrimRight(baseURL, "/"),
        Resty:   resty.New().SetTimeout(5 * time.Second).SetBaseURL(baseURL),
    }
}
func (c *Client) Health(ctx, path) error { ... }
func (c *Client) GetJSON(ctx, path, out) error { ... }
```

**New: `internal/ports/platform/` — One file per external service:**
```
internal/ports/platform/
├── qdrant.go      // Qdrant interface: Health, CollectionExists, UpsertPoint, SearchPoints, DeletePoint
├── cache.go       // Cache interface: Get, Set, Del
├── mongodb.go     // MongoDB interface (if needed)
└── bifrost.go     // Bifrost interface (example for future use)
```

Each interface defines only the methods the application layer needs — no
HTTP types leak into ports.

**Refactored: Existing client adapters embed `platform.Client`:**

`internal/adapters/persistence/qdrant/client.go.tmpl` becomes:
```go
type QdrantClient struct{ *platform.Client }
var _ platform.Qdrant = (*QdrantClient)(nil)

func NewQdrantClient(cfg config.QdrantConfig) *QdrantClient {
    return &QdrantClient{Client: platform.NewClient(cfg.Addr)}
}
func (c *QdrantClient) Health(ctx context.Context) error { ... }
func (c *QdrantClient) UpsertPoint(ctx, collection, pointID, vector, payload) error { ... }
// etc.
```

**`cmd/server/main.go.tmpl`** — Wire clients through `platform.NewClient`:
```go
{{if .Has "qdrant"}}
platformClients.Qdrant = qdrantclient.NewQdrantClient(cfg.Qdrant)
{{end}}
// PlatformClients struct (optional holder for dependency injection):
type PlatformClients struct {
    Qdrant  platform.Qdrant
    Cache   platform.Cache
    MongoDB platform.MongoDB
}
```

### Migration

`crank update` on existing projects leaves existing client files untouched
(`SkipIfExists: true` on old paths). The new `platform.Client` and port
interfaces are additive — projects can adopt per-service. The `crank doctor`
tool gains a check: "platform clients use typed interfaces."

---

## Feature 3: Logging with ContextHandler (not L(ctx))

### Problem

Crank's generated logging uses an `L(ctx)` function pattern — the caller must
remember to call `logging.L(ctx)` to get a logger enriched with request-scoped
values. Third-party code, deferred calls, and helper functions that don't have
access to `ctx` miss the enrichment entirely.

```
// Cranks's approach — manual enrichment:
reqLog := logging.L(ctx)
reqLog.Info("doing work")  // has request_id

someHelper() // no request_id
```

### Solution

Adopt surge's `ContextHandler` pattern — a `slog.Handler` wrapper that
automatically injects `request_id` and `user_id` from the Go context into
EVERY log record, regardless of who calls `slog.Default()` or
`slog.With()`.

### Template Changes

**`pkg/logging/logger.go.tmpl`** — Replace `L(ctx)` pattern with
`ContextHandler` wrapper:

```go
type ContextHandler struct {
    slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
        r.AddAttrs(slog.String("request_id", id))
    }
    if uid, ok := ctx.Value(userIDKey).(string); ok && uid != "" {
        r.AddAttrs(slog.String("user_id", uid))
    }
    return h.Handler.Handle(ctx, r)
}
```

Handler stack (innermost to outermost):
```
JSONHandler → redactionHandler → ContextHandler → slog.New()
```

The `New()` constructor:
```go
func New(level slog.Leveler, addSource bool) *slog.Logger {
    jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
    redactHandler := &redactionHandler{handler: jsonHandler}
    ctxHandler := &ContextHandler{Handler: redactHandler}
    return slog.New(ctxHandler)
}
```

The `L(ctx)` function is **removed** — callers just use `slog.Default()`,
`slog.Info()`, etc. Context enrichment happens automatically as long as the
context is passed through.

Keep `WithRequestID`, `WithUserID`, `RequestIDFromContext`,
`UserIDFromContext` for middleware usage.

Replace `logging.L(c.Request().Context())` calls in the HTTP error handler
and middleware with direct `slog` calls:
```go
// Before:
reqLog := logging.L(c.Request().Context())
reqLog.Warn("validation failed", "errors", ve.Errors)

// After:
slog.WarnContext(c.Request().Context(), "validation failed", "errors", ve.Errors)
```

### Advantages over current approach

| Concern | L(ctx) | ContextHandler |
|---------|--------|----------------|
| Third-party code | No enrichment | Automatic enrichment |
| Deferred goroutines | No enrichment | Automatic enrichment (if ctx propagated) |
| Forget to call L(ctx) | Missing request_id | Impossible — always injected |
| slog.Default() | No request_id | Always has request_id |
| Testability | Manual context wiring | Standard slog test helpers work |

---

## Feature 4: Audit Trail Feature

### Problem

Crank has no mechanism to track entity-level event history. Domains generate
events (via shared.DomainEvent) and publish them through the EventBus, but
there is no persistence layer for querying "what happened to this entity over
time." Surge's `agent_run_events` table solves this — generalized, it becomes
a reusable crank feature.

### Solution

Add an `audit` feature that generates:

1. An event log table (`domain_events` or `entity_events`)
2. A domain aggregate + port interface
3. A GORM/Bun repository
4. A query handler for listing events by entity
5. A migration pair

The audit feature **integrates with the EventBus** — it subscribes to all
domain events and persists them. This is the "write side" of the outbox
pattern (outbox publishes; audit logs).

### Template Changes

**New: `internal/domain/audit/` — Event log aggregate:**
```go
type AuditEvent struct {
    ID        string
    EntityType string   // e.g. "user", "agent"
    EntityID   string
    EventType  string   // e.g. "user.created"
    Payload    json.RawMessage
    OccurredAt time.Time
}
```

**New: `internal/ports/audit.go` — Port interface:**
```go
type AuditStore interface {
    Append(ctx, events ...shared.DomainEvent) error
    ListByEntity(ctx, entityType, entityID string) ([]AuditEvent, error)
}
```

**New: Adapter — subscribes to EventBus, persists to DB:**
```go
type AuditLogger struct {
    repo domain.AuditRepository
}
func (a *AuditLogger) Subscribe(bus *eventbus.InMemory) {
    bus.Subscribe(func(ctx context.Context, ev shared.DomainEvent) error {
        return a.repo.Save(ctx, toRow(ev))
    })
}
```

**New: `internal/application/audit/query_handler.go`:**
```go
type QueryHandler struct {
    repo domain.AuditRepository
}
func (h *QueryHandler) ListByEntity(ctx, entityType, entityID string) ([]AuditEvent, error)
```

**`cmd/server/main.go.tmpl`** (when `audit` enabled):
```go
// Wire audit logger into event bus.
auditLogger := auditadapter.NewLogger(auditRepo)
auditLogger.Subscribe(bus)
```

**Migration:** `migrations/XXXXXXXXXX_add_audit_events.up.sql`:
```sql
CREATE TABLE audit_events (
    id          VARCHAR(255) PRIMARY KEY,
    entity_type VARCHAR(128) NOT NULL,
    entity_id   VARCHAR(255) NOT NULL,
    event_type  VARCHAR(255) NOT NULL,
    payload     JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_entity ON audit_events (entity_type, entity_id, occurred_at DESC);
```

### Dependencies / Requirements

- Requires `bun` or `gorm` (needs a database)
- Does NOT require `outbox` — they are complementary (outbox publishes, audit
  logs). Can be used together or independently.

### Config additions

```
# crank:config-section
audit:
  enabled: true
```

---

## Feature 5: UoW — Log Publish Failures (small refinement)

### Problem

Crank's `InMemoryUoW.SaveAndPublish` swallows publish errors silently (uses
`_ = u.bus.Publish(...)`). Surge's version logs them.

### Fix

One-line change in `internal/adapters/uow/in_memory_uow.go.tmpl`:

```go
// Before:
_ = u.bus.Publish(ctx, events...)

// After:
if err := u.bus.Publish(ctx, events...); err != nil {
    slog.ErrorContext(ctx, "failed to publish domain events", "error", err)
}
```

Also the method comment — clarify that the save is NOT unwound on publish
failure (intentional, matches surge's behavior).

---

## Implementation Order

| Step | Feature | Dependencies | Effort |
|------|---------|-------------|--------|
| 1 | Logging: ContextHandler | None (templates only) | Small |
| 2 | UoW: log publish errors | None (templates only) | Trivial |
| 3 | Two-process arch | Temporal feature | Medium |
| 4 | Platform client pattern | Qdrant, Redis, MongoDB features | Large |
| 5 | Audit trail | Base (EventBus + DB) | Medium |

Steps 1-2 are template-only changes and ship independently. Step 3 refactors
existing temporal templates. Step 4 is the biggest surface area (touches every
external-service adapter). Step 5 is a new feature package.

---

## Appendix: Docker Compose with Worker Service

When `temporal` is enabled, generate `docker-compose.yml` with:

```yaml
services:
  postgres: ...
  temporal: ...
  temporal-ui: ...
  server:
    build:
      dockerfile: Dockerfile
    depends_on: [postgres, temporal]
  worker:
    build:
      dockerfile: Dockerfile.worker
    depends_on: [postgres, temporal]
    profiles: [worker]  # opt-in by default
```

With profile handling for dev (`docker compose --profile worker up`) vs
production (always-on).
