---
title: Tool Wrappers Reference
type: reference
---

# Tool Wrappers & Development Commands

`crank` acts as a unified developer toolchain wrapper. Instead of requiring developers to install, manage, and memorize disparate CLI flags for `go build`, `go test`, `air`, `golang-migrate`, `swag`, `gofmt`, and `govet`, Crank exposes them all under consistent `crank` subcommands.

---

## Command Overview

| Subcommand | Wrapped Tool | Description |
| --- | --- | --- |
| `crank run` | `go run` | Compiles and executes the project's entrypoint (`cmd/server/main.go`). |
| `crank dev` | `air` | Starts development server with live reload / hot-reloading on file changes. |
| `crank build` | `go build` | Compiles a production binary into `bin/server`. |
| `crank test` | `go test` | Executes unit and integration tests with optional flags (`-v`, `--race`). |
| `crank doctor` | *In-Process Health Checks* | Validates Go environment, database connectivity, and required CLI tools. |
| `crank migrate` | `golang-migrate` | Runs database DDL migrations (`up`, `down`, `version`, `force`). |
| `crank make swag` | `swaggo/swag` | Generates Swagger / OpenAPI documentation into `docs/swagger/`. |
| `crank gofmt` | `gofmt` | Formats all `.go` files across the project directory recursively. |
| `crank vet` | `go vet` | Runs Go static analysis checks to catch code smells and potential bugs. |
| `crank tidy` | `go mod tidy` | Cleans up unused module dependencies in `go.mod` and `go.sum`. |

---

## Detailed Command Usage

### `crank run`
Runs the main service entrypoint:
```bash
crank run
crank run --project=./services/bookstore
```

---

### `crank dev`
Starts the service under live-reloading mode via `air`:
```bash
crank dev
```

---

### `crank build`
Compiles the application binary with sensible build flags:
```bash
crank build
# Output: bin/server
```

---

### `crank test`
Executes test suites and forwards standard `go test` arguments:
```bash
crank test
crank test -v
crank test ./internal/application/...
```

---

### `crank doctor`
Performs comprehensive health checks on your project and development environment:
```bash
crank doctor
```

**Checks performed**:
- Go version compatibility (requires Go 1.26+).
- Installed CLI binaries (`air`, `golang-migrate`, `swag`).
- Connection check to database specified in `.env`.
- Project manifest integrity (`.crank.yaml`).

---

### `crank migrate`
Executes SQL migrations using `golang-migrate`:
```bash
crank migrate up
crank migrate down 1
crank migrate version
```

---

### `crank make swag`
Parses Echo handler annotations and generates OpenAPI / Swagger 2.0 specifications:
```bash
crank make swag
# Generates: docs/swagger/swagger.json and docs/swagger/swagger.yaml
```

---

### `crank gofmt`, `crank vet`, `crank tidy`
Project maintenance subcommands:
```bash
crank gofmt
crank vet
crank tidy
```
