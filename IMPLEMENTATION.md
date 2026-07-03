# Engineering Implementation Plan: crank Framework Upgrades

> Implementation spec for backporting surge patterns into crank's code generation templates.
> PRD: `PRD.md` in this repo.

---

## Table of Contents

1. [Logging: ContextHandler](#1-logging-contexthandler)
2. [UoW: Log Publish Failures](#2-uow-log-publish-failures)
3. [Two-Process Architecture](#3-two-process-architecture)
4. [Platform Client Pattern](#4-platform-client-pattern)
5. [Audit Trail Feature](#5-audit-trail-feature)
6. [Integration Tests](#6-integration-tests)
7. [E2E Tests](#7-e2e-tests)
8. [Implementation Order](#8-implementation-order)
9. [Risk Assessment](#9-risk-assessment)

---

## 1. Logging: ContextHandler

**Goal:** Replace the `L(ctx)` function pattern with a `slog.Handler` wrapper that auto-injects `request_id` and `user_id` from context into every log record.

### Files to Modify

#### 1.1 `internal/bootstrap/features/base/templates/pkg_logging_logger.go.tmpl`

**Current state:** Defines `New()` which returns `slog.New(&redactionHandler{handler: handler})`. Also defines `L(ctx)` which manually pulls request_id/user_id from context and calls `slog.Default().With(...)`.

**Changes:**

Remove the `L(ctx)` function entirely. Add a `ContextHandler` type that wraps an inner `slog.Handler` and injects context values in `Handle()`. Update `New()` to wrap the handler stack:

```
JSONHandler → redactionHandler → ContextHandler → slog.New()
```

**New template content (replaces entire file):**

```go
package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// ContextHandler wraps a slog.Handler and automatically injects
// request_id and user_id from the Go context into every log record.
// This means any code calling slog.Info(), slog.Default(), etc. with a
// context-aware call (e.g. slog.InfoContext) gets enrichment for free —
// no need to remember to call a helper function.
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

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// New creates a slog.Logger with a three-layer handler stack:
//
//	JSONHandler → redactionHandler → ContextHandler
//
// The JSON handler emits structured output, the redaction handler scrubs
// sensitive values, and the context handler auto-injects request_id and
// user_id from the Go context. When addSource is true every log entry
// includes a compressed source location (e.g. "handler/user.go:42").
func New(level slog.Leveler, addSource bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(shortSource(src))
				}
			}
			return a
		},
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	redactHandler := &redactionHandler{handler: jsonHandler}
	ctxHandler := &ContextHandler{Handler: redactHandler}

	return slog.New(ctxHandler)
}

// shortSource compresses a full source path to the last two directory
// components plus the file name: "/app/internal/handler/user.go" →
// "handler/user.go". If the path is shorter than three segments it is
// returned unchanged.
func shortSource(src *slog.Source) string {
	parts := strings.Split(src.File, string(filepath.Separator))
	if len(parts) <= 2 {
		return src.File
	}
	short := filepath.Join(parts[len(parts)-2:]...)
	return short + ":" + itoa(src.Line)
}

// itoa converts a positive int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithUserID stores the authenticated user ID in the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// RequestIDFromContext returns the request ID stored in the context, or an
// empty string if none is present.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// UserIDFromContext returns the user ID stored in the context.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
```

**Key changes from current:**
- `L(ctx)` function removed entirely
- `ContextHandler` struct added with `Handle`, `WithAttrs`, `WithGroup`
- `New()` wraps `jsonHandler → redactHandler → ctxHandler`
- Context helper functions (`WithRequestID`, `WithUserID`, `RequestIDFromContext`, `UserIDFromContext`) retained for middleware use

#### 1.2 `internal/bootstrap/features/base/templates/internal_adapters_http_web_middleware_logging.go.tmpl`

**Current state:** Uses `logging.L(ctx)` to get a request-scoped logger, then calls `reqLogger.LogAttrs(ctx, level, msg, attrs...)`.

**Changes:** Replace `logging.L(ctx)` with `slog.Default()` (or just use `slog.LogAttrs` directly). Since the `ContextHandler` is now in the handler stack, `slog.LogAttrs(ctx, level, msg, attrs...)` will automatically inject `request_id`.

**Specific edits:**

Line 45: Remove `reqLogger := logging.L(ctx)` — no longer needed.

Line 83: Change `reqLogger.LogAttrs(ctx, level, msg, attrs...)` to `slog.LogAttrs(ctx, level, msg, attrs...)`.

Add `"log/slog"` to imports if not already present (it is — used for `slog.LevelInfo` etc.).

The middleware template diff:

```go
// Before:
reqLogger := logging.L(ctx)
// ... handler runs ...
reqLogger.LogAttrs(ctx, level, msg, attrs...)

// After:
// (reqLogger removed — slog.Default() already has ContextHandler)
// ... handler runs ...
slog.LogAttrs(ctx, level, msg, attrs...)
```

#### 1.3 `internal/bootstrap/features/base/templates/cmd_server_main.go.tmpl`

**Current state:** Line 216: `reqLog := logging.L(c.Request().Context())` in the HTTP error handler.

**Changes:** Replace with `slog` calls using context-aware variants:

```go
// Before:
reqLog := logging.L(c.Request().Context())
if ve, ok := err.(*validator.ValidationError); ok {
    reqLog.Warn("validation failed", "errors", ve.Errors)
    // ...
}
reqLog.Error("unhandled error", "error", err)

// After:
ctx := c.Request().Context()
if ve, ok := err.(*validator.ValidationError); ok {
    slog.WarnContext(ctx, "validation failed", "errors", ve.Errors)
    // ...
}
slog.ErrorContext(ctx, "unhandled error", "error", err)
```

**Note:** `slog.WarnContext` and `slog.ErrorContext` automatically pick up `request_id` from context via the `ContextHandler`. No need for `L(ctx)`.

#### 1.4 `internal/bootstrap/features/temporal/templates/cmd_worker_main.go.tmpl`

**Current state:** Defines its own `parseLevel` function (lines 52-61).

**Changes:** Remove the local `parseLevel` — use `logging.ParseLevel` instead. Add `"{{.ModulePath}}/pkg/logging"` import (already present). Change:

```go
// Before:
logger := logging.New(parseLevel(cfg.Logging.Level), cfg.Logging.AddSource)

// After:
logger := logging.New(logging.ParseLevel(cfg.Logging.Level), cfg.Logging.AddSource)
```

Remove the `parseLevel` function entirely. Add `ParseLevel` to `pkg/logging/logger.go.tmpl` (it was in surge's version but missing from crank):

**Add to `pkg_logging_logger.go.tmpl`:**

```go
// ParseLevel converts a config string to a slog.Level. Defaults to Info.
func ParseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

#### 1.5 `internal/bootstrap/features/base/templates/cmd_server_main.go.tmpl`

**Current state:** Lines 83-84: `level := parseLevel(cfg.Logging.Level)` and line 267-278: local `parseLevel` function.

**Changes:** Replace local `parseLevel` with `logging.ParseLevel`:

```go
// Before:
level := parseLevel(cfg.Logging.Level)
logger := logging.New(level, cfg.Logging.AddSource)

// After:
logger := logging.New(logging.ParseLevel(cfg.Logging.Level), cfg.Logging.AddSource)
```

Remove the `parseLevel` function at lines 267-278.

---

## 2. UoW: Log Publish Failures

**Goal:** Stop silently swallowing EventBus publish errors in `InMemoryUoW`.

### Files to Modify

#### 2.1 `internal/bootstrap/features/base/templates/internal_adapters_uow_in_memory_uow.go.tmpl`

**Current state (line 36):**
```go
_ = u.bus.Publish(ctx, events...)
```

**Change to:**
```go
if err := u.bus.Publish(ctx, events...); err != nil {
    slog.ErrorContext(ctx, "failed to publish domain events", "error", err)
}
```

Add `"log/slog"` to the import block.

**Full updated template:**

```go
package uow

import (
	"context"
	"log/slog"

	"{{.ModulePath}}/internal/domain/shared"
	"{{.ModulePath}}/internal/ports"
)

type InMemoryUoW struct {
	bus ports.EventBus
}

func NewInMemoryUoW(bus ports.EventBus) *InMemoryUoW {
	return &InMemoryUoW{bus: bus}
}

func (u *InMemoryUoW) SaveAndPublish(ctx context.Context, save func(ctx context.Context) error, events []shared.DomainEvent) error {
	if err := save(ctx); err != nil {
		return err
	}
	if u.bus != nil && len(events) > 0 {
		if err := u.bus.Publish(ctx, events...); err != nil {
			slog.ErrorContext(ctx, "failed to publish domain events", "error", err)
		}
	}
	return nil
}
```

---

## 3. Two-Process Architecture

**Goal:** Split the Temporal worker contract so `cmd/server` runs a worker with NO activities (sample workflows only) and `cmd/worker` runs the real worker with all activities.

### Files to Modify

#### 3.1 `internal/bootstrap/features/temporal/templates/internal_adapters_temporal_worker.go.tmpl`

**Current state:** `NewWorker(c client.Client, cfg config.TemporalConfig) worker.Worker` — registers all workflows and activities unconditionally.

**Changes:** Add a third parameter `acts *activity.Activities` (nil for server, non-nil for worker). Only register activities when `acts != nil`.

**New template content:**

```go
package temporal

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"{{.ModulePath}}/internal/adapters/temporal/activity"
	"{{.ModulePath}}/internal/adapters/temporal/workflow"
	"{{.ModulePath}}/internal/config"
)

// NewClient constructs a Temporal client using the application config. The
// caller is responsible for calling Close during shutdown.
func NewClient(cfg config.TemporalConfig, logger *slog.Logger) (client.Client, error) {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
		Logger:    slogAdapter{logger: logger},
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewWorker creates a Temporal worker bound to the configured task queue.
//
// Pass acts=nil for the in-process server worker (sample workflows only, no
// activities). Pass a non-nil acts for the dedicated worker process that
// registers all real activities.
//
// Workflows generated with `crank make workflow` are wired at the
// crank:workflow-register marker. Activities generated with `crank make
// activity` are registered on the Acts struct via the crank:activity-register
// marker — do not remove either marker.
func NewWorker(c client.Client, cfg config.TemporalConfig, acts *activity.Activities) worker.Worker {
	w := worker.New(c, cfg.TaskQueue, worker.Options{})
	registerWorkflows(w)
	if acts != nil {
		acts.Register(w)
	}
	return w
}

func registerWorkflows(w worker.Worker) {
	w.RegisterWorkflow(workflow.GreetingWorkflow)
	// crank:workflow-register (do not remove — `crank make workflow` inserts registrations here)
}
```

**Key changes:**
- `NewWorker` signature: `NewWorker(c, cfg)` → `NewWorker(c, cfg, acts *activity.Activities)`
- `registerActivities` function removed — activities now registered via `acts.Register(w)`
- Activity registration moved to the `Activities` struct

#### 3.2 New file: `internal/bootstrap/features/temporal/templates/internal_adapters_temporal_activity_activities.go.tmpl`

**Purpose:** An `Activities` container struct that holds activity dependencies and registers all activities with a worker. Generated activities splice their registration at the `// crank:activity-register` marker inside `Register()`.

**Template content:**

```go
package activity

import "go.temporal.io/sdk/worker"

// Activities is a container for all activity dependencies and their
// registrations. The dedicated worker process (cmd/worker/main.go)
// constructs this with real repository/platform clients; the API server
// passes nil to run only sample workflows.
//
// Add activity fields here and register them in Register(). Activities
// generated with `crank make activity` are wired at the marker below.
type Activities struct {
	// crank:activity-fields (do not remove — `crank make activity` inserts fields here)
}

// NewActivities constructs an Activities container. Callers add fields
// directly on the struct after construction, or future constructors can
// accept typed dependencies.
func NewActivities() *Activities {
	return &Activities{}
}

// Register registers all activities with the given worker.
func (a *Activities) Register(w worker.Worker) {
	w.RegisterActivity(Greet)
	// crank:activity-register (do not remove — `crank make activity` inserts registrations here)
}
```

#### 3.3 `internal/bootstrap/features/temporal/feature.go`

**Changes:** Add the new activities template to the `Files()` slice:

```go
func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_temporal_logger.go.tmpl", OutputPath: "internal/adapters/temporal/logger.go"},
		{TemplatePath: "templates/internal_adapters_temporal_worker.go.tmpl", OutputPath: "internal/adapters/temporal/worker.go"},
		{TemplatePath: "templates/internal_adapters_temporal_activity_activities.go.tmpl", OutputPath: "internal/adapters/temporal/activity/activities.go"},
		{TemplatePath: "templates/internal_adapters_temporal_workflow_greeting.go.tmpl", OutputPath: "internal/adapters/temporal/workflow/greeting.go"},
		{TemplatePath: "templates/internal_adapters_temporal_activity_greeting.go.tmpl", OutputPath: "internal/adapters/temporal/activity/greeting.go"},
		{TemplatePath: "templates/cmd_worker_main.go.tmpl", OutputPath: "cmd/worker/main.go"},
	}
}
```

#### 3.4 `internal/bootstrap/features/temporal/templates/cmd_worker_main.go.tmpl`

**Current state:** Calls `temporal.NewWorker(c, cfg.Temporal)` — no activities passed.

**Changes:** Construct an `activity.NewActivities()` and pass it to `NewWorker`. Also use `logging.ParseLevel` instead of local `parseLevel`.

**New template content:**

```go
package main

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/worker"

	"{{.ModulePath}}/internal/adapters/temporal"
	"{{.ModulePath}}/internal/adapters/temporal/activity"
	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/pkg/logging"
)

// The Temporal worker process. It connects to Temporal, registers all
// workflows and activities (see internal/adapters/temporal/worker.go) and
// polls the configured task queue until interrupted (Ctrl+C / SIGTERM).
//
// Run it alongside the API server with:
//
//	go run ./cmd/worker
func main() {
	cfg := config.Load()

	logger := logging.New(logging.ParseLevel(cfg.Logging.Level), cfg.Logging.AddSource)
	slog.SetDefault(logger)

	logger.Info("starting temporal worker",
		"host_port", cfg.Temporal.HostPort,
		"namespace", cfg.Temporal.Namespace,
		"task_queue", cfg.Temporal.TaskQueue,
	)

	c, err := temporal.NewClient(cfg.Temporal, logger)
	if err != nil {
		logger.Error("failed to connect to temporal", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	acts := activity.NewActivities()
	w := temporal.NewWorker(c, cfg.Temporal, acts)

	logger.Info("temporal worker running; press Ctrl+C to stop", "task_queue", cfg.Temporal.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Error("temporal worker stopped with error", "error", err)
		os.Exit(1)
	}
}
```

#### 3.5 `internal/bootstrap/features/base/templates/cmd_server_main.go.tmpl`

**Current state (lines 153-166):** When temporal is enabled, starts a worker goroutine with `temporal.NewWorker(tc, cfg.Temporal)`.

**Changes:** Pass `nil` as the activities argument to indicate the server worker runs sample workflows only:

```go
{{if .Has "temporal"}}
	tc, err := temporal.NewClient(cfg.Temporal, logger)
	if err != nil {
		logger.Error("failed to connect to temporal", "error", err)
		os.Exit(1)
	}
	defer tc.Close()
	go func() {
		w := temporal.NewWorker(tc, cfg.Temporal, nil)
		if err := w.Run(worker.InterruptCh()); err != nil {
			logger.Error("temporal worker stopped with error", "error", err)
		}
	}()
{{end}}
```

**Diff:** `temporal.NewWorker(tc, cfg.Temporal)` → `temporal.NewWorker(tc, cfg.Temporal, nil)`

#### 3.6 New file: `internal/bootstrap/features/temporal/templates/Dockerfile.worker.tmpl`

**Purpose:** Separate Dockerfile for the worker process.

**Template content:**

```dockerfile
# syntax=docker/dockerfile:1.6
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/worker /app/worker
COPY configs/config.yaml /app/configs/config.yaml
USER nonroot:nonroot
ENTRYPOINT ["/app/worker"]
```

Add to `feature.go` Files():

```go
{TemplatePath: "templates/Dockerfile.worker.tmpl", OutputPath: "Dockerfile.worker"},
```

#### 3.7 `internal/bootstrap/scaffold/wire_temporal.go`

**Current state:** `wireActivity` splices `w.RegisterActivity(activity.<Pascal>Activity)` into `worker.go` at the `// crank:activity-register` marker.

**Changes:** The marker has moved from `worker.go` (inside `registerActivities`) to `activity/activities.go` (inside `Register`). Update:

- `workerFile` constant stays `internal/adapters/temporal/worker.go` for workflows
- New `activitiesFile` constant: `internal/adapters/temporal/activity/activities.go` for activities
- `wireActivity` targets `activitiesFile` and splices into the `Register` method

**Updated `wire_temporal.go`:**

```go
const (
	workerFile       = "internal/adapters/temporal/worker.go"
	activitiesFile   = "internal/adapters/temporal/activity/activities.go"
	markerWorkflowRegister = "// crank:workflow-register"
	markerActivityRegister = "// crank:activity-register"
)

func wireActivity(projectDir string, r Resource) (wireResult, error) {
	reg := fmt.Sprintf("w.RegisterActivity(activity.%sActivity)", r.Pascal)
	hint := fmt.Sprintf(`could not auto-register the activity in %s. Add this to Activities.Register():

  %s`, activitiesFile, reg)
	return wireWorker(projectDir, markerActivityRegister, reg, hint, activitiesFile)
}

func wireWorker(projectDir, marker, regLine, hint, targetFile string) (wireResult, error) {
	path := filepath.Join(projectDir, targetFile)
	// ... rest unchanged, uses targetFile instead of workerFile
}
```

**Also:** `wireWorkflow` stays targeting `workerFile`:

```go
func wireWorkflow(projectDir string, r Resource) (wireResult, error) {
	reg := fmt.Sprintf("w.RegisterWorkflow(workflow.%sWorkflow)", r.Pascal)
	hint := fmt.Sprintf(`could not auto-register the workflow in %s. Add this to registerWorkflows():

  %s`, workerFile, reg)
	return wireWorker(projectDir, markerWorkflowRegister, reg, hint, workerFile)
}
```

#### 3.8 New marker: `// crank:activity-fields`

The `activities.go` template has a `// crank:activity-fields` marker for adding dependency fields to the `Activities` struct. This is used when an activity needs repository or platform client dependencies.

Currently `crank make activity` generates standalone activity functions. In the future, activities that need dependencies would add a field to `Activities` and reference it. For now, the marker exists but the generator doesn't splice into it — it's for manual use. The scaffold generator only splices `w.RegisterActivity(...)` into `Register()`.

---

## 4. Platform Client Pattern

**Goal:** Introduce a shared HTTP client helper (`internal/adapters/platform/`) and typed port interfaces (`internal/ports/platform/`) so external service clients follow a consistent pattern: nil-safe, testable, health-checkable.

This is a **new feature** called `platform` that provides the base infrastructure. Existing features (qdrant, redis, mongodb) are refactored to use the pattern.

### Architecture

```
internal/ports/platform/       # Port interfaces (pure Go, no HTTP types)
├── qdrant.go                  # Qdrant interface
├── cache.go                   # Cache interface (moved from ports/cache.go)
└── mongodb.go                 # MongoDB interface

internal/adapters/platform/    # Adapter implementations
├── client.go                  # Base HTTP helper (resty, 5s timeout, nil-safe)
├── qdrant.go                  # QdrantClient embedding *Client
├── redis.go                   # RedisClient (wraps go-redis, implements Cache)
└── mongodb.go                 # MongoDBClient (wraps mongo-driver)
```

### New Feature: `platform`

#### 4.1 `internal/bootstrap/features/platform/feature.go`

```go
package platform

import (
	"embed"

	"github.com/anurag925/crank/internal/bootstrap"
)

//go:embed templates/*.tmpl
var tmpls embed.FS

type feature struct{}

func init() {
	if err := bootstrap.GlobalRegistry.Register(feature{}); err != nil {
		panic(err)
	}
}

func (feature) Name() string { return "platform" }
func (feature) Description() string {
	return "Platform client pattern: shared HTTP helper + typed port interfaces for external services"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/go-resty/resty/v2",
	}
}

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_adapters_platform_client.go.tmpl", OutputPath: "internal/adapters/platform/client.go"},
		{TemplatePath: "templates/internal_ports_platform_types.go.tmpl", OutputPath: "internal/ports/platform/types.go"},
	}
}
```

#### 4.2 `internal/bootstrap/features/platform/templates/internal_adapters_platform_client.go.tmpl`

```go
package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client is a small HTTP helper used by every typed platform client. It owns
// only the minimum surface: request building, timeouts, status checks, and
// JSON decoding. It does not retry or do tracing.
type Client struct {
	BaseURL string
	Resty   *resty.Client
}

// NewClient builds a Client with a 5-second timeout. An empty baseURL is
// allowed; the caller is responsible for guarding the client when it is empty.
func NewClient(baseURL string) *Client {
	r := resty.New().
		SetTimeout(5 * time.Second).
		SetBaseURL(strings.TrimRight(baseURL, "/"))

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Resty:   r,
	}
}

// Health returns nil if the server responds with a success-ish status and an
// error otherwise. Honours the OpenBao-style "sealed/uninitialized" convention
// where 501 and 503 still count as alive.
func (c *Client) Health(ctx context.Context, path string) error {
	if c == nil || c.BaseURL == "" {
		return fmt.Errorf("platform: base url is empty")
	}

	resp, err := c.Resty.R().SetContext(ctx).Get("/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return err
	}

	statusCode := resp.StatusCode()
	if statusCode >= 200 && statusCode < 400 {
		return nil
	}
	if statusCode == http.StatusServiceUnavailable || statusCode == http.StatusNotImplemented {
		return nil
	}

	url := c.BaseURL + "/" + strings.TrimLeft(path, "/")
	return fmt.Errorf("platform: %s returned %d", url, statusCode)
}

// GetJSON issues a GET and decodes JSON into out. Does not follow redirects.
func (c *Client) GetJSON(ctx context.Context, path string, out any) error {
	if c == nil || c.BaseURL == "" {
		return fmt.Errorf("platform: base url is empty")
	}

	resp, err := c.Resty.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetResult(out).
		Get("/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return err
	}

	if resp.IsError() {
		url := c.BaseURL + "/" + strings.TrimLeft(path, "/")
		return fmt.Errorf("platform: %s returned %d", url, resp.StatusCode())
	}
	return nil
}
```

#### 4.3 `internal/bootstrap/features/platform/templates/internal_ports_platform_types.go.tmpl`

```go
package platform

// This package defines port interfaces for external platform services.
// Each interface exposes only the minimum surface the application layer
// needs — no HTTP, resty, or library types leak into ports.
```

(This is a placeholder file to create the `internal/ports/platform/` package. Feature-specific interfaces are added by their respective features below.)

#### 4.4 Register in `cmd/crank/main.go`

Add blank import:

```go
_ "github.com/anurag925/crank/internal/bootstrap/features/platform"
```

#### 4.5 Refactor: Qdrant feature

**`internal/bootstrap/features/qdrant/feature.go`** — Add platform as a dependency, add port interface template, refactor client to use `platform.Client`:

```go
func (feature) Dependencies() []string {
	return []string{
		"github.com/qdrant/go-client",
		"github.com/go-resty/resty/v2",
	}
}

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_ports_platform_qdrant.go.tmpl", OutputPath: "internal/ports/platform/qdrant.go"},
		{TemplatePath: "templates/internal_adapters_platform_qdrant.go.tmpl", OutputPath: "internal/adapters/platform/qdrant.go"},
	}
}
```

**New: `internal/bootstrap/features/qdrant/templates/internal_ports_platform_qdrant.go.tmpl`:**

```go
package platform

import "context"

// Qdrant is the internal client for the Qdrant vector store. Only the
// minimum surface used by the application layer is exposed.
type Qdrant interface {
	Health(ctx context.Context) error
	CollectionExists(ctx context.Context, name string) (bool, error)
	EnsureCollection(ctx context.Context, name string, vectorSize int) error
	UpsertPoint(ctx context.Context, collection string, pointID string, vector []float32, payload map[string]any) error
	SearchPoints(ctx context.Context, collection string, vector []float32, limit int) ([]SearchResult, error)
	DeletePoint(ctx context.Context, collection string, pointID string) error
}

type SearchResult struct {
	PointID string
	Score   float32
	Payload map[string]any
}
```

**New: `internal/bootstrap/features/qdrant/templates/internal_adapters_platform_qdrant.go.tmpl`:**

```go
package platform

import (
	"context"
	"fmt"
	"net/http"

	"{{.ModulePath}}/internal/ports/platform"
)

// QdrantClient is the typed HTTP client for the Qdrant vector store.
type QdrantClient struct{ *Client }

func NewQdrantClient(baseURL string) *QdrantClient {
	return &QdrantClient{Client: NewClient(baseURL)}
}

// Compile-time assertion that QdrantClient satisfies the port.
var _ platform.Qdrant = (*QdrantClient)(nil)

func (c *QdrantClient) Health(ctx context.Context) error {
	if err := c.Client.Health(ctx, "/readyz"); err == nil {
		return nil
	}
	return c.Client.Health(ctx, "/")
}

func (c *QdrantClient) CollectionExists(ctx context.Context, name string) (bool, error) {
	if c == nil || c.BaseURL == "" {
		return false, fmt.Errorf("qdrant: base url is empty")
	}
	resp, err := c.Resty.R().SetContext(ctx).Get("/collections/" + name)
	if err != nil {
		return false, err
	}
	switch resp.StatusCode() {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("qdrant: %s returned %d", c.BaseURL+"/collections/"+name, resp.StatusCode())
	}
}

func (c *QdrantClient) EnsureCollection(ctx context.Context, name string, vectorSize int) error {
	exists, err := c.CollectionExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	body := map[string]any{
		"vectors": map[string]any{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	resp, err := c.Resty.R().SetContext(ctx).SetBody(body).Put("/collections/" + name)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("qdrant: EnsureCollection %s returned %d: %s", name, resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *QdrantClient) UpsertPoint(ctx context.Context, collection string, pointID string, vector []float32, payload map[string]any) error {
	req := map[string]any{
		"points": []map[string]any{
			{"id": pointID, "vector": vector, "payload": payload},
		},
	}
	resp, err := c.Resty.R().SetContext(ctx).SetBody(req).Put(fmt.Sprintf("/collections/%s/points", collection))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("qdrant: UpsertPoint returned %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *QdrantClient) SearchPoints(ctx context.Context, collection string, vector []float32, limit int) ([]platform.SearchResult, error) {
	req := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	var out struct {
		Result []struct {
			ID      string         `json:"id"`
			Score   float32        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	resp, err := c.Resty.R().SetContext(ctx).SetBody(req).SetResult(&out).Post(fmt.Sprintf("/collections/%s/points/search", collection))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("qdrant: SearchPoints returned %d: %s", resp.StatusCode(), resp.String())
	}
	results := make([]platform.SearchResult, 0, len(out.Result))
	for _, r := range out.Result {
		results = append(results, platform.SearchResult{
			PointID: r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		})
	}
	return results, nil
}

func (c *QdrantClient) DeletePoint(ctx context.Context, collection string, pointID string) error {
	req := map[string]any{"points": []string{pointID}}
	resp, err := c.Resty.R().SetContext(ctx).SetBody(req).Post(fmt.Sprintf("/collections/%s/points/delete", collection))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("qdrant: DeletePoint returned %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}
```

**Remove old file:** `internal/bootstrap/features/qdrant/templates/internal_adapters_persistence_qdrant_client.go.tmpl` — replaced by the platform pattern. The old `qdrant.NewClient(ctx, cfg)` constructor using the gRPC client is replaced by `platform.NewQdrantClient(cfg.Qdrant.Host + ":" + port)`.

**Config injection:** Update the qdrant entry in `config_inject.go` to add a `Addr` field (combined host:port string) alongside the existing `Host`/`Port` fields, or keep them separate and construct the URL in `cmd/server/main.go`.

**`cmd_server_main.go.tmpl` changes:**

```go
{{if .Has "qdrant"}}
	qdrantClient := platform.NewQdrantClient(fmt.Sprintf("http://%s:%d", cfg.Qdrant.Host, cfg.Qdrant.Port))
{{end}}
```

#### 4.6 Refactor: Redis feature

**`internal/bootstrap/features/redis/feature.go`** — Keep the existing `internal/ports/cache.go` port (it's already good). Add a platform-compatible adapter that implements `ports.Cache`:

**New file: `internal/bootstrap/features/redis/templates/internal_adapters_platform_redis.go.tmpl`:**

```go
package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"{{.ModulePath}}/internal/ports"
)

// RedisCache implements ports.Cache using go-redis. It is nil-safe: a nil
// receiver returns ("", false, nil) from Get and nil from Set/Del.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

var _ ports.Cache = (*RedisCache)(nil)

func (c *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	if c == nil || c.client == nil {
		return "", false, nil
	}
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("redis get %s: %w", key, err)
	}
	return val, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set %s: %w", key, err)
	}
	return nil
}

func (c *RedisCache) Del(ctx context.Context, keys ...string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, nil
	}
	n, err := c.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis del: %w", err)
	}
	return n, nil
}
```

**Update `redis/feature.go` Files():**

```go
func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_ports_cache.go.tmpl", OutputPath: "internal/ports/cache.go"},
		{TemplatePath: "templates/internal_adapters_cache_redis_client.go.tmpl", OutputPath: "internal/adapters/cache/redis/client.go"},
		{TemplatePath: "templates/internal_adapters_platform_redis.go.tmpl", OutputPath: "internal/adapters/platform/redis.go"},
	}
}
```

#### 4.7 MongoDB — Keep as-is for now

MongoDB uses a native Go driver (not HTTP), so the platform client pattern (resty-based) doesn't apply directly. The MongoDB client stays as `internal/adapters/persistence/mongodb/client.go`. A port interface can be added later if needed.

#### 4.8 `cmd_server_main.go.tmpl` — Wire platform clients

Add a `PlatformClients` holder struct and wire clients when features are enabled:

```go
{{if or (.Has "qdrant") (.Has "redis") (.Has "mongodb")}}
	type platformClients struct {
		{{if .Has "qdrant"}}Qdrant  platform.Qdrant{{end}}
		{{if .Has "redis"}}Cache   ports.Cache{{end}}
	}
{{end}}
```

This is optional — the composition root can wire clients directly into handlers without a holder struct. The holder is a convenience for projects with many platform services.

#### 4.9 Config injection for `platform` feature

Add entry in `config_inject.go`'s `featureConfigData()`:

```go
"platform": {
    // No config of its own — platform clients use the config of their
    // respective features (qdrant, redis, etc.). This entry exists so
    // crank add platform is a no-op for config injection.
    StructField: "",
    StructDef:   "",
    Defaults:    "",
    YAMLSection: "",
    EnvSection:  "",
},
```

Actually, `platform` has no config of its own — it's a base infrastructure feature. The `injectConfig` function already handles unknown features gracefully (returns nil). So no entry needed in `featureConfigData`.

---

## 5. Audit Trail Feature

**Goal:** New `audit` feature that persists domain events to a database table, queryable by entity type and ID. Subscribes to the EventBus and logs every event.

### New Feature: `audit`

#### 5.1 `internal/bootstrap/features/audit/feature.go`

```go
package audit

import (
	"embed"

	"github.com/anurag925/crank/internal/bootstrap"
)

//go:embed templates/*.tmpl
var tmpls embed.FS

type feature struct{}

func init() {
	if err := bootstrap.GlobalRegistry.Register(feature{}); err != nil {
		panic(err)
	}
}

func (feature) Name() string { return "audit" }
func (feature) Description() string {
	return "Audit trail: persists domain events to a database table, queryable by entity type and ID"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string {
	return []string{
		"github.com/google/uuid",
	}
}

func (feature) Requirements() []string {
	return []string{"bun", "gorm"}
```

}

```go
func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		{TemplatePath: "templates/internal_domain_audit_event.go.tmpl", OutputPath: "internal/domain/audit/event.go"},
		{TemplatePath: "templates/internal_domain_audit_repository.go.tmpl", OutputPath: "internal/domain/audit/repository.go"},
		{TemplatePath: "templates/internal_ports_audit.go.tmpl", OutputPath: "internal/ports/audit.go"},
		{TemplatePath: "templates/internal_adapters_persistence_bun_audit_repository.go.tmpl", OutputPath: "internal/adapters/persistence/bun/audit_repository.go", Requires: "bun"},
		{TemplatePath: "templates/internal_adapters_persistence_gorm_audit_repository.go.tmpl", OutputPath: "internal/adapters/persistence/gorm/audit_repository.go", Requires: "gorm"},
		{TemplatePath: "templates/internal_adapters_audit_logger.go.tmpl", OutputPath: "internal/adapters/audit/logger.go"},
		{TemplatePath: "templates/internal_application_audit_query_handler.go.tmpl", OutputPath: "internal/application/audit/query_handler.go"},
		{TemplatePath: "templates/internal_adapters_http_web_audit_handler.go.tmpl", OutputPath: "internal/adapters/http/web/audit_handler.go"},
		{TemplatePath: "templates/migrations_000003_add_audit_events.up.sql.tmpl", OutputPath: "migrations/000003_add_audit_events.up.sql"},
		{TemplatePath: "templates/migrations_000003_add_audit_events.down.sql.tmpl", OutputPath: "migrations/000003_add_audit_events.down.sql"},
	}
}
```

#### 5.2 Templates

**`internal_domain_audit_event.go.tmpl`:**

```go
package audit

import (
	"time"
)

// AuditEvent is a persisted record of a domain event. It is the queryable
// audit trail — every event published through the EventBus is appended
// here by the audit logger.
type AuditEvent struct {
	id          string
	entityType  string
	entityID    string
	eventType   string
	payload     []byte
	occurredAt  time.Time
}

func NewAuditEvent(id, entityType, entityID, eventType string, payload []byte, occurredAt time.Time) AuditEvent {
	return AuditEvent{
		id:         id,
		entityType: entityType,
		entityID:   entityID,
		eventType:  eventType,
		payload:    payload,
		occurredAt: occurredAt,
	}
}

func (e AuditEvent) ID() string         { return e.id }
func (e AuditEvent) EntityType() string { return e.entityType }
func (e AuditEvent) EntityID() string   { return e.entityID }
func (e AuditEvent) EventType() string  { return e.eventType }
func (e AuditEvent) Payload() []byte    { return e.payload }
func (e AuditEvent) OccurredAt() time.Time { return e.occurredAt }
```

**`internal_domain_audit_repository.go.tmpl`:**

```go
package audit

import "context"

// Repository is the port for persisting and querying audit events.
type Repository interface {
	Append(ctx context.Context, events ...AuditEvent) error
	ListByEntity(ctx context.Context, entityType, entityID string) ([]AuditEvent, error)
}
```

**`internal_ports_audit.go.tmpl`:**

```go
package ports

import (
	"context"

	"{{.ModulePath}}/internal/domain/audit"
)

// AuditStore is the application-layer port for querying audit events.
// The audit feature provides a GORM/Bun implementation.
type AuditStore interface {
	ListByEntity(ctx context.Context, entityType, entityID string) ([]audit.AuditEvent, error)
}
```

**`internal_adapters_persistence_gorm_audit_repository.go.tmpl`:**

```go
package gorm

import (
	"context"
	"time"

	"{{.ModulePath}}/internal/domain/audit"

	"gorm.io/gorm"
)

type auditRow struct {
	ID         string    `gorm:"column:id;primaryKey;type:varchar(255)"`
	EntityType string    `gorm:"column:entity_type;type:varchar(128);index"`
	EntityID   string    `gorm:"column:entity_id;type:varchar(255);index"`
	EventType  string    `gorm:"column:event_type;type:varchar(255)"`
	Payload    []byte    `gorm:"column:payload;type:jsonb"`
	OccurredAt time.Time `gorm:"column:occurred_at"`
}

func (auditRow) TableName() string { return "audit_events" }

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

var _ audit.Repository = (*AuditRepository)(nil)

func (r *AuditRepository) Append(ctx context.Context, events ...audit.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([]auditRow, 0, len(events))
	for _, e := range events {
		rows = append(rows, auditRow{
			ID:         e.ID(),
			EntityType: e.EntityType(),
			EntityID:   e.EntityID(),
			EventType:  e.EventType(),
			Payload:    e.Payload(),
			OccurredAt: e.OccurredAt(),
		})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *AuditRepository) ListByEntity(ctx context.Context, entityType, entityID string) ([]audit.AuditEvent, error) {
	var rows []auditRow
	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("occurred_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	events := make([]audit.AuditEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, audit.NewAuditEvent(
			row.ID, row.EntityType, row.EntityID, row.EventType, row.Payload, row.OccurredAt,
		))
	}
	return events, nil
}
```

**`internal_adapters_persistence_bun_audit_repository.go.tmpl`:**

```go
package bun

import (
	"context"
	"database/sql"
	"time"

	"{{.ModulePath}}/internal/domain/audit"

	"github.com/uptrace/bun"
)

type auditRow struct {
	bun.BaseModel `bun:"table:audit_events"`

	ID         string    `bun:"id,pk,type:varchar(255)"`
	EntityType string    `bun:"entity_type,type:varchar(128)"`
	EntityID   string    `bun:"entity_id,type:varchar(255)"`
	EventType  string    `bun:"event_type,type:varchar(255)"`
	Payload    []byte    `bun:"payload,type:jsonb"`
	OccurredAt time.Time `bun:"occurred_at"`
}

type AuditRepository struct {
	db *bun.DB
}

func NewAuditRepository(db *bun.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

var _ audit.Repository = (*AuditRepository)(nil)

func (r *AuditRepository) Append(ctx context.Context, events ...audit.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([]auditRow, 0, len(events))
	for _, e := range events {
		rows = append(rows, auditRow{
			ID:         e.ID(),
			EntityType: e.EntityType(),
			EntityID:   e.EntityID(),
			EventType:  e.EventType(),
			Payload:    e.Payload(),
			OccurredAt: e.OccurredAt(),
		})
	}
	_, err := r.db.NewInsert().Model(&rows).Exec(ctx)
	return err
}

func (r *AuditRepository) ListByEntity(ctx context.Context, entityType, entityID string) ([]audit.AuditEvent, error) {
	var rows []auditRow
	err := r.db.NewSelect().
		Model(&rows).
		Where("entity_type = ?", entityType).
		Where("entity_id = ?", entityID).
		OrderExpr("occurred_at DESC").
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	events := make([]audit.AuditEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, audit.NewAuditEvent(
			row.ID, row.EntityType, row.EntityID, row.EventType, row.Payload, row.OccurredAt,
		))
	}
	return events, nil
}
```

**`internal_adapters_audit_logger.go.tmpl`:**

```go
package audit

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"{{.ModulePath}}/internal/domain/audit"
	"{{.ModulePath}}/internal/domain/shared"
)

// Logger subscribes to the EventBus and persists every domain event as an
// AuditEvent. It is the write side of the audit trail.
type Logger struct {
	repo audit.Repository
}

func NewLogger(repo audit.Repository) *Logger {
	return &Logger{repo: repo}
}

// Subscribe registers the logger with the in-memory event bus. Every
// published domain event is encoded and appended to the audit table.
func (l *Logger) Subscribe(bus interface{ Subscribe(func(context.Context, shared.DomainEvent) error) }) {
	bus.Subscribe(func(ctx context.Context, ev shared.DomainEvent) error {
		payload, err := shared.EncodeEvent(ev)
		if err != nil {
			slog.ErrorContext(ctx, "audit: failed to encode event", "event", ev.EventName(), "error", err)
			return nil
		}
		auditEvent := audit.NewAuditEvent(
			uuid.NewString(),
			entityTypeFromEvent(ev),
			entityIDFromEvent(ev),
			ev.EventName(),
			payload,
			ev.OccurredAt(),
		)
		if err := l.repo.Append(ctx, auditEvent); err != nil {
			slog.ErrorContext(ctx, "audit: failed to append event", "event", ev.EventName(), "error", err)
		}
		return nil
	})
}

// entityTypeFromEvent extracts the entity type from the event name
// (e.g. "user.created" → "user").
func entityTypeFromEvent(ev shared.DomainEvent) string {
	name := ev.EventName()
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			return name[:i]
		}
	}
	return name
}

// entityIDFromEvent best-effort extracts the aggregate ID from the event
// payload. Falls back to empty string if not found.
func entityIDFromEvent(ev shared.DomainEvent) string {
	// The encoded payload is a JSON envelope; the aggregate ID is typically
	// in the event body. This is a best-effort extraction — domain events
	// with a struct ID field will have it in the JSON.
	return ""
}
```

**`internal_application_audit_query_handler.go.tmpl`:**

```go
package audit

import (
	"context"

	"{{.ModulePath}}/internal/domain/audit"
)

type QueryHandler struct {
	repo audit.Repository
}

func NewQueryHandler(repo audit.Repository) *QueryHandler {
	return &QueryHandler{repo: repo}
}

func (h *QueryHandler) ListByEntity(ctx context.Context, entityType, entityID string) ([]audit.AuditEvent, error) {
	return h.repo.ListByEntity(ctx, entityType, entityID)
}
```

**`internal_adapters_http_web_audit_handler.go.tmpl`:**

```go
package web

import (
	"net/http"

	"github.com/labstack/echo/v5"

	auditapp "{{.ModulePath}}/internal/application/audit"
)

type AuditHandler struct {
	qry *auditapp.QueryHandler
}

func NewAuditHandler(qry *auditapp.QueryHandler) *AuditHandler {
	return &AuditHandler{qry: qry}
}

func (h *AuditHandler) Register(g *echo.Group) {
	g.GET("/audit/events", h.ListByEntity)
}

func (h *AuditHandler) ListByEntity(c *echo.Context) error {
	entityType := c.QueryParam("entity_type")
	entityID := c.QueryParam("entity_id")
	if entityType == "" || entityID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "entity_type and entity_id query params are required")
	}
	events, err := h.qry.ListByEntity(c.Request().Context(), entityType, entityID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, events)
}
```

**`migrations_000003_add_audit_events.up.sql.tmpl`:**

```sql
CREATE TABLE IF NOT EXISTS audit_events (
    id          VARCHAR(255) PRIMARY KEY,
    entity_type VARCHAR(128) NOT NULL,
    entity_id   VARCHAR(255) NOT NULL,
    event_type  VARCHAR(255) NOT NULL,
    payload     JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_events (entity_type, entity_id, occurred_at DESC);
```

**`migrations_000003_add_audit_events.down.sql.tmpl`:**

```sql
DROP TABLE IF EXISTS audit_events;
```

#### 5.3 Config injection

Add entry in `config_inject.go`:

```go
"audit": {
    StructField: "",
    StructDef:   "",
    Defaults:    "",
    YAMLSection: "",
    EnvSection:  "",
},
```

(Audit has no config of its own — it uses the existing database config.)

#### 5.4 `cmd_server_main.go.tmpl` — Wire audit logger

Add conditional wiring when audit is enabled:

```go
{{if .Has "audit"}}
{{if .Has "gorm"}}
	auditRepo := gorm.NewAuditRepository(gormDB)
{{else if .Has "bun"}}
	auditRepo := bun.NewAuditRepository(db)
{{end}}
	auditLogger := auditadapter.NewLogger(auditRepo)
	auditLogger.Subscribe(bus)
	auditQry := auditapp.NewQueryHandler(auditRepo)
	auditHandler := web.NewAuditHandler(auditQry)
{{end}}
```

Add imports:

```go
{{if .Has "audit"}}
	auditadapter "{{.ModulePath}}/internal/adapters/audit"
	auditapp "{{.ModulePath}}/internal/application/audit"
{{end}}
```

Add to `MountConfig` and `Mount` call (wired via `// crank:http-fields` and `// crank:http-register` markers).

#### 5.5 Register in `cmd/crank/main.go`

```go
_ "github.com/anurag925/crank/internal/bootstrap/features/audit"
```

---

## 6. Integration Tests

Add tests to `internal/bootstrap/integration_test.go`:

### 6.1 Logging tests

```go
func TestBase_Logging_ContextHandler(t *testing.T) {
	t.Run("ContextHandler injected in New()", func(t *testing.T) {
		project := generateProject(t, "base")
		logger := readFile(t, project, "pkg/logging/logger.go")
		assertContains(t, logger, "ContextHandler")
		assertContains(t, logger, "func (h *ContextHandler) Handle")
		assertNotContains(t, logger, "func L(")
	})

	t.Run("no L(ctx) in middleware", func(t *testing.T) {
		project := generateProject(t, "base")
		mw := readFile(t, project, "internal/adapters/http/web/middleware/logging.go")
		assertNotContains(t, mw, "logging.L(")
		assertContains(t, mw, "slog.LogAttrs(ctx")
	})

	t.Run("no L(ctx) in main.go error handler", func(t *testing.T) {
		project := generateProject(t, "base")
		main := readFile(t, project, "cmd/server/main.go")
		assertNotContains(t, main, "logging.L(")
		assertContains(t, main, "slog.WarnContext")
		assertContains(t, main, "slog.ErrorContext")
	})

	t.Run("ParseLevel exists", func(t *testing.T) {
		project := generateProject(t, "base")
		logger := readFile(t, project, "pkg/logging/logger.go")
		assertContains(t, logger, "func ParseLevel")
	})

	t.Run("no parseLevel in main.go", func(t *testing.T) {
		project := generateProject(t, "base")
		main := readFile(t, project, "cmd/server/main.go")
		assertNotContains(t, main, "func parseLevel")
		assertContains(t, main, "logging.ParseLevel")
	})
}
```

### 6.2 UoW tests

```go
func TestBase_UoW_LogsPublishErrors(t *testing.T) {
	project := generateProject(t, "base")
	uow := readFile(t, project, "internal/adapters/uow/in_memory_uow.go")
	assertContains(t, uow, "slog.ErrorContext")
	assertContains(t, uow, "failed to publish domain events")
	assertNotContains(t, uow, "_ = u.bus.Publish")
}
```

### 6.3 Two-process arch tests

```go
func TestTemporal_TwoProcess_WorkerSignature(t *testing.T) {
	project := generateProject(t, "base", "gorm", "temporal")

	worker := readFile(t, project, "internal/adapters/temporal/worker.go")
	assertContains(t, worker, "acts *activity.Activities")
	assertContains(t, worker, "if acts != nil")
	assertContains(t, worker, "acts.Register(w)")
	assertNotContains(t, worker, "func registerActivities")

	activities := readFile(t, project, "internal/adapters/temporal/activity/activities.go")
	assertContains(t, activities, "type Activities struct")
	assertContains(t, activities, "func (a *Activities) Register")
	assertContains(t, activities, "// crank:activity-register")

	workerMain := readFile(t, project, "cmd/worker/main.go")
	assertContains(t, workerMain, "activity.NewActivities()")
	assertContains(t, workerMain, "temporal.NewWorker(c, cfg.Temporal, acts)")
	assertNotContains(t, workerMain, "func parseLevel")

	serverMain := readFile(t, project, "cmd/server/main.go")
	assertContains(t, serverMain, "temporal.NewWorker(tc, cfg.Temporal, nil)")
}

func TestTemporal_DockerfileWorker(t *testing.T) {
	project := generateProject(t, "base", "gorm", "temporal")
	assertFileExists(t, project, "Dockerfile.worker")
	dockerfile := readFile(t, project, "Dockerfile.worker")
	assertContains(t, dockerfile, "go build -o /out/worker ./cmd/worker")
}
```

### 6.4 Platform client tests

```go
func TestPlatform_FilesExist(t *testing.T) {
	project := generateProject(t, "base", "platform")
	assertFileExists(t, project, "internal/adapters/platform/client.go")
	assertFileExists(t, project, "internal/ports/platform/types.go")

	client := readFile(t, project, "internal/adapters/platform/client.go")
	assertContains(t, client, "type Client struct")
	assertContains(t, client, "func NewClient(baseURL string) *Client")
	assertContains(t, client, "func (c *Client) Health(ctx context.Context, path string) error")
	assertContains(t, client, "func (c *Client) GetJSON(ctx context.Context, path string, out any) error")
	assertContains(t, client, "5 * time.Second")
}

func TestQdrant_PlatformPattern(t *testing.T) {
	project := generateProject(t, "base", "gorm", "platform", "qdrant")
	assertFileExists(t, project, "internal/ports/platform/qdrant.go")
	assertFileExists(t, project, "internal/adapters/platform/qdrant.go")

	port := readFile(t, project, "internal/ports/platform/qdrant.go")
	assertContains(t, port, "type Qdrant interface")
	assertContains(t, port, "Health(ctx context.Context) error")
	assertContains(t, port, "UpsertPoint")
	assertContains(t, port, "SearchPoints")

	adapter := readFile(t, project, "internal/adapters/platform/qdrant.go")
	assertContains(t, adapter, "type QdrantClient struct{ *Client }")
	assertContains(t, adapter, "var _ platform.Qdrant = (*QdrantClient)(nil)")
}

func TestRedis_PlatformPattern(t *testing.T) {
	project := generateProject(t, "base", "gorm", "platform", "redis")
	assertFileExists(t, project, "internal/adapters/platform/redis.go")
	assertFileExists(t, project, "internal/ports/cache.go")

	adapter := readFile(t, project, "internal/adapters/platform/redis.go")
	assertContains(t, adapter, "type RedisCache struct")
	assertContains(t, adapter, "var _ ports.Cache = (*RedisCache)(nil)")
}
```

### 6.5 Audit trail tests

```go
func TestAudit_FilesExist(t *testing.T) {
	project := generateProject(t, "base", "gorm", "audit")
	assertFileExists(t, project, "internal/domain/audit/event.go")
	assertFileExists(t, project, "internal/domain/audit/repository.go")
	assertFileExists(t, project, "internal/ports/audit.go")
	assertFileExists(t, project, "internal/adapters/persistence/gorm/audit_repository.go")
	assertFileExists(t, project, "internal/adapters/audit/logger.go")
	assertFileExists(t, project, "internal/application/audit/query_handler.go")
	assertFileExists(t, project, "internal/adapters/http/web/audit_handler.go")
	assertFileExists(t, project, "migrations/000003_add_audit_events.up.sql")
	assertFileExists(t, project, "migrations/000003_add_audit_events.down.sql")
}

func TestAudit_BunRepository(t *testing.T) {
	project := generateProject(t, "base", "bun", "audit")
	assertFileExists(t, project, "internal/adapters/persistence/bun/audit_repository.go")
}

func TestAudit_NoORM_Fails(t *testing.T) {
	_, err := generateProjectErr(t, "base", "audit")
	require.Error(t, err)
	assertContains(t, err.Error(), "requires a database ORM")
}

func TestAudit_LoggerSubscribes(t *testing.T) {
	project := generateProject(t, "base", "gorm", "audit")
	logger := readFile(t, project, "internal/adapters/audit/logger.go")
	assertContains(t, logger, "func (l *Logger) Subscribe")
	assertContains(t, logger, "bus.Subscribe")
}

func TestAudit_MigrationContent(t *testing.T) {
	project := generateProject(t, "base", "gorm", "audit")
	up := readFile(t, project, "migrations/000003_add_audit_events.up.sql")
	assertContains(t, up, "CREATE TABLE IF NOT EXISTS audit_events")
	assertContains(t, up, "entity_type")
	assertContains(t, up, "entity_id")
	assertContains(t, up, "event_type")
	assertContains(t, up, "payload")
	assertContains(t, up, "idx_audit_entity")
}

func TestAudit_MainWiring(t *testing.T) {
	project := generateProject(t, "base", "gorm", "audit")
	main := readFile(t, project, "cmd/server/main.go")
	assertContains(t, main, "auditadapter")
	assertContains(t, main, "auditLogger.Subscribe(bus)")
	assertContains(t, main, "NewAuditHandler")
}

func TestAudit_OutboxCoexistence(t *testing.T) {
	project := generateProject(t, "base", "gorm", "audit", "outbox")
	assertFileExists(t, project, "internal/adapters/audit/logger.go")
	assertFileExists(t, project, "internal/adapters/outbox/worker.go")
}
```

---

## 7. E2E Tests

Update `e2e/e2e_test.go`:

### 7.1 Add to `allFeatureNames`

```go
var allFeatureNames = []string{
	"base", "auth", "crypto", "bun", "gorm", "redis", "mongodb",
	"qdrant", "temporal", "otel", "outbox", "platform", "audit",
}
```

### 7.2 Add compile cases

```go
var compileCases = []string{
	"base_only", "auth", "bun", "gorm", "redis", "mongodb",
	"qdrant", "crypto", "temporal", "auth_bun_crypto",
	"platform", "platform_qdrant", "platform_redis",
	"audit_gorm", "audit_bun", "all",
}
```

### 7.3 Add temporal two-process e2e test

```go
func TestE2E_TwoProcessArchitecture(t *testing.T) {
	dir := generateProject(t, "base", "gorm", "temporal")

	// Verify worker.go has nil activities
	worker := readFile(t, dir, "internal/adapters/temporal/worker.go")
	assertContains(t, worker, "acts *activity.Activities")

	// Verify cmd/server passes nil
	serverMain := readFile(t, dir, "cmd/server/main.go")
	assertContains(t, serverMain, "NewWorker(tc, cfg.Temporal, nil)")

	// Verify cmd/worker passes acts
	workerMain := readFile(t, dir, "cmd/worker/main.go")
	assertContains(t, workerMain, "NewWorker(c, cfg.Temporal, acts)")

	// Compile both binaries
	compileProject(t, dir)
}
```

### 7.4 Add audit e2e test

```go
func TestE2E_AuditTrail(t *testing.T) {
	dir := generateProject(t, "base", "gorm", "audit")
	compileProject(t, dir)

	// Verify migration exists
	assertExists(t, dir, "migrations/000003_add_audit_events.up.sql")
}
```

### 7.5 Update `allFeaturesMinusGorm`

```go
func allFeaturesMinusGorm() []string {
	features := []string{}
	for _, f := range allFeatureNames {
		if f != "gorm" {
			features = append(features, f)
		}
	}
	return features
}
```

### 7.6 Add blank imports

```go
import (
	_ "github.com/anurag925/crank/internal/bootstrap/features/audit"
	_ "github.com/anurag925/crank/internal/bootstrap/features/platform"
)
```

---

## 8. Implementation Order

| Phase | Feature | Files Changed/Created | Depends On | Risk |
|-------|---------|----------------------|------------|------|
| **1** | Logging: ContextHandler | 4 templates modified | None | Low — template-only |
| **2** | UoW: log publish errors | 1 template modified | None | Trivial |
| **3** | Two-process arch | 5 templates modified, 2 created, 1 Go file | Phase 1 (for `ParseLevel`) | Medium — signature change |
| **4** | Platform client pattern | 1 new feature (2 files), 2 features refactored (4 files) | None | Medium — new dependency |
| **5** | Audit trail | 1 new feature (10 files), 1 template modified, config entry | Phase 4 (uses platform) | Medium — new feature |

### Phase 1: Logging (ship first)

**Files:**
- `features/base/templates/pkg_logging_logger.go.tmpl` — rewrite
- `features/base/templates/internal_adapters_http_web_middleware_logging.go.tmpl` — edit
- `features/base/templates/cmd_server_main.go.tmpl` — edit (L(ctx) → slog)
- `features/temporal/templates/cmd_worker_main.go.tmpl` — edit (parseLevel → ParseLevel)

**Validation:** `go test ./internal/...`, `go test -tags e2e ./e2e/... -run TestE2E_GenerateAndCompile`

### Phase 2: UoW (ship with Phase 1)

**Files:**
- `features/base/templates/internal_adapters_uow_in_memory_uow.go.tmpl` — edit

**Validation:** `go test ./internal/...`

### Phase 3: Two-process arch (ship after Phase 1)

**Files:**
- `features/temporal/templates/internal_adapters_temporal_worker.go.tmpl` — rewrite
- `features/temporal/templates/internal_adapters_temporal_activity_activities.go.tmpl` — new
- `features/temporal/templates/cmd_worker_main.go.tmpl` — rewrite
- `features/temporal/templates/Dockerfile.worker.tmpl` — new
- `features/temporal/feature.go` — edit (add file mappings)
- `features/base/templates/cmd_server_main.go.tmpl` — edit (nil activities)
- `scaffold/wire_temporal.go` — edit (activities target file)

**Validation:** `go test ./internal/... -run TestTemporal`, `go test -tags e2e ./e2e/... -run TestE2E_TwoProcess`

### Phase 4: Platform client (ship after Phase 3)

**Files:**
- `features/platform/feature.go` — new
- `features/platform/templates/internal_adapters_platform_client.go.tmpl` — new
- `features/platform/templates/internal_ports_platform_types.go.tmpl` — new
- `features/qdrant/feature.go` — edit
- `features/qdrant/templates/internal_ports_platform_qdrant.go.tmpl` — new
- `features/qdrant/templates/internal_adapters_platform_qdrant.go.tmpl` — new
- `features/qdrant/templates/internal_adapters_persistence_qdrant_client.go.tmpl` — remove
- `features/redis/feature.go` — edit
- `features/redis/templates/internal_adapters_platform_redis.go.tmpl` — new
- `features/base/templates/cmd_server_main.go.tmpl` — edit (wire platform clients)
- `cmd/crank/main.go` — edit (blank import)

**Validation:** `go test ./internal/... -run TestPlatform`, `go test -tags e2e ./e2e/... -run TestE2E_GenerateAndCompile`

### Phase 5: Audit trail (ship after Phase 4)

**Files:**
- `features/audit/feature.go` — new
- `features/audit/templates/` — 10 new template files
- `features/base/templates/cmd_server_main.go.tmpl` — edit (wire audit)
- `config_inject.go` — edit (add audit entry)
- `cmd/crank/main.go` — edit (blank import)

**Validation:** `go test ./internal/... -run TestAudit`, `go test -tags e2e ./e2e/... -run TestE2E_AuditTrail`

---

## 9. Risk Assessment

### Risk 1: `NewWorker` signature breakage

**Impact:** Existing projects generated with the old `NewWorker(c, cfg)` signature will fail to compile after `crank update`.

**Mitigation:** `crank update` currently only bumps the version stamp — it does NOT re-render templates. Existing projects keep their old `worker.go` and `cmd/server/main.go`. The new signature only applies to new projects (`crank init`) and feature additions (`crank add temporal` to a project that doesn't have it yet).

**If template re-rendering is added later:** The `NewWorker` signature change is a compile error, which is the safest kind of breakage — the developer sees it immediately and the fix is mechanical (add `nil` or `acts`).

### Risk 2: `L(ctx)` removal breakage

**Impact:** Any generated project that calls `logging.L(ctx)` in custom code will fail to compile.

**Mitigation:** `L(ctx)` is only called in generated code (middleware, error handler). Custom user code that calls `L(ctx)` is rare. The compile error is immediate and the fix is `slog.InfoContext(ctx, ...)`.

**Alternative:** Keep `L(ctx)` as a deprecated wrapper that returns `slog.Default()` — but this defeats the purpose. Better to break cleanly.

### Risk 3: Qdrant gRPC → HTTP migration

**Impact:** The refactored Qdrant client uses HTTP (resty) instead of the gRPC client (`github.com/qdrant/go-client`). This changes the dependency and the connection model.

**Mitigation:** Keep the old gRPC client template as an alternative. Add a `--qdrant-transport` flag or a `qdrant-grpc` feature. Or, simpler: the platform client pattern is additive — the old `internal/adapters/persistence/qdrant/client.go` can coexist with `internal/adapters/platform/qdrant.go`.

**Decision:** For Phase 4, keep the old qdrant client as-is and add the platform HTTP client alongside it. The developer chooses which to use. The platform client is the recommended path for new projects.

### Risk 4: Audit feature migration numbering

**Impact:** The audit migration uses `000003_` prefix. If a project already has a migration with that prefix (e.g., from `crank make migration`), there will be a conflict.

**Mitigation:** Use a timestamp-based prefix instead of a fixed number, matching the pattern used by `crank make migration`. The `000003_` prefix is only for the initial generation — `crank add audit` should use a timestamp.

### Risk 5: Platform feature without any services

**Impact:** `crank init myapp --features=base,platform` generates the platform client helper but no typed clients. This is valid but unusual.

**Mitigation:** The `platform` feature is a base infrastructure feature — it's always pulled in when qdrant or redis are enabled (add `platform` to their `Requirements()`). Standalone use is fine but produces only the `client.go` and `types.go` files.

---

## Appendix A: File Summary

### Files Modified

| File | Phase |
|------|-------|
| `internal/bootstrap/features/base/templates/pkg_logging_logger.go.tmpl` | 1 |
| `internal/bootstrap/features/base/templates/internal_adapters_http_web_middleware_logging.go.tmpl` | 1 |
| `internal/bootstrap/features/base/templates/cmd_server_main.go.tmpl` | 1, 3, 4, 5 |
| `internal/bootstrap/features/temporal/templates/cmd_worker_main.go.tmpl` | 1, 3 |
| `internal/bootstrap/features/base/templates/internal_adapters_uow_in_memory_uow.go.tmpl` | 2 |
| `internal/bootstrap/features/temporal/templates/internal_adapters_temporal_worker.go.tmpl` | 3 |
| `internal/bootstrap/features/temporal/feature.go` | 3 |
| `internal/bootstrap/scaffold/wire_temporal.go` | 3 |
| `internal/bootstrap/features/qdrant/feature.go` | 4 |
| `internal/bootstrap/features/redis/feature.go` | 4 |
| `internal/bootstrap/config_inject.go` | 5 |
| `cmd/crank/main.go` | 4, 5 |
| `internal/bootstrap/integration_test.go` | all |
| `e2e/e2e_test.go` | all |

### Files Created

| File | Phase |
|------|-------|
| `internal/bootstrap/features/temporal/templates/internal_adapters_temporal_activity_activities.go.tmpl` | 3 |
| `internal/bootstrap/features/temporal/templates/Dockerfile.worker.tmpl` | 3 |
| `internal/bootstrap/features/platform/feature.go` | 4 |
| `internal/bootstrap/features/platform/templates/internal_adapters_platform_client.go.tmpl` | 4 |
| `internal/bootstrap/features/platform/templates/internal_ports_platform_types.go.tmpl` | 4 |
| `internal/bootstrap/features/qdrant/templates/internal_ports_platform_qdrant.go.tmpl` | 4 |
| `internal/bootstrap/features/qdrant/templates/internal_adapters_platform_qdrant.go.tmpl` | 4 |
| `internal/bootstrap/features/redis/templates/internal_adapters_platform_redis.go.tmpl` | 4 |
| `internal/bootstrap/features/audit/feature.go` | 5 |
| `internal/bootstrap/features/audit/templates/internal_domain_audit_event.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_domain_audit_repository.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_ports_audit.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_adapters_persistence_bun_audit_repository.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_adapters_persistence_gorm_audit_repository.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_adapters_audit_logger.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_application_audit_query_handler.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/internal_adapters_http_web_audit_handler.go.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/migrations_000003_add_audit_events.up.sql.tmpl` | 5 |
| `internal/bootstrap/features/audit/templates/migrations_000003_add_audit_events.down.sql.tmpl` | 5 |

### Files Removed

| File | Phase | Reason |
|------|-------|--------|
| `internal/bootstrap/features/qdrant/templates/internal_adapters_persistence_qdrant_client.go.tmpl` | 4 | Replaced by platform HTTP client |

## Appendix B: Validation Commands

After each phase, run:

```bash
# Unit + integration tests
go test ./internal/... -v -count=1

# Format check
go vet ./...

# E2E tests (slow — builds real binary, runs go get, compiles generated projects)
go test -tags e2e ./e2e/... -v -count=1 -timeout 600s

# Specific phase tests
go test ./internal/... -run TestBase_Logging -v          # Phase 1
go test ./internal/... -run TestBase_UoW -v              # Phase 2
go test ./internal/... -run TestTemporal_TwoProcess -v   # Phase 3
go test ./internal/... -run TestPlatform -v              # Phase 4
go test ./internal/... -run TestAudit -v                 # Phase 5
```
