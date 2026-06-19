---
title: Generated Project Structure
---

# Generated project structure

A generated project follows a layered, Domain-Driven layout. The exact files depend on selected features, but the shape stays consistent.

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
│   │   ├── persistence/
│   │   └── uow/
│   ├── application/
│   ├── config/
│   ├── domain/
│   ├── model/
│   ├── ports/
│   └── validator/
├── pkg/
│   └── logging/
├── migrations/
├── docs/
├── .agents/
│   └── skills/
│       └── crank-project/
│           └── SKILL.md
├── .crank.yaml
├── AGENTS.md
├── .env.example
├── .gitignore
├── .air.toml
├── Dockerfile
├── Makefile
└── go.mod
```

Some directories are feature-specific. For example, `migrations/` appears when GORM or Bun is enabled, and Temporal adapter folders appear when `temporal` is enabled.

## `cmd/server`

The application entry point and composition root.

Responsibilities:

- load configuration
- initialize logging
- create infrastructure clients and adapters
- create application command/query handlers
- mount HTTP routes
- start the server
- handle graceful shutdown

Generated services are wired here when needed.

## `configs`

Configuration files that are safe to commit.

```text
configs/config.yaml
```

This file documents defaults for the service. Secrets should not live here; use `.env` or real environment variables instead.

## `.crank.yaml`

Project manifest used by `crank`.

It tracks:

- project name
- Go module path
- enabled features
- generator version metadata

Do not delete it. Commands like `crank add`, `crank make`, `crank migrate`, and `crank doctor` use it to understand the project.

AI coding agents should also use `.crank.yaml` as the primary signal that a repository is Crank-managed and as the source of truth for enabled features.

## Agent guidance files

Generated projects include lightweight guidance for AI agents:

```text
AGENTS.md
.agents/skills/crank-project/SKILL.md
```

`AGENTS.md` is a root-level signpost for agents. The Zed skill under `.agents/skills/crank-project/` gives Zed agents detailed Crank workflow instructions.

Both files tell agents to use the system-installed `crank` command, not a local `./crank` binary, and to pass `--project` when running from outside the project root.

These files are generated with skip-if-exists behavior, so team customizations are preserved.

See [AI agent support](./ai-agents.md) for details.

## `internal/domain`

Pure domain code.

Generated resources live under their own package:

```text
internal/domain/book/
├── book.go
├── book_id.go
├── events.go
├── errors.go
└── repository.go
```

Domain packages should not import HTTP frameworks, databases, environment loaders, or other infrastructure packages.

Typical contents:

- aggregate types
- value objects
- domain events
- domain errors
- repository interfaces / ports

## `internal/application`

Use cases for each resource.

```text
internal/application/book/
├── commands.go
├── command_handler.go
├── queries.go
└── query_handler.go
```

Application code coordinates domain objects and repository ports. It should remain independent of concrete HTTP/database implementations.

## `internal/adapters`

Infrastructure implementations.

### HTTP adapter

```text
internal/adapters/http/web/
├── server.go
├── routes.go
├── user_handler.go
└── middleware/
```

HTTP handlers translate requests into application commands/queries and translate results into JSON responses.

Generated handlers are wired into `routes.go` using marker comments. This keeps `crank make handler` and `crank make scaffold` idempotent.

### Persistence adapters

```text
internal/adapters/persistence/
├── memory/
├── gorm/
└── bun/
```

Which adapters exist depends on selected features:

| Feature | Persistence adapter |
| --- | --- |
| no ORM | in-memory |
| `gorm` | GORM + in-memory |
| `bun` | Bun + in-memory |

### Event bus and unit of work

Base projects include in-memory implementations for cross-cutting ports such as event bus and unit of work.

The `outbox` feature adds PostgreSQL-backed event persistence for reliable publication.

## `internal/config`

Configuration loading and typed config structs.

This package uses Viper for YAML and environment resolution. Feature config sections are consolidated here rather than each feature defining its own config package.

When you add a feature later, `crank add` injects new config fields and defaults using marker comments.

## `internal/validator`

Request validation setup based on `go-playground/validator`.

Handlers can bind request bodies with Echo. The custom binder validates automatically after `Bind()` succeeds.

## `pkg/logging`

Reusable logging helpers built on `log/slog`, including redaction support.

## `migrations`

SQL migration files for golang-migrate.

```text
migrations/
├── 000001_init.up.sql
├── 000001_init.down.sql
└── 20260619123045_create_books.up.sql
```

Use:

```bash
crank migrate up
crank migrate down --steps 1
crank make migration add_index_to_books
```

## Generated `Makefile`

Generated projects intentionally keep `Makefile` responsibilities narrow. Common development tasks are native `crank` commands. The Makefile is for project-specific extras such as `clean` or custom team tasks.

If a command is not native to `crank`, `crank` can delegate to a matching Makefile target:

```bash
crank clean
crank seed-demo-data environment=local
```

## `docs` in generated projects

Generated applications use `docs/` for Swagger output produced by:

```bash
crank swag
```

This is separate from the `crank` repository's own documentation site.
