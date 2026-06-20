---
title: Base Feature
---

# Base feature

The `base` feature is the **foundation of every generated project**. It is always included and cannot be removed. It provides the core project structure, HTTP server, configuration, validation, logging, and development tooling.

## What it provides

| Layer | Files | Purpose |
|-------|-------|---------|
| 🚀 **Composition root** | `cmd/server/main.go` | Single entry point that wires all dependencies |
| ⚙️ **Configuration** | `internal/config/config.go` | Viper + godotenv + env struct-tag loading |
| 📝 **Shared model** | `internal/model/model.go` | API response types (`APIError`) |
| 🧠 **Domain** | `internal/domain/shared/`, `internal/domain/user/` | Aggregate root, typed ID, domain events, sentinel errors, repository port |
| ⚙️ **Application** | `internal/application/user/` | CQRS command/query handlers for the `User` aggregate |
| 🔌 **Ports** | `internal/ports/` | Cross-cutting interfaces (`EventBus`, `UnitOfWork`) |
| 🌐 **HTTP** | `internal/adapters/http/web/` | Echo server, routes, user handler, logging middleware |
| 💾 **In-memory adapters** | `internal/adapters/persistence/memory/`, `internal/adapters/eventbus/`, `internal/adapters/uow/` | In-memory implementations for testing and local dev |
| ✅ **Validator** | `internal/validator/` | `go-playground/validator` setup with a custom Echo binder |
| 📊 **Logging** | `pkg/logging/` | `log/slog` helpers with redaction support |
| 📄 **Project metadata** | `configs/config.yaml`, `.env.example`, `Makefile`, `.air.toml`, `Dockerfile`, `.gitignore`, `go.mod`, `.crank.yaml`, `AGENTS.md`, `.agents/skills/` | Standard project files |
| 📖 **Swagger** | `docs/docs.go` | Swagger/OpenAPI plumbing |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [Echo v4](https://github.com/labstack/echo) | HTTP router and web framework | [echo.labstack.com](https://echo.labstack.com) |
| [Viper](https://github.com/spf13/viper) | Configuration management (YAML + env) | [github.com/spf13/viper](https://github.com/spf13/viper) |
| [godotenv](https://github.com/joho/godotenv) | `.env` file loader | [github.com/joho/godotenv](https://github.com/joho/godotenv) |
| [caarlos0/env](https://github.com/caarlos0/env) | Struct-tag based env variable parsing | [github.com/caarlos0/env](https://github.com/caarlos0/env) |
| [go-playground/validator](https://github.com/go-playground/validator) | Request struct validation | [pkg.go.dev](https://pkg.go.dev/github.com/go-playground/validator/v10) |
| [log/slog](https://pkg.go.dev/log/slog) | Structured logging (stdlib) | [pkg.go.dev/log/slog](https://pkg.go.dev/log/slog) |
| [swaggo/swag](https://github.com/swaggo/swag) | Swagger/OpenAPI generation | [github.com/swaggo/swag](https://github.com/swaggo/swag) |
| [echo-swagger](https://github.com/swaggo/echo-swagger) | Echo Swagger middleware | [github.com/swaggo/echo-swagger](https://github.com/swaggo/echo-swagger) |
| [testify](https://github.com/stretchr/testify) | Test assertions and mocks | [github.com/stretchr/testify](https://github.com/stretchr/testify) |
| [air](https://github.com/air-verse/air) | Live-reload during development | [github.com/air-verse/air](https://github.com/air-verse/air) |

## Learning resources

- **Domain-Driven Design** — [dddcommunity.org](https://www.dddcommunity.org/)
- **CQRS pattern** — [martinfowler.com/bliki/CQRS.html](https://martinfowler.com/bliki/CQRS.html)
- **Go project layout conventions** — [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- **Echo guide** — [echo.labstack.com/guide](https://echo.labstack.com/guide)
- **Viper documentation** — [github.com/spf13/viper#getting-started](https://github.com/spf13/viper#getting-started)
