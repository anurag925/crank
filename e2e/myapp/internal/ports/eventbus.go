// Package ports defines the cross-cutting interfaces (ports) that adapters
// in the project's infrastructure layer implement. Application services
// depend only on these abstractions so the choice of transport, broker, or
// database can change without rippling outward.
package ports

import (
	"context"

	"myapp/internal/domain/shared"
)

// EventBus is the abstraction over a message broker or in-process bus used
// to publish domain events. The base feature provides an in-memory
// implementation; production deployments may swap in NATS, Kafka, RabbitMQ
// or similar without touching application code.
type EventBus interface {
	// Publish sends each event to the bus. Failures should be logged and
	// not block the calling aggregate's persistence path.
	Publish(ctx context.Context, evs ...shared.DomainEvent) error
}
