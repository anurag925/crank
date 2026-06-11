# rev

A modular CLI that scaffolds production-ready Go backend services and wraps common
development tools as subcommands. Given a project name and a list of features, it
generates a clean, layered Go service with sensible defaults and a curated set of
optional modules. All day-to-day development tasks are accessible through `rev`.

## Features

| Feature   | Description |
| --------- | ----------- |
| `base`    | Echo v4 HTTP server, Viper configuration, structured `slog` logging, health probe, graceful shutdown, Makefile, Dockerfile, Air live-reload config |
| `auth`    | JWT issuance + validation, refresh tokens, bcrypt password hashing, Echo middleware for protected routes |
| `crypto`  | AES-256-GCM encryption/decryption helpers, config-driven secret, base64-url output |
| `postgres`| PostgreSQL via Bun ORM, golang-migrate migrations, repository pattern with `GetByEmail` lookup |
| `redis`   | Redis client (session storage / caching / rate limiting) |
| `mongodb` | MongoDB client (document storage / aggregation) |

## Install / Build

```bash
go build -o rev ./cmd/bootstrap
```

## Usage

### Project Lifecycle

```bash
# Scaffold a new project (tools are checked/installed automatically)
./rev init myapp --features=base,auth,postgres

# List available features
./rev list

# Add a feature to an existing project
./rev add redis --project=./myapp

# Generate a new migration
./rev make migration create_orders --project=./myapp
```

### Development Tools (all accept --project or use current directory)

```bash
# Run migrations
./rev migrate up --project=./myapp
cd myapp && ./rev migrate up                    # current directory

# Run the project
./rev run --project=./myapp
cd myapp && ./rev run

# Build the binary
./rev build --project=./myapp

# Run with live reload
./rev dev --project=./myapp

# Generate Swagger docs
./rev swag --project=./myapp

# Run tests
./rev test -v --project=./myapp

# Format code
./rev gofmt --project=./myapp

# Vet code
./rev vet --project=./myapp

# Tidy module dependencies
./rev tidy --project=./myapp

# List all available tool subcommands
./rev tools
```

### Auto-install

When you run `rev init`, the CLI checks that all tools required by the selected
features are installed. Missing tools are installed automatically via `go install`.
When you run a tool subcommand directly and the binary is missing, rev attempts
auto-install before giving up.

## Tool Subcommands

| Subcommand | Wraps | Binary | Requires |
|------------|-------|--------|----------|
| `rev migrate` | golang-migrate | `migrate` | `postgres` feature |
| `rev swag` | swaggo/swag | `swag` | — |
| `rev build` | `go build` | `go` | — |
| `rev run` | `go run ./cmd/server` | `go` | — |
| `rev dev` | air (live reload) | `air` | — |
| `rev test` | `go test ./...` | `go` | — |
| `rev gofmt` | `gofmt -s -w .` | `gofmt` | — |
| `rev vet` | `go vet ./...` | `go` | — |
| `rev tidy` | `go mod tidy` | `go` | — |

### Adding a New Tool Wrapper

1. Create `internal/bootstrap/tools/<name>/tool.go`
2. Implement the `bootstrap.Tool` interface
3. Self-register in `init()`: `bootstrap.GlobalToolRegistry.MustRegister(&tool{})`
4. Add a blank import in `cmd/bootstrap/main.go`

The command factory handles `--project`, binary lookup, auto-install, and execution.

## Architecture

```
rev/
├── cmd/bootstrap/main.go          # CLI entry point (cobra)
├── internal/
│   ├── bootstrap/                 # generation engine + tool wrapper system
│   │   ├── generator.go           # Generate / Add entry points
│   │   ├── feature.go             # Feature interface + registry
│   │   ├── tool.go                # Tool interface + ToolRegistry
│   │   ├── tool_registry.go       # GlobalToolRegistry singleton
│   │   ├── context.go             # template context
│   │   ├── manifest.go            # .bootstrap.yaml I/O
│   │   ├── registry.go            # process-wide feature registry
│   │   ├── result.go
│   │   ├── commands/              # one cobra command per subcommand
│   │   │   ├── init.go            # `rev init` (includes tool checking)
│   │   │   ├── add.go             # `rev add`
│   │   │   ├── list.go            # `rev list`
│   │   │   ├── make.go            # `rev make`
│   │   │   └── tools.go           # generic tool command factory
│   │   └── tools/                 # tool wrappers (one package per CLI)
│   │       ├── install.go         # shared InstallGoTool helper
│   │       ├── migrate/           # golang-migrate wrapper
│   │       ├── swag/              # swaggo/swag wrapper
│   │       ├── build/             # go build wrapper
│   │       ├── run/               # go run wrapper
│   │       ├── dev/               # air live-reload wrapper
│   │       ├── test/              # go test wrapper
│   │       ├── gofmt/             # gofmt wrapper
│   │       ├── vet/               # go vet wrapper
│   │       └── tidy/              # go mod tidy wrapper
│   ├── utils/                     # filesystem + exec helpers
│   │   ├── fileutil.go
│   │   └── exec.go
│   └── bootstrap/features/        # one package per installable module
│       ├── base/
│       ├── auth/
│       ├── postgres/
│       ├── redis/
│       └── mongodb/
└── go.mod
```

Each feature is a Go package with:

1. An `embed.FS` containing `templates/*.tmpl`
2. A `Files() []FileMapping` describing template → output path pairs
3. A self-registration in `init()` against `bootstrap.GlobalRegistry`

Each tool is a Go package with:

1. An implementation of the `bootstrap.Tool` interface
2. A self-registration in `init()` against `bootstrap.GlobalToolRegistry`

Templates are rendered with Go's `text/template`; the supplied `Context` exposes
the project name, module path and a `Has(name string) bool` helper so templates
can include feature-specific sections with `{{if .Has "postgres"}} ... {{end}}`.

## Configuration strategy

`configs/config.yaml` ships safe defaults that are committed to source. Secrets and
environment-specific values live in `.env` (git-ignored) and are loaded by Viper
as env vars. Environment variables always win. Config files follow the
[golang-standards/project-layout](https://github.com/golang-standards/project-layout#configs) `configs/` convention.

## Phase status

| Phase | Description | Status |
| ----- | ----------- | ------ |
| 1 | Core CLI scaffold + base feature | ✅ |
| 2 | Auth feature | ✅ |
| 3 | Postgres feature | ✅ |
| 4 | Project generation + dev tooling (Makefile, Air, Dockerfile) | ✅ |
| 5 | Redis module | ✅ (placeholder) |
| 6 | MongoDB module | ✅ (placeholder) |
| 7 | Pluggable tool wrapper system | ✅ |
