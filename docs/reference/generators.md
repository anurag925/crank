---
title: Generators
---

# Generators

`crank make` generates application code inside an existing `crank` project.

```bash
crank make <kind> <name> [field:type ...]
```

## Resource names

You can pass names in common forms:

```bash
crank make scaffold OrderItem
crank make scaffold order_item
crank make scaffold order-items
```

`crank` derives singular, plural, PascalCase, camelCase, snake_case, and kebab-case forms for file names, route paths, struct names, and table names.

## Field specs

```text
name:type
```

Examples:

```bash
crank make scaffold Product title:string price:float active:bool
crank make scaffold Customer email:email external_id:uuid
crank make model Subscription starts_at:time plan:text seats:int
```

Supported field types:

| Type | Go type | PostgreSQL type |
| --- | --- | --- |
| `string` | `string` | `TEXT` |
| `text` | `string` | `TEXT` |
| `int` | `int` | `INTEGER` |
| `int64` | `int64` | `BIGINT` |
| `float` | `float64` | `DOUBLE PRECISION` |
| `float64` | `float64` | `DOUBLE PRECISION` |
| `bool` | `bool` | `BOOLEAN` |
| `time` | `time.Time` | `TIMESTAMPTZ` |
| `uuid` | `uuid.UUID` | `UUID` |

All aggregates use `uuid.UUID` from `github.com/google/uuid` for the primary ID. The ID holds the aggregate identity and is validated against `uuid.Nil`. Commands carry `ID string`, parsed by the command handler.

## Generator kinds

### `model`

Generates the domain layer: aggregate, domain events, errors, repository port, and migration when an ORM is enabled.

```bash
crank make model Order customer:string total:float
```

### `repository`

Generates the persistence adapter for a resource. Uses GORM or in-memory depending on enabled features. All repos use the row DTO pattern — a private `{name}Row` struct with ORM tags, `toAggregate()` via `Rehydrate()`, and `rowFromAggregate()`. Domain aggregates have zero ORM tags.

```bash
crank make repository Order
```

### `service`

Generates application command/query files using the `uow.TxRepositories` pattern: `repos.{Name}s().Save(ctx, u)`.

```bash
crank make service Order
```

### `handler`

Generates a v1 Echo HTTP handler and wires routes into `internal/adapters/http/web/v1/routes.go`. Uses `api.Error` for responses.

```bash
crank make handler Product title:string price:float
```

### `scaffold`

Full stack: domain, application, persistence, HTTP handler in `web/v1/`, route wiring, migrations, optional tests. Also splices new repository accessors into `TxRepositories` interfaces.

```bash
crank make scaffold Invoice number:string amount:float --tests
```

### `workflow` / `activity`

Temporal workflow or activity. Requires `temporal` feature.

### `migration`

Timestamped SQL up/down pair.

### `seed`

Generate seed data files or run seed up/down.

```bash
crank make seed                       # Scaffold seed infrastructure
crank make seed User                  # Generate seed data for User model
crank make seed User --count 20       # 20 rows of fake data
crank make seed up                    # Apply seed data
crank make seed down                  # Roll back seed data
```

The seed generator parses the domain aggregate's exported fields and Go struct tags, then produces a Go file with GORM-backed insert statements. Each seed file includes:

- `Seed<Model>Up(db *gorm.DB) error` — inserts entries with `clause.OnConflict{DoNothing: true}` for idempotent re-runs
- `Seed<Model>Down(db *gorm.DB) error` — deletes entries by a well-known UUID prefix

The seed files import and use the real domain types directly — no DTOs.

### `swag`

Generate Swagger/OpenAPI documentation:

```bash
crank make swag
```

Runs `swag init -g cmd/server/main.go -o docs --parseInternal --parseDependency`.

### `skill`

Regenerate the `.agents/skills/crank-project/SKILL.md` file:

```bash
crank make skill
```

This keeps AI coding assistants in sync with the project's current architecture conventions.

## Flags

| Flag | Description |
| --- | --- |
| `--project` | Target project directory. Defaults to `.` |
| `--only` | Generate only the requested kind, skipping dependencies |
| `--force` | Overwrite primary artifact if it exists |
| `--skip-migration` | Skip migration generation |
| `--tests` | Generate `_test.go` files |
| `--seed` | Also generate a seed file (used with scaffold/model/handler) |
| `--count` | Number of seed rows to generate. Defaults to 10. |
