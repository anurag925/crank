# Agent Guide — crank

## Project Overview

A modular CLI tool that scaffolds production-ready Go backend services and wraps common development tools as subcommands. Given a project name and a list of features, it generates a clean, layered Go service with sensible defaults and optional modules (auth, postgres, redis, mongodb). All day-to-day development tasks (build, test, migrate, swag, etc.) are accessible through the `crank` CLI so developers never need to leave it.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.21 |
| CLI Framework | Cobra (`github.com/spf13/cobra`) |
| Config (generated) | Viper + YAML + .env |
| HTTP (generated) | Echo v4 |
| Docs (generated) | Swagger via [swaggo/swag](https://github.com/swaggo/swag) + [echo-swagger](https://github.com/swaggo/echo-swagger) |
| ORM (generated) | Bun |
| Logging (generated) | `log/slog` (stdlib) |
| Migrations (generated) | golang-migrate |
| Templating | `text/template` + `embed.FS` |
| YAML parsing | `gopkg.in/yaml.v3` |

## Build & Run

```bash
# Build the CLI binary
go build -o crank ./cmd/bootstrap

# Scaffold a new project (tools are checked/installed automatically)
./crank init myapp --features=base,auth,postgres

# List available features
./crank list

# List available tool subcommands
./crank tools

# Add a feature to an existing project
./crank add redis --project=./myapp

# Generate a migration
./crank make migration create_orders --project=./myapp

# Run migrations (with --project or from inside the project directory)
./crank migrate up --project=./myapp
cd myapp && ./crank migrate up

# Run a generated project
./crank run --project=./myapp
cd myapp && ./crank run

# Other tool subcommands (all accept --project or use current directory)
./crank build --project=./myapp
./crank test -v --project=./myapp
./crank swag --project=./myapp
./crank dev --project=./myapp
./crank gofmt --project=./myapp
./crank vet --project=./myapp
./crank tidy --project=./myapp
```

## Architecture

```
crank/
├── cmd/bootstrap/main.go                  # CLI entry point (Cobra root command)
├── internal/
│   ├── bootstrap/
│   │   ├── feature.go                     # Feature interface, Registry, template rendering
│   │   ├── tool.go                        # Tool interface, ToolInvocation, ToolRegistry
│   │   ├── tool_registry.go               # GlobalToolRegistry singleton
│   │   ├── generator.go                   # Generate() and Add() orchestration logic
│   │   ├── context.go                     # Template context (ProjectName, ModulePath, Has())
│   │   ├── manifest.go                    # .bootstrap.yaml encode/decode
│   │   ├── registry.go                    # GlobalRegistry singleton
│   │   ├── result.go                      # Result.FeaturesUsed() helper
│   │   ├── gomod.go                       # GoGet, Tidy helpers (go get + go mod tidy)
│   │   ├── commands/                      # One Cobra command per CLI subcommand
│   │   │   ├── init.go                    # `crank init` (includes tool checking)
│   │   │   ├── add.go                     # `crank add`
│   │   │   ├── list.go                    # `crank list`
│   │   │   ├── make.go                    # `crank make`
│   │   │   └── tools.go                   # Generic tool command factory + `crank tools`
│   │   ├── tools/                         # Tool wrappers (one package per external CLI)
│   │   │   ├── install.go                 # Shared InstallGoTool helper
│   │   │   ├── migrate/                   # `crank migrate` → golang-migrate
│   │   │   ├── swag/                      # `crank swag` → swaggo/swag
│   │   │   ├── build/                     # `crank build` → go build
│   │   │   ├── run/                       # `crank run` → go run
│   │   │   ├── dev/                       # `crank dev` → air
│   │   │   ├── test/                      # `crank test` → go test
│   │   │   ├── gofmt/                     # `crank gofmt` → gofmt
│   │   │   ├── vet/                       # `crank vet` → go vet
│   │   │   └── tidy/                      # `crank tidy` → go mod tidy
│   │   └── features/                      # One package per installable module
│   │       ├── base/                      # Echo + Viper + slog + dev tooling
│   │       ├── auth/                      # JWT middleware + auth handlers
│   │       ├── crypto/                    # AES-256-GCM encrypt/decrypt helpers
│   │       ├── postgres/                  # Bun ORM + migrations
│   │       ├── redis/                     # Redis client (placeholder)
│   │       └── mongodb/                   # MongoDB client (placeholder)
│   └── utils/
│       ├── fileutil.go                    # EnsureDir, WriteFile, PathExists, etc.
│       └── exec.go                        # FindBinary, RunExternal, ShellJoin
├── test/                                  # Integration test artifacts (git-ignored)
├── SPEC.md                                # Full product specification
└── go.mod
```

## Key Abstractions

### Feature Interface (`internal/bootstrap/feature.go`)

Every installable module implements this:

```go
type Feature interface {
    Name() string
    Description() string
    Files() []FileMapping
    Templates() embed.FS
    Dependencies() []string
}
```

- `Name()` — short identifier used in `--features` lists (e.g. `"base"`, `"auth"`, `"postgres"`)
- `Description()` — shown by `crank list`
- `Dependencies()` — Go module paths fetched via `go get` after scaffolding
- `Files()` — template-to-output path mappings
- `Templates()` — the `embed.FS` containing `.tmpl` files

### Tool Interface (`internal/bootstrap/tool.go`)

Every external CLI wrapper implements this:

```go
type Tool interface {
    Name() string                        // subcommand name (e.g. "migrate", "swag")
    Description() string                 // short help text
    LongDescription() string             // detailed help with examples
    BinaryName() string                  // executable on PATH
    InstallCmd() string                  // human-readable install instruction
    RequiresFeatures() []string          // features that must be enabled
    AddFlags(cmd *cobra.Command)         // register custom CLI flags
    Prepare(projectDir string, cmd *cobra.Command) (*ToolInvocation, error)
    Install() error                      // auto-install the tool
}
```

- `Name()` — the subcommand name (e.g. `"migrate"` → `crank migrate`)
- `BinaryName()` — the executable to look up on PATH (e.g. `"migrate"`, `"swag"`)
- `InstallCmd()` — shown when the tool is missing; also used by `crank init` for auto-install
- `RequiresFeatures()` — e.g. `migrate` requires `"postgres"`; empty means always available
- `AddFlags()` — lets tools register custom flags (e.g. `--database-url`, `--steps`)
- `Prepare()` — builds the `ToolInvocation` (args, working dir, stdin, env)
- `Install()` — downloads/installs the tool (usually via `go install`)

### ToolInvocation (`internal/bootstrap/tool.go`)

```go
type ToolInvocation struct {
    Binary string   // full path to the binary
    Args   []string // arguments (without binary name)
    Dir    string   // working directory
    Env    []string // additional KEY=VALUE env vars
    Stdin  bool     // whether to pass os.Stdin through
}
```

### FileMapping (`internal/bootstrap/feature.go`)

```go
type FileMapping struct {
    TemplatePath string  // path inside the embedded FS
    OutputPath   string  // destination relative to project root
    SkipIfExists bool    // leave existing file untouched
}
```

### Context (`internal/bootstrap/context.go`)

Passed to every template during rendering:

- `ProjectName` — user-supplied name
- `ModulePath` — Go module path for `go.mod` / imports
- `PackageName` — last segment of module path
- `Features` — list of enabled feature names
- `Has(name string) bool` — check if a feature is active (used in templates with `{{if .Has "postgres"}}...{{end}}`)
- `Require(names ...string) error` — fail if a feature is missing

### Registry (`internal/bootstrap/feature.go` + `registry.go`)

- `GlobalRegistry` — process-wide singleton in `registry.go`
- Features self-register in `init()` via `bootstrap.GlobalRegistry.MustRegister(feature{})`
- `cmd/bootstrap/main.go` imports feature packages with `_` (blank import) to trigger registration

### ToolRegistry (`internal/bootstrap/tool.go` + `tool_registry.go`)

- `GlobalToolRegistry` — process-wide singleton in `tool_registry.go`
- Tools self-register in `init()` via `bootstrap.GlobalToolRegistry.MustRegister(tool{})`
- `cmd/bootstrap/main.go` imports tool packages with `_` (blank import) to trigger registration
- `ForFeatures(features)` — returns tools whose requirements are satisfied by the given feature set
- `ForFeature(feature)` — returns tools that specifically require one feature

### Generator (`internal/bootstrap/generator.go`)

- `Generate(reg, opts)` — creates a new project from scratch; `base` is always first; returns `Result.Dependencies` for the caller to run `go get`
- `Add(reg, projectDir, featureName)` — adds a feature to an existing project; re-renders all features; updates `.bootstrap.yaml` manifest; returns `Result.Dependencies` with only the new feature's deps
- `GoGet(projectDir, deps)` — runs `go get <deps...>` then `go mod tidy` in the project directory
- `Tidy(projectDir)` — runs `go mod tidy` in the project directory

## Conventions & Patterns

### Adding a New Feature

1. Create `internal/bootstrap/features/<name>/feature.go`
2. Create `internal/bootstrap/features/<name>/templates/` with `.tmpl` files
3. Implement the `Feature` interface
4. Self-register in `init()`:
   ```go
   func init() {
       bootstrap.GlobalRegistry.MustRegister(feature{})
   }
   ```
5. Add a blank import in `cmd/bootstrap/main.go`:
   ```go
   _ "github.com/anurag925/crank/internal/bootstrap/features/<name>"
   ```

### Adding a New Tool Wrapper

1. Create `internal/bootstrap/tools/<name>/tool.go`
2. Implement the `Tool` interface
3. Self-register in `init()`:
   ```go
   func init() {
       bootstrap.GlobalToolRegistry.MustRegister(&tool{})
   }
   ```
4. If the tool needs custom flags, implement `AddFlags(cmd *cobra.Command)` and read them in `Prepare()` via `cmd.Flags()`
5. Add a blank import in `cmd/bootstrap/main.go`:
   ```go
   _ "github.com/anurag925/crank/internal/bootstrap/tools/<name>"
   ```

The command factory in `commands/tools.go` handles everything else: `--project` flag, binary lookup, auto-install on missing, and execution.

### Adding Custom Validators

The generated project includes a ready-to-use [`go-playground/validator`](https://github.com/go-playground/validator) setup in `internal/validator/`. To add a custom validator:

1. Open `internal/validator/validator.go`
2. Inside `Init()`, register your validator **before** the closing comment block:
   ```go
   validate.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
       return strings.TrimSpace(fl.Field().String()) != ""
   })
   ```
3. Use the tag in your structs: `validate:"required,notblank"`
4. Add a human-readable message for the tag in `internal/validator/errors.go` → `humanMessage()`

### Adding Validation Tags to Structs

Add `validate` struct tags to any request/model struct. The custom Echo binder runs validation automatically after `c.Bind()`:

```go
type CreateOrderInput struct {
    ProductID string  `json:"product_id" validate:"required,uuid"`
    Quantity  int     `json:"quantity"  validate:"required,gt=0,lte=999"`
    Notes     string  `json:"notes"     validate:"max=500"`
}
```

Handlers only need `c.Bind(&input)` — if it returns `nil`, the input is valid.

### Template Conventions

- Template files use `.tmpl` extension
- Naming convention: `<path_with_underscores>.<ext>.tmpl` (e.g., `internal_handler_user.go.tmpl`)
- Use `{{.ProjectName}}`, `{{.ModulePath}}`, `{{.PackageName}}` for project-specific values
- Use `{{if .Has "feature"}}...{{end}}` for conditional sections
- Templates use `text/template` with `missingkey=error` option

### Code Style

- Standard Go formatting (`gofmt`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- No external linters configured; follow idiomatic Go
- `internal/` package convention — nothing is importable outside the module
- Utility functions live in `internal/utils/`

### Configuration Strategy (Generated Projects)

Config files live in a top-level `configs/` directory, following the
[golang-standards/project-layout](https://github.com/golang-standards/project-layout#configs) convention.

- `configs/config.yaml` — safe defaults, committed to source
- `.env.example` — template for local env overrides
- `.env` — secrets and env-specific values, git-ignored
- Viper searches `./configs` first, then `.` as fallback
- Viper priority: env vars > .env > configs/config.yaml

## What Lives Where

| Concern | Location |
|---------|----------|
| CLI entry point & root command | `cmd/bootstrap/main.go` |
| Subcommand definitions | `internal/bootstrap/commands/*.go` |
| Feature interface & registry | `internal/bootstrap/feature.go` |
| Tool interface & registry | `internal/bootstrap/tool.go` + `tool_registry.go` |
| Tool command factory | `internal/bootstrap/commands/tools.go` |
| Makefile delegation fallback | `internal/bootstrap/commands/makedelegate.go` |
| Tool implementations | `internal/bootstrap/tools/<name>/tool.go` |
| Tool install helper | `internal/bootstrap/tools/install.go` |
| Exec utilities | `internal/utils/exec.go` |
| Global feature registry | `internal/bootstrap/registry.go` |
| Global tool registry | `internal/bootstrap/tool_registry.go` |
| Project generation logic | `internal/bootstrap/generator.go` |
| Template context | `internal/bootstrap/context.go` |
| Manifest I/O (.bootstrap.yaml) | `internal/bootstrap/manifest.go` |
| Result helpers | `internal/bootstrap/result.go` |
| Filesystem utilities | `internal/utils/fileutil.go` |
| Feature implementations | `internal/bootstrap/features/<name>/feature.go` |
| Feature templates | `internal/bootstrap/features/<name>/templates/*.tmpl` |
| Generated test projects | `test/` (git-ignored) |

## Available Tool Subcommands

| Subcommand | Wraps | Binary | Requires |
|------------|-------|--------|----------|
| `crank migrate` | golang-migrate | `migrate` | `postgres` feature |
| `crank swag` | swaggo/swag | `swag` | — |
| `crank build` | `go build` | `go` | — |
| `crank run` | `go run ./cmd/server` | `go` | — |
| `crank dev` | air (live reload) | `air` | — |
| `crank test` | `go test ./...` | `go` | — |
| `crank gofmt` | `gofmt -s -w .` | `gofmt` | — |
| `crank vet` | `go vet ./...` | `go` | — |
| `crank tidy` | `go mod tidy` | `go` | — |

All tools accept `--project <dir>`. If `--project` is not specified, the current directory is used as the project root.

## Makefile Delegation

The generated project ships **both** a `Makefile` and is usable through the `crank`
CLI, but they have **non-overlapping** responsibilities to avoid confusion:

- `crank` is the single source of truth for common development tasks (build, run,
  dev, test, fmt, vet, tidy, swag, migrate). These are intentionally **not**
  duplicated as Makefile targets.
- The generated `Makefile` holds only targets that `crank` does not provide
  natively (e.g. `clean`) and is the place for project-specific custom targets.

To bridge the two, `crank` transparently delegates unknown subcommands to the
project's `Makefile`. When you run `crank <name>` and `<name>` is **not** a native
crank command, crank looks up `<name>` as a target in the target project's
`Makefile` (respecting `--project`) and runs `make <name>` for you. Any extra
arguments (e.g. `name=foo`) are forwarded to make verbatim.

```bash
# `clean` is not a native crank command, but the Makefile defines it, so this
# runs `make clean` in the project directory.
crank clean --project ./myapp
crank greet name=anurag --project ./myapp   # → make greet name=anurag (custom target)
```

Precedence rules:

- **Native crank commands always win.** The fallback is only consulted for names
  cobra does not already recognize, so `crank build` always runs the native build
  tool. (The Makefile no longer defines a `build` target anyway.)
- If the name matches neither a crank command nor a Makefile target (or there is
  no `Makefile`), crank reports the usual `unknown command` error.

This is implemented in `internal/bootstrap/commands/makedelegate.go` and hooked
into `cmd/bootstrap/main.go` before `cobra`'s `Execute()`. It also provides a
natural extension path: project-specific behavior can be expressed as Makefile
targets without modifying crank itself.

## Testing

There are currently no Go unit tests (`*_test.go`) in the project. The `test/` directory contains a generated sample project (`test/myapp/`) used for manual/integration verification. To validate changes:

1. Build the CLI: `go build -o crank ./cmd/bootstrap`
2. Generate a test project: `./crank init testapp --features=base,auth,postgres`
3. Verify the output structure and generated files

## Dependencies

Direct dependencies (from `go.mod`):
- `github.com/spf13/cobra v1.8.0` — CLI framework
- `gopkg.in/yaml.v3 v3.0.1` — YAML manifest parsing

Generated projects pull in their own dependencies (Echo, Bun, Viper, etc.) via their `go.mod` templates.
