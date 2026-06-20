---
title: Features
---

# Features

Features are installable modules selected during `crank init` or added later with `crank add`.

```bash
crank init myapp --features=base,auth,redis
crank add temporal --project ./myapp
```

The `base` feature is always included. GORM is the default ORM unless you select Bun.

## Feature selection rules

### `base` is always present

Even if you omit it, `crank` includes `base` because every generated project needs the common layout, HTTP server, configuration, validation, logging, and development files.

### GORM is the default ORM

These two commands create a GORM-backed project:

```bash
crank init myapp --features=base
crank init myapp --features=base,auth
```

To use Bun:

```bash
crank init myapp --features=base,auth --use-bun
# or
crank init myapp --features=base,auth,bun
```

If you pass only `base`, the current CLI still adds the default ORM (`gorm`). This keeps generated projects database-ready by default.

### GORM and Bun are mutually exclusive

A project cannot contain both `gorm` and `bun`. Pick one per service.

### Some features have requirements

`outbox` requires an ORM-backed project, so it needs `gorm` or `bun`.

Temporal generators require the `temporal` feature:

```bash
crank add temporal --project ./myapp
crank make workflow OrderFulfillment --project ./myapp
```

## Available features

| Feature | Description | Typical use |
| --- | --- | --- |
| `base` | DDD layout, Echo HTTP server, config, validation, logging, Swagger plumbing, in-memory adapters, Dockerfile, `.air.toml`, and Makefile. | Every generated project. |
| `gorm` | PostgreSQL persistence with GORM, database factory, user repository, migrations, and golang-migrate integration. | Default relational database option. |
| `bun` | PostgreSQL persistence with Bun, database factory, user repository, migrations, and golang-migrate integration. | Use when you prefer Bun/uptrace. |
| `auth` | JWT auth, bcrypt password hashing, auth endpoints, JWT middleware, protected `/me` route, token service, and password/email value objects. | APIs with login/session requirements. |
| `crypto` | AES-256-GCM encryption/decryption helper backed by config-driven secret. | Encrypt sensitive values before storage or transport. |
| `redis` | Redis cache port and go-redis client wired into the composition root. | Caching, sessions, rate limiting, distributed coordination. |
| `mongodb` | MongoDB client wired into the composition root. | Document storage or workloads that fit MongoDB. |
| `qdrant` | Qdrant vector database client. | Semantic search, embedding storage, vector retrieval. |
| `temporal` | Temporal client, worker wiring, logging bridge, and workflow/activity adapter layout. | Durable workflows, background orchestration, retries. |
| `otel` | OpenTelemetry tracing with stdout exporter by default and Echo middleware spans per request. | Local tracing and observability foundations. |
| `outbox` | Transactional outbox table and worker pattern for publishing domain events after commit. Requires `gorm` or `bun`. | Reliable event publishing with PostgreSQL-backed transactions. |
| `views` | React SPA with Vite, embedded frontend served by the Go binary with dev-mode hot reload. | APIs that also serve a frontend UI. |

## Adding features after project creation

Use `crank add`:

```bash
crank add redis --project ./myapp
```

`crank add` does four things:

1. Loads `.crank.yaml` to understand the project.
2. Renders feature templates into the project.
3. Injects config sections into existing config files using marker comments.
4. Installs Go dependencies and updates the manifest.

Config injection is designed to preserve user edits. Existing files are not blindly overwritten.

## Choosing an ORM

| Choose | When |
| --- | --- |
| `gorm` | You want the default, familiar Go ORM with broad ecosystem support. |
| `bun` | You prefer Bun's query builder style and uptrace ecosystem. |
| generated in-memory adapters | Use the in-memory adapters for tests, prototypes, or code paths that should not touch the database. |

Both ORM features use PostgreSQL and golang-migrate for SQL migrations.

## Recommended combinations

### Minimal database-ready API

```bash
crank init api --features=base
```

### Authenticated relational API

```bash
crank init api --features=base,auth
```

### Bun-backed service with Redis

```bash
crank init api --features=base,auth,redis --use-bun
```

### Workflow-heavy service

```bash
crank init worker-api --features=base,auth,temporal
```

### Observable API with outbox

```bash
crank init events-api --features=base,auth,outbox,otel
```

Because GORM is the default ORM, `outbox` has an ORM available in this example.
