---
title: Crank Documentation
---

# Crank

`crank` is a CLI for creating and maintaining production-ready Go backend services. It scaffolds a layered service, installs selected features, and wraps day-to-day development tools as first-class `crank` subcommands.

Instead of stitching together routing, configuration, validation, persistence, migrations, Swagger, live reload, and project scripts by hand, you choose the features you need and let `crank` generate a working project with sensible defaults.

## What you can build

A generated `crank` project gives you:

- A Go 1.26 backend service using Echo v5.
- A Domain-Driven layout with domain, application, adapter, and composition-root layers.
- Configuration through `configs/config.yaml`, `.env`, and environment variables.
- Structured logging with `log/slog` including redaction and context-aware enrichment.
- Request validation with `go-playground/validator` via automatic binder validation.
- Swagger/OpenAPI generation with `swaggo/swag`.
- Optional PostgreSQL persistence through GORM, where the domain aggregate doubles as the GORM model.
- Optional modules for auth (JWT with token revocation), Redis, MongoDB, Qdrant, Temporal, OpenTelemetry, React views, crypto helpers, transactional outbox, and audit trailing.
- `TxRepositories` — transaction-scoped domain repos so application handlers never import persistence adapters.
- Versioned HTTP handlers at `/api/v1` with self-scoped user endpoints.
- One CLI surface for build, run, test, format, vet, migrations, Swagger, live reload, health checks, and code generation.
- Agent-friendly project metadata through `.crank.yaml`, `AGENTS.md`, and a project-local agent skill.

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

| Section | Page | Use it when you want to... |
| --- | --- | --- |
| **📚 Tutorials** | [Installation](./tutorials/installation.md) | Install `crank` or build it from source. |
| | [Getting started](./tutorials/getting-started.md) | Scaffold your first app and run it locally. |
| **📖 Reference** | [Commands](./reference/commands.md) | Learn the full CLI surface. |
| | [Features](./reference/features.md) | Pick the modules to include in a project. |
| | [Generators](./reference/generators.md) | Generate models, handlers, scaffolds, workflows, activities, and migrations. |
| | [Configuration](./reference/configuration.md) | Configure generated applications safely. |
| | [Project structure](./reference/project-structure.md) | Quick reference for where generated code lives. |
| | [Quick reference](./reference/quick-reference.md) | Cheat sheet for commands, features, and field types. |
| **🧠 Explanation** | [Navigating the generated application](./explanation/generated-app.md) | **Deep dive into the generated code — architecture diagrams, request lifecycle, layer walkthroughs, feature modules, and testing patterns.** |
| | [AI agent support](./explanation/ai-agents.md) | Understand how generated projects guide AI agents to use Crank correctly. |
| | [Feature: Base](./explanation/features/base.md) | Foundation — Echo, Viper, config, validation, logging, in-memory adapters. |
| | [Feature: GORM](./explanation/features/gorm.md) | PostgreSQL persistence via GORM (default ORM). |
| | [Feature: Auth](./explanation/features/auth.md) | JWT authentication, bcrypt hashing, auth endpoints. |
| | [Feature: Crypto](./explanation/features/crypto.md) | AES-256-GCM encryption helpers. |
| | [Feature: Redis](./explanation/features/redis.md) | Redis caching client and port interface. |
| | [Feature: MongoDB](./explanation/features/mongodb.md) | MongoDB document database client. |
| | [Feature: Qdrant](./explanation/features/qdrant.md) | Qdrant vector database client. |
| | [Feature: Temporal](./explanation/features/temporal.md) | Temporal workflow orchestration. |
| | [Feature: OpenTelemetry](./explanation/features/otel.md) | Distributed tracing with OpenTelemetry. |
| | [Feature: Outbox](./explanation/features/outbox.md) | Transactional outbox for reliable event delivery. |
| | [Feature: Views](./explanation/features/views.md) | React SPA with Vite, embedded in Go binary. |
| **🔧 How-to Guides** | [Development workflow](./how-to/development-workflow.md) | Build, run, test, format, migrate, and generate Swagger docs. |
| | [Recipes](./how-to/recipes.md) | Copy practical workflows for common use cases. |
| | [Troubleshooting](./how-to/troubleshooting.md) | Fix common setup and runtime issues. |
| | [Hosting](./how-to/hosting.md) | Host the documentation site. |
| | [Contributing](./how-to/contributing.md) | Work on `crank` itself. |

## Core idea

`crank` separates two concerns:

1. **Project generation** — `crank init`, `crank add`, and `crank make` write code into a generated service.
2. **Project operation** — commands like `crank run`, `crank test`, `crank migrate`, and `crank swag` wrap common tools consistently.

That means teams can standardize on a familiar Go project shape while still keeping all generated code ordinary, editable Go.

## Recommended next step

Start with [Getting started](./tutorials/getting-started.md), then read [Features](./reference/features.md) before choosing optional modules for a real project.
