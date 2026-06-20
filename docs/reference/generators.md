---
title: Generators
---

# Generators

`crank make` generates application code inside an existing `crank` project.

```bash
crank make <kind> <name> [field:type ...]
```

Generators are useful after the initial scaffold, when you want to add resources, handlers, migrations, Temporal workflows, or Temporal activities.

## Resource names

You can pass names in common forms:

```bash
crank make scaffold OrderItem
crank make scaffold order_item
crank make scaffold order-items
```

`crank` derives singular, plural, PascalCase, camelCase, snake_case, and kebab-case forms for file names, route paths, struct names, and table names.

## Field specs

Fields use this format:

```text
name:type
```

The type is optional and defaults to `string`.

Examples:

```bash
crank make scaffold Product title:string price:float active:bool
crank make scaffold Customer email:email external_id:uuid
crank make model Subscription starts_at:time plan:text seats:int
```

Supported field types:

| Type | Go type | PostgreSQL type | Validation |
| --- | --- | --- | --- |
| `string` | `string` | `TEXT` | `required` |
| `text` | `string` | `TEXT` | `required` |
| `int` | `int` | `INTEGER` | `required` |
| `int64` | `int64` | `BIGINT` | `required` |
| `float` | `float64` | `DOUBLE PRECISION` | `required` |
| `float64` | `float64` | `DOUBLE PRECISION` | `required` |
| `bool` | `bool` | `BOOLEAN` | none |
| `time` | `time.Time` | `TIMESTAMPTZ` | `required` |
| `uuid` | `string` | `UUID` | `required,uuid` |
| `email` | `string` | `TEXT` | `required,email` |

## Generator kinds

### `model`

Generates the domain layer for a resource:

- aggregate
- typed ID value object
- domain events
- domain errors
- repository port
- migration when GORM or Bun is enabled, unless `--skip-migration` is used

```bash
crank make model Order customer:string total:float
```

Use this when you want to start from the domain and add adapters later.

### `repository`

Generates the persistence adapter for a resource.

```bash
crank make repository Order
```

Behavior depends on the project features:

| Project feature | Repository generated |
| --- | --- |
| `gorm` | GORM-backed PostgreSQL adapter plus in-memory adapter support. |
| `bun` | Bun-backed PostgreSQL adapter plus in-memory adapter support. |
| no ORM | In-memory adapter. |

If the domain aggregate already exists and you do not pass fields, `crank` tries to infer fields from the existing domain file so generated code remains compatible.

### `service`

Generates application command/query files:

- command definitions
- command handler
- query definitions
- query handler

```bash
crank make service Order
```

Use this when you already have the domain and persistence pieces but need application use cases.

### `handler`

Generates an Echo HTTP handler and wires routes automatically.

```bash
crank make handler Product title:string price:float
```

By default, `handler` also generates the required domain, application, and persistence files if they do not already exist. Use `--only` to create only the handler:

```bash
crank make handler Product --only
```

Route wiring is idempotent. If automatic wiring fails, `crank` prints a manual hint and leaves existing files untouched.

### `scaffold`

Generates the full stack for a resource:

- domain aggregate/value objects/events/errors
- repository port
- application command/query handlers
- persistence adapter
- HTTP handler
- route wiring
- migration when an ORM is enabled
- optional tests

```bash
crank make scaffold Invoice number:string amount:float --tests
```

This is the best default when you want a complete CRUD resource.

### `workflow`

Generates a Temporal workflow.

```bash
crank make workflow OrderFulfillment order_id:uuid
```

Requires the `temporal` feature:

```bash
crank add temporal --project ./myapp
```

The generator writes the workflow under the Temporal adapter area and registers it with the worker using marker-based wiring.

### `activity`

Generates a Temporal activity.

```bash
crank make activity ChargeCard amount:float --tests
```

Requires the `temporal` feature. Activity logging uses the Temporal SDK activity logger, which the generated worker bridges to `slog`.

### `migration`

Creates a blank SQL migration pair:

```bash
crank make migration create_orders
```

Generated files are timestamped:

```text
migrations/20260619123045_create_orders.up.sql
migrations/20260619123045_create_orders.down.sql
```

## Useful flags

### `--project`

Target a project from another directory:

```bash
crank make scaffold Book title:string --project ./bookstore
```

### `--tests`

Generate `_test.go` files beside generated layers:

```bash
crank make scaffold Book title:string --tests
```

For ORM-backed projects, repository and handler tests are intentionally lightweight and do not require a live database. In-memory paths can exercise full CRUD behavior.

### `--only`

Generate only the requested kind:

```bash
crank make handler Book --only
```

Use this when you already created or hand-wrote the dependency layers.

### `--force`

Overwrite the primary generated artifact if it already exists:

```bash
crank make handler Book --force
```

Non-primary dependency files are skipped when they already exist.

### `--skip-migration`

Avoid generating a migration even when GORM or Bun is enabled:

```bash
crank make scaffold AuditLog message:text --skip-migration
```

## A complete example

```bash
crank init bookstore --features=base,auth
cd bookstore
cp .env.example .env

crank make scaffold Book title:string author:string price:float --tests
crank migrate up
crank test
crank run
```

Then call your generated endpoints according to the routes registered in `internal/adapters/http/web/routes.go`.
