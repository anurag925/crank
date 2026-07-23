---
title: Transactional Outbox Pattern
type: concept
---

# Transactional Outbox Pattern in Crank

When building distributed microservices, publishing domain events (e.g. to Redis Pub/Sub, Kafka, or RabbitMQ) directly inside HTTP handlers often leads to dual-write inconsistencies. If the database commit succeeds but the message broker call fails, data becomes desynchronized.

`crank` solves this problem by providing a built-in **Transactional Outbox Pattern** module (`--features=outbox`).

---

## How It Works

```
                     ┌──────────────────────────────────────────────────┐
                     │              Application Handler                 │
                     └────────────────────────┬─────────────────────────┘
                                              │
                                              ▼
┌───────────────────────────────────────────────────────────────────────────────────────┐
│  Atomic Database Transaction (Unit of Work)                                           │
│                                                                                       │
│   1. INSERT INTO orders (...) ───────────────► Write Business State                   │
│   2. INSERT INTO outbox_messages (...) ──────► Write Event Payload                    │
│                                                                                       │
└─────────────────────────────────────────────┬─────────────────────────────────────────┘
                                              │ Commit
                                              ▼
                                   ┌────────────────────┐
                                   │  PostgreSQL DB     │
                                   └──────────┬─────────┘
                                              │ Polling / Listen
                                              ▼
                                   ┌────────────────────┐
                                   │   Outbox Worker    │
                                   └──────────┬─────────┘
                                              │ Publish
                                              ▼
                                   ┌────────────────────┐
                                   │   Redis / NATS /   │
                                   │   Message Broker   │
                                   └────────────────────┘
```

---

## Core Components

### 1. The Outbox Table
When the `outbox` feature is enabled, Crank generates an `outbox_messages` table migration:

```sql
CREATE TABLE outbox_messages (
    id UUID PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_outbox_pending ON outbox_messages (status, created_at);
```

---

### 2. Publishing Events inside Unit of Work

Your domain repository or application handler writes events directly to the outbox repository within the *same atomic database transaction*:

```go
func (h *OrderHandler) CreateOrder(ctx context.Context, req CreateOrderRequest) error {
	return h.uow.Do(ctx, func(repos uow.TxRepositories) error {
		// 1. Save Aggregate
		order, err := domain.NewOrder(req.CustomerID, req.Total)
		if err != nil {
			return err
		}
		if err := repos.Orders().Save(ctx, order); err != nil {
			return err
		}

		// 2. Queue Outbox Event inside the SAME transaction
		eventPayload, _ := json.Marshal(OrderCreatedEvent{
			OrderID: order.ID,
			Total:   order.Total,
		})

		return repos.Outbox().Publish(ctx, "orders.created", eventPayload)
	})
}
```

---

### 3. The Asynchronous Outbox Worker

Crank launches a background worker process (`internal/adapters/outbox/worker.go`) that periodically fetches `pending` outbox messages, dispatches them to external message brokers, and updates their status to `processed`.

- **At-Least-Once Delivery**: Messages are guaranteed to be published even if the service crashes mid-request.
- **Automatic Retries**: Failed message deliveries are retried with exponential backoff before being marked as `failed`.
- **Idempotency Support**: Every outbox message contains a unique `uuid.UUID` event ID that consumers can use for deduplication.

---

## Enabling Outbox in Your Project

To include the transactional outbox in a new service:

```bash
crank init myapp --features=base,gorm,outbox
```

Or add it to an existing GORM project:

```bash
crank add outbox
```
