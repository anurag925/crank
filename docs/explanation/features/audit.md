---
title: Audit Feature
---

# Audit feature

The `audit` feature adds a **domain event audit trail**. Every domain event published through the event bus is persisted to an `audit_events` database table, queryable by entity type and entity ID. This gives you a permanent, queryable record of all state changes without adding instrumentation logic to your domain code.

Requires `gorm`.

## What it provides

| File | Purpose |
|------|---------|
| `internal/domain/audit/event.go` | `AuditEvent` domain type (entity type, entity ID, event type, JSON payload, timestamp) |
| `internal/domain/audit/repository.go` | Audit repository port |
| `internal/ports/audit.go` | `AuditStore` port — query interface at the application boundary |
| `internal/adapters/persistence/gorm/audit_repository.go` | GORM-backed audit repository (when gorm enabled) |
| `internal/adapters/audit/logger.go` | Event bus subscriber that writes domain events to the audit store |
| `internal/application/audit/query_handler.go` | CQRS query handler for listing audit events by entity |
| `internal/adapters/http/web/v1/audit_handler.go` | HTTP handler exposing audit queries under `/api/v1/audit` |
| `db/migrations/000003_add_audit_events.up.sql` | Audit events table schema |

### How it works

1. Application handlers publish domain events through the event bus after saving aggregates.
2. The audit logger (`internal/adapters/audit/logger.go`) subscribes to all events and writes each one to the `audit_events` table via the GORM-backed repository.
3. The query handler (`internal/application/audit/query_handler.go`) retrieves events by entity type and entity ID.
4. The HTTP handler (`v1/audit_handler.go`) exposes a REST endpoint to query the audit trail.

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/audit/entities/:entity_type/:entity_id` | ✅ Bearer | List audit events for a specific entity |

## Tech stack

| Library | Purpose |
|---------|---------|
| [google/uuid](https://github.com/google/uuid) | UUID generation for audit event IDs |

## Notes

- The audit logger is wired in `cmd/server/main.go` by `crank make scaffold`. It subscribes to the event bus and writes events asynchronously.
- Audit events are write-only through the logger and read-only through the query handler — application code never creates `AuditEvent` values directly.
- The `audit_events` table is append-only. No update or delete operations are exposed.
- Undecodable event payloads are still persisted with the available metadata — the event type and entity reference are always preserved.
- Query paths are self-scoped: users can only query audit events for entities they own (IDOR protection via JWT subject matching).
