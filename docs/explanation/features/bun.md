---
title: Bun Feature
---

# Bun feature

The `bun` feature adds PostgreSQL persistence using **Bun** (by uptrace). Pass `--use-bun` during `crank init` or include `bun` in your `--features` list to use it instead of the default GORM.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/bun/db.go` | PostgreSQL connection via Bun's `pgdriver` |
| `internal/adapters/persistence/bun/migrate.go` | Migration runner using golang-migrate |
| `internal/adapters/persistence/bun/user_repository.go` | Bun-backed `UserRepository` |
| `migrations/000001_init.up.sql` | Initial database schema |
| `migrations/000001_init.down.sql` | Initial schema rollback |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [Bun](https://bun.uptrace.dev) | SQL-first Go ORM with query builder | [bun.uptrace.dev](https://bun.uptrace.dev) |
| [pgdriver](https://bun.uptrace.dev) | PostgreSQL driver used by Bun | [bun.uptrace.dev](https://bun.uptrace.dev) |
| [golang-migrate](https://github.com/golang-migrate/migrate) | Database migration tool | [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate) |
| PostgreSQL | Relational database | [postgresql.org/docs](https://www.postgresql.org/docs/) |

## Learning resources

- **Bun documentation** — [bun.uptrace.dev](https://bun.uptrace.dev) (query builder, models, migrations, relations)
- **Bun GitHub** — [github.com/uptrace/bun](https://github.com/uptrace/bun)
- **golang-migrate** — [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- **PostgreSQL documentation** — [postgresql.org/docs](https://www.postgresql.org/docs/)

## Notes

- Bun and GORM are **mutually exclusive** — a project cannot have both.
- Bun's `crank make scaffold` generates Bun-backed repository implementations + create-table migrations.
- If you're already using the uptrace ecosystem (OpenTelemetry, go-redis), Bun fits naturally alongside those tools.
