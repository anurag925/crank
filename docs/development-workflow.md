---
title: Development Workflow
---

# Development workflow

`crank` is intended to be the primary interface for routine work in generated projects.

Generated projects also include `AGENTS.md` and `.agents/skills/crank-project/SKILL.md` so AI coding agents can detect `.crank.yaml`, read enabled features, and use the same system-installed `crank` command you use. See [AI agent support](./ai-agents.md).

## Run locally

From inside a generated project:

```bash
cp .env.example .env
crank run
```

From another directory:

```bash
crank run --project ./myapp
```

## Live reload

```bash
crank dev
```

This runs `air` using the generated `.air.toml`. If `air` is not installed, `crank` attempts to install it automatically.

## Build

```bash
crank build
```

The build tool compiles the server entry point under `cmd/server` and writes a binary to `bin/`.

## Test

```bash
crank test
crank test -v
crank test -count=1
```

Extra flags are forwarded to `go test`.

## Format and vet

```bash
crank gofmt
crank vet
```

Use these before opening a pull request.

## Keep dependencies tidy

```bash
crank tidy
```

This runs `go mod tidy` in the target project.

## Migrations

ORM-backed projects use golang-migrate.

Create a migration:

```bash
crank make migration add_status_to_orders
```

Apply migrations:

```bash
crank migrate up
```

Roll back one migration:

```bash
crank migrate down --steps 1
```

Use an explicit database URL:

```bash
crank migrate up --database-url postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable
```

## Swagger/OpenAPI

Generate Swagger docs:

```bash
crank swag
```

Start the server and open:

```text
http://localhost:8080/swagger/index.html
```

`crank swag` runs `swag init` against the server entry point and writes generated docs into the generated application's `docs/` directory.

## Health checks

Run:

```bash
crank doctor
```

This catches common project drift:

- invalid manifest
- mismatched module path
- handler files that are not registered
- application services not wired in the composition root
- migration timestamp problems

Use fail-fast mode when debugging CI failures:

```bash
crank doctor --fail-fast
```

## Makefile delegation

Generated projects include a small `Makefile` for project-specific targets. Native `crank` commands are not duplicated there.

If you add this target:

```makefile
seed-demo:
	go run ./cmd/seed-demo
```

You can run it as either:

```bash
make seed-demo
crank seed-demo
```

Native commands always take precedence, so `crank build` runs the native `crank` build wrapper even if a Makefile target named `build` exists.

## Suggested local loop

```bash
crank dev
# edit code in another terminal
crank gofmt
crank test
crank vet
crank doctor
```

For schema changes:

```bash
crank make migration add_field_to_table
crank migrate up
crank test
```

For new resources:

```bash
crank make scaffold Resource field:type --tests
crank migrate up
crank test
crank doctor
```
