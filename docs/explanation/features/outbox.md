---
title: Outbox Feature
---

# Outbox feature

The `outbox` feature implements the **transactional outbox pattern** — persisting domain events to a database table within the same transaction as the aggregate save, then draining them asynchronously to the event bus. This ensures reliable event delivery.

Requires `gorm` or `bun` (an ORM must be enabled).

## What it provides

| File | Purpose |
|------|---------|
| `internal/domain/outbox/event.go` | Outbox event domain type |
| `internal/domain/outbox/repository.go` | Outbox repository port |
| `internal/adapters/persistence/bun/outbox_repository.go` | Bun-backed outbox repository (when `bun` enabled) |
| `internal/adapters/persistence/gorm/outbox_repository.go` | GORM-backed outbox repository (when `gorm` enabled) |
| `internal/adapters/outbox/bun_uow.go` | Bun transactional Unit of Work (when `bun` enabled) |
| `internal/adapters/outbox/gorm_uow.go` | GORM transactional Unit of Work (when `gorm` enabled) |
| `internal/adapters/outbox/worker.go` | Background poller that drains the outbox to the event bus |
| `migrations/000002_add_outbox_events.up.sql` | Outbox table schema |
| `migrations/000002_add_outbox_events.down.sql` | Outbox table rollback |

### How it works

1. Application handler creates aggregate + domain events
2. Unit of Work saves the aggregate and writes events to `outbox_events` in a **single database transaction**
3. Background worker polls the outbox table periodically
4. Worker publishes unpublished events to the in-memory event bus
5. Published events are deleted (at-least-once delivery)

This eliminates the window between "aggregate saved" and "event published" that the default in-memory UoW has.

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| PostgreSQL | Transactional storage for the outbox table | [postgresql.org/docs](https://www.postgresql.org/docs/) |
| [golang-migrate](https://github.com/golang-migrate/migrate) | Migrations for the outbox table | [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate) |

(Depends on either Bun or GORM — see the respective feature docs.)

## Learning resources

- **Transactional outbox pattern** — [microservices.io/patterns/data/transactional-outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- **Outbox pattern explained** — [engineeringblog.yelp.com/2021/06/building-reliable-events.html](https://engineeringblog.yelp.com/2021/06/building-reliable-events.html)

## Notes

- The outbox feature replaces the default in-memory UoW with a transaction-backed version.
- The worker runs in a goroutine inside the server process. For production, consider running it as a separate process to avoid affecting request latency.
- At-least-once delivery means subscribers should be idempotent.
- Polling interval is configurable in `configs/config.yaml` under the `outbox` section.
