---
title: Commands
---

# Commands

`crank` has three categories of commands:

1. Project lifecycle commands.
2. Code generators.
3. Tool wrappers for day-to-day development.

## Project lifecycle

### `crank init`

Scaffold a new project.

```bash
crank init <project> [flags]
```

Examples:

```bash
crank init myapp --features=base,auth
crank init myapp --features=base,auth --use-bun
crank init myapp --module=github.com/acme/myapp --features=base,redis
crank init myapp --target ./services
```

Flags:

| Flag | Description |
| --- | --- |
| `--features` | Comma-separated feature list. Defaults to `base`; GORM is added automatically unless an ORM is already selected or `--use-bun` is passed. |
| `--module` | Go module path. Defaults to the project name. |
| `--target` | Parent directory in which to create the project. Defaults to `.`. |
| `--force` | Overwrite an existing non-empty target directory. |
| `--use-bun` | Use Bun instead of the default GORM ORM. |

When run in a terminal without explicit flags, `crank init` can prompt interactively for project name, module path, ORM, features, target directory, and overwrite behavior.

### `crank add`

Install one feature into an existing generated project.

```bash
crank add <feature> --project ./myapp
```

Examples:

```bash
crank add redis --project ./myapp
crank add temporal --project ./myapp
crank add otel --project ./myapp
```

The target project must contain `.crank.yaml`. `crank add` renders the feature's files, injects config sections into existing config files, updates the manifest, and runs `go get` / `go mod tidy` for new dependencies.

### `crank list`

List all installable features:

```bash
crank list
```

Use this before choosing the `--features` list for a new project.

### `crank tools`

List all tool wrappers:

```bash
crank tools
```

Each listed tool is available as:

```bash
crank <tool> [args] --project <dir>
```

## Code generation

### `crank make`

Generate code inside an existing project.

```bash
crank make <kind> <name> [field:type ...] [flags]
```

Supported kinds:

| Kind | Generates |
| --- | --- |
| `model` | Domain aggregate, value object, events, errors, repository port, and migration when an ORM is enabled. |
| `repository` | Persistence adapter. Uses GORM, Bun, or in-memory depending on enabled features. |
| `service` | Application command/query handlers. |
| `handler` | Echo HTTP handler and route wiring. Also generates dependencies unless `--only` is used. |
| `scaffold` | Full stack: domain, application, persistence, HTTP handler, route wiring, and migrations. |
| `workflow` | Temporal workflow. Requires the `temporal` feature. |
| `activity` | Temporal activity. Requires the `temporal` feature. |
| `migration` | Blank SQL migration pair. |

Flags:

| Flag | Description |
| --- | --- |
| `--project` | Target project directory. Defaults to `.`. |
| `--only` | Generate only the requested kind, skipping dependencies. |
| `--force` | Overwrite the primary generated artifact if it exists. |
| `--skip-migration` | Do not generate a table migration even when GORM or Bun is enabled. |
| `--tests` | Generate `_test.go` files alongside generated layers. |

Examples:

```bash
crank make model Order customer:string total:float
crank make handler Product title:string price:float
crank make handler Product --only
crank make scaffold Invoice number:string amount:float --tests
crank make workflow OrderFulfillment order_id:uuid
crank make activity ChargeCard amount:float --tests
crank make repository Ticket
crank make migration create_orders
```

## Tool wrappers

Tool commands accept `--project`. If omitted, the current directory is used.

| Command | What it does | External binary |
| --- | --- | --- |
| `crank build` | Compile `./cmd/server` into `bin/<project>`. | `go` |
| `crank run` | Run the server with `go run ./cmd/server`. | `go` |
| `crank dev` | Run with live reload. | `air` |
| `crank test` | Run `go test ./...`; extra flags are forwarded. | `go` |
| `crank gofmt` | Run `gofmt -s -w .`; extra args can be used for alternate gofmt behavior. | `gofmt` |
| `crank vet` | Run `go vet ./...`. | `go` |
| `crank tidy` | Run `go mod tidy`. | `go` |
| `crank swag` | Run `swag init` for Swagger/OpenAPI output. | `swag` |
| `crank migrate` | Run database migrations through golang-migrate. | `migrate` |
| `crank doctor` | Run project health checks. | none |

### `crank migrate`

`crank migrate` requires the `gorm` or `bun` feature.

```bash
crank migrate up
crank migrate down --steps 1
crank migrate --database-url postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable up
```

By default it runs `up`. The database URL is resolved in this order:

1. `--database-url`
2. `DATABASE_URL`
3. Values under `database:` in `configs/config.yaml`

### `crank doctor`

`crank doctor` checks high-signal project health:

- `.crank.yaml` parses and has required fields.
- `go.mod` module path matches `.crank.yaml`.
- Generated handlers are wired into HTTP routes.
- Generated services are wired into `cmd/server/main.go`.
- Migration files are uniquely timestamped and sorted.

Examples:

```bash
crank doctor
crank doctor --fail-fast
crank doctor --project ./myapp
```

## Makefile delegation

Native `crank` commands always win. If you run an unknown `crank` command and the target project has a matching `Makefile` target, `crank` delegates to `make`.

```bash
crank clean --project ./myapp
crank greet name=Anurag --project ./myapp
```

This lets project-specific tasks live in the generated `Makefile` without duplicating native commands like `build`, `test`, or `run`.
