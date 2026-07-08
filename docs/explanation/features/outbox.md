---
title: Outbox Feature
---

# Outbox feature

The `outbox` feature implements the **transactional outbox pattern**. Domain events are persisted to `outbox_events` within the same database transaction as the aggregate save, then drained asynchronously to the event bus via a background worker.

Requires `gorm` or `bun`.

## What it provides

| File | Purpose |
|------|---------|
| `internal/domain/outbox/event.go` | Outbox event domain type |
| `internal/domain/outbox/repository.go` | Outbox repository port |
| `internal/adapters/outbox/gorm_uow.go` | GORM transactional UoW with `TxRepositories` |
| `internal/adapters/outbox/bun_uow.go` | Bun transactional UoW with `TxRepositories` |
| `internal/adapters/persistence/gorm/outbox_repository.go` | GORM-backed outbox repository |
| `internal/adapters/persistence/bun/outbox_repository.go` | Bun-backed outbox repository |
| `internal/adapters/outbox/worker.go` | Background poller + publisher |
| `migrations/000002_add_outbox_events.up.sql` | Outbox table schema |

### How it works

1. Command handler calls `uow.SaveAndPublish(ctx, func(ctx, repos) { repos.Users().Save(ctx, u) }, events)`
2. GormUoW/BunUoW runs the save closure inside a transaction with transaction-scoped repos (`TxRepositories`)
3. After the save succeeds, outbox rows are appended in the same transaction
4. Background worker polls `outbox_events`, decodes events, publishes to EventBus
5. Undecodable events are marked published to prevent infinite loops

## Notes

- The outbox replaces the default in-memory UoW with a transaction-backed version.
- The `TxRepositories` pattern means command handlers never import `*gorm.DB` — they call `repos.Users().Save(ctx, u)`.
- At-least-once delivery — subscribers should be idempotent.
- Polling interval configurable under the `outbox` section.
