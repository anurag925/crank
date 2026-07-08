---
title: GORM Feature
---

# GORM feature

The `gorm` feature adds PostgreSQL persistence using **GORM**. This is the default ORM — if you don't specify `--use-bun`, GORM is added automatically.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/gorm/db.go` | PostgreSQL connection factory via GORM |
| `internal/adapters/persistence/gorm/migrate.go` | Migration runner using golang-migrate |
| `internal/adapters/persistence/gorm/user_repository.go` | GORM-backed `UserRepository` with row DTO pattern |
| `migrations/000001_init.up.sql` | Initial database schema |
| `migrations/000001_init.down.sql` | Initial schema rollback |

### Row DTO pattern

Repositories use a private `userRow` struct with GORM tags. The domain aggregate has zero ORM tags:

```go
type userRow struct {
    ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
    Name      string    `gorm:"column:name;not null"`
    Email     string    `gorm:"column:email;not null;uniqueIndex"`
    CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
    UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (row *userRow) toAggregate() *user.User {
    return user.Rehydrate(row.ID, row.Name, row.Email, "", row.CreatedAt, row.UpdatedAt)
}
```

The repository saves the row DTO via `db.Save(rowFromAggregate(u))` and scans into the row via `db.First(&row, "id = ?", id)`. Domain aggregates stay pure — no `gorm` tags, no `TableName()` method.

## Notes

- GORM and Bun are **mutually exclusive**.
- `crank make scaffold` generates a GORM-backed repository with the row DTO pattern plus an in-memory adapter.
