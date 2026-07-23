---
title: Configuration
type: reference
---

# Configuration

Generated projects use a layered configuration strategy:

1. `configs/config.yaml` for safe defaults.
2. `.env` for local secrets and machine-specific values.
3. Real environment variables for production and CI/CD.

Environment variables take precedence over `.env`, and `.env` takes precedence over YAML defaults.

## Files

### `configs/config.yaml`

Committed to source control.

Use it for:

- application name
- host and port defaults
- log level defaults
- non-secret feature defaults
- database host/port/name defaults for local development

Do not put production secrets here.

### `.env.example`

Committed to source control.

Use it as a template showing which environment variables a developer or deployment needs.

### `.env`

Not committed.

Use it locally for:

- database passwords
- JWT secrets
- API keys
- local overrides

Create it from the example file:

```bash
cp .env.example .env
```

### `.crank.yaml`

Committed to source control.

This is the `crank` project manifest. It is not application runtime configuration. It tells the CLI which features are enabled and what module path the project uses.

AI coding agents should also use `.crank.yaml` to detect Crank-generated projects and avoid assuming optional features are installed. See [AI agent support](../explanation/ai-agents.md).

## Common application settings

A generated project commonly includes YAML like this:

```yaml
app:
  name: myapp
  host: 0.0.0.0
  port: 8080
  env: development

logging:
  level: debug
  format: json
```

ORM-backed projects include database settings:

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  name: myapp
  sslmode: disable
```

Auth-backed projects include JWT settings:

```yaml
jwt:
  expiration: 24h
  refresh_expiration: 168h
```

Secrets such as `JWT_SECRET` should come from `.env` or the environment.

## Database URL resolution

`crank migrate` resolves the database URL in this order:

1. `--database-url`
2. `DATABASE_URL`
3. `database:` values from `configs/config.yaml`

Examples:

```bash
crank migrate up --database-url postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable
DATABASE_URL=postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable crank migrate up
crank migrate up
```

## Feature config injection

When you run:

```bash
crank add redis --project ./myapp
```

`crank` injects the Redis config fields and YAML/env sections into existing files. It uses marker comments such as:

```go
// crank:config-fields
// crank:config-structs
// crank:config-defaults
```

and YAML/env markers such as:

```text
# crank:config-section
# crank:env-section
```

This keeps additions idempotent and preserves your edits.

## Adding validation rules

Generated projects include validator setup in:

```text
internal/validator/validator.go
internal/validator/errors.go
```

To add a custom validation tag:

```go
validate.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
    return strings.TrimSpace(fl.Field().String()) != ""
})
```

Then use it in request structs:

```go
type CreateOrderInput struct {
    ProductID string `json:"product_id" validate:"required,uuid"`
    Quantity  int    `json:"quantity" validate:"required,gt=0,lte=999"`
    Notes     string `json:"notes" validate:"max=500"`
}
```

Add a human-readable error message in `internal/validator/errors.go` so API clients receive clear feedback.

## Production recommendations

- Set secrets as real environment variables or through your platform's secret manager.
- Keep `configs/config.yaml` free of passwords and tokens.
- Use a strong `JWT_SECRET` in auth-enabled projects.
- Set `APP_ENV=production` or the generated equivalent environment field.
- Use `DATABASE_URL` for migrations in CI/CD.
- Run `crank doctor`, `crank test`, and `crank vet` before deploying.
