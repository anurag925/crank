---
title: Bun Feature
---

# Bun feature

The `bun` feature adds PostgreSQL persistence using **Bun** (by uptrace). Pass `--use-bun` during `crank init` or include `bun` in your `--features` list.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/bun/db.go` | PostgreSQL connection via Bun's `pgdriver` |
| `internal/adapters/persistence/bun/migrate.go` | Migration runner using golang-migrate |
| `internal/adapters/persistence/bun/user_repository.go` | Bun-backed `UserRepository` with row DTO pattern |
| `migrations/000001_init.up.sql` | Initial database schema |
| `migrations/000001_init.down.sql` | Initial schema rollback |

### Row DTO pattern

Repositories use a private `userRow` struct with `bun` tags. The domain aggregate has zero ORM tags. The repository accepts `bun.IDB` (interface satisfied by both `*bun.DB` and `bun.Tx`) so it works inside transactions.

## Notes

- Bun and GORM are **mutually exclusive**.
- Bun's `crank make scaffold` generates Bun-backed repository implementations plus in-memory adapter.
