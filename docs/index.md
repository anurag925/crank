---
title: crank Documentation
---

# crank

`crank` is a CLI for creating and maintaining production-ready Go backend services. It scaffolds a layered service, installs selected features, and wraps day-to-day development tools as first-class `crank` subcommands.

Instead of stitching together routing, configuration, validation, persistence, migrations, Swagger, live reload, and project scripts by hand, you choose the features you need and let `crank` generate a working project with sensible defaults.

## What you can build

A generated `crank` project gives you:

- A Go 1.21 backend service using Echo v4.
- A Domain-Driven layout with domain, application, adapter, and composition-root layers.
- Configuration through `configs/config.yaml`, `.env`, and environment variables.
- Structured logging with `log/slog`.
- Request validation with `go-playground/validator`.
- Swagger/OpenAPI generation with `swaggo/swag`.
- Optional PostgreSQL persistence through GORM or Bun.
- Optional modules for auth, Redis, MongoDB, Qdrant, Temporal, OpenTelemetry, React views, crypto helpers, and transactional outbox support.
- One CLI surface for build, run, test, format, vet, migrations, Swagger, live reload, health checks, and code generation.

## Quick example

```bash
# Install crank
curl -fsSL https://raw.githubusercontent.com/anurag925/crank/main/install.sh | sh

# Create a new service with the default ORM, GORM
crank init bookstore --features=base,auth

# Start the service
cd bookstore
cp .env.example .env
crank run
```

The generated server listens on `http://localhost:8080` by default. The health endpoint is available at:

```bash
curl http://localhost:8080/health
```

## How the docs are organized

| Page | Use it when you want to... |
| --- | --- |
| [Installation](./installation.md) | Install `crank` or build it from source. |
| [Getting started](./getting-started.md) | Scaffold your first app and run it locally. |
| [Commands](./commands.md) | Learn the full CLI surface. |
| [Features](./features.md) | Pick the modules to include in a project. |
| [Generators](./generators.md) | Generate models, handlers, scaffolds, workflows, activities, and migrations. |
| [Generated project structure](./project-structure.md) | Understand where generated code lives. |
| [Configuration](./configuration.md) | Configure generated applications safely. |
| [Development workflow](./development-workflow.md) | Build, run, test, format, migrate, and generate Swagger docs. |
| [Recipes](./recipes.md) | Copy practical workflows for common use cases. |
| [Troubleshooting](./troubleshooting.md) | Fix common setup and runtime issues. |
| [Contributing](./contributing.md) | Work on `crank` itself. |

## Core idea

`crank` separates two concerns:

1. **Project generation** — `crank init`, `crank add`, and `crank make` write code into a generated service.
2. **Project operation** — commands like `crank run`, `crank test`, `crank migrate`, and `crank swag` wrap common tools consistently.

That means teams can standardize on a familiar Go project shape while still keeping all generated code ordinary, editable Go.

## Recommended next step

Start with [Getting started](./getting-started.md), then read [Features](./features.md) before choosing optional modules for a real project.
