// Package ports defines the cross-cutting interfaces (ports) that adapters
// in the project's infrastructure layer implement. Application services
// depend only on these abstractions so the choice of transport, broker, or
// database can change without rippling outward.
package ports

import (
	"context"

	"myapp/internal/domain/shared"
)

// UnitOfWork abstracts the atomicity boundary between persisting an
// aggregate and publishing the events it recorded. Implementations decide
// whether the operations are wrapped in a database transaction (the outbox
// pattern) or run sequentially (the default in-memory unit of work).
//
// Application services use this port instead of calling repo.Save and
// bus.Publish directly so the same code can run against either backend
// without modification.
type UnitOfWork interface {
	// SaveAndPublish persists the aggregate (via the supplied closure) and
	// then publishes the recorded events. The closure runs first; the
	// returned error short-circuits the publish step. The events slice is
	// consumed even on error paths so the caller does not need to reset
	// the aggregate's internal buffer.
	SaveAndPublish(ctx context.Context, save func(ctx context.Context) error, events []shared.DomainEvent) error
}
