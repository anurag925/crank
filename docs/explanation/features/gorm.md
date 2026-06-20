---
title: GORM Feature
---

# GORM feature

The `gorm` feature adds PostgreSQL persistence using the **GORM** ORM. This is the default ORM — if you don't specify `--use-bun`, GORM is added automatically.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/gorm/db.go` | PostgreSQL connection factory via GORM |
| `internal/adapters/persistence/gorm/migrate.go` | Migration runner using golang-migrate |
| `internal/adapters/persistence/gorm/user_repository.go` | GORM-backed `UserRepository` |
| `migrations/000001_init.up.sql` | Initial database schema |
| `migrations/000001_init.down.sql` | Initial schema rollback |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [GORM v2](https://gorm.io) | Go ORM with rich feature set | [gorm.io/docs](https://gorm.io/docs) |
| [GORM PostgreSQL driver](https://gorm.io/docs/connecting_to_the_database.html) | PostgreSQL connectivity | [gorm.io/docs](https://gorm.io/docs/connecting_to_the_database.html) |
| [golang-migrate](https://github.com/golang-migrate/migrate) | Database migration tool | [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate) |
| PostgreSQL | Relational database | [postgresql.org/docs](https://www.postgresql.org/docs/) |

## Learning resources

- **GORM documentation** — [gorm.io/docs](https://gorm.io/docs) (associations, hooks, preloading, raw SQL)
- **GORM tutorials** — [gorm.io/docs](https://gorm.io/docs/) (getting started, CRUD, querying)
- **golang-migrate** — [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate) (migration files, CLI usage)
- **PostgreSQL documentation** — [postgresql.org/docs](https://www.postgresql.org/docs/)

## Notes

- GORM and Bun are **mutually exclusive** — a project cannot have both.
- Running `crank make scaffold` with GORM enabled generates a migration for the new resource (unless `--skip-migration` is passed).
