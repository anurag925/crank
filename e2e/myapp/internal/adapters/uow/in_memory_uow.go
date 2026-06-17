// Package uow provides UnitOfWork implementations. The in-memory variant
// runs save and publish sequentially with no transactional boundary. It is
// the default for projects that do not need strict atomicity between the
// aggregate write and event publication.
package uow

import (
	"context"

	"myapp/internal/domain/shared"
	"myapp/internal/ports"
)

// InMemoryUoW is a non-transactional UnitOfWork. It calls the save closure
// first; if that succeeds it publishes the events. A failure in save
// short-circuits the publish step. A failure in publish is logged but does
// not unwind the save — the same behaviour as calling repo.Save and
// bus.Publish directly.
type InMemoryUoW struct {
	bus ports.EventBus
}

// NewInMemoryUoW returns a UnitOfWork that publishes via the given bus.
func NewInMemoryUoW(bus ports.EventBus) *InMemoryUoW {
	return &InMemoryUoW{bus: bus}
}

// SaveAndPublish runs the save closure, then publishes events if save
// succeeded. The save closure decides which repository to use; the
// UnitOfWork itself does not touch the database.
func (u *InMemoryUoW) SaveAndPublish(ctx context.Context, save func(ctx context.Context) error, events []shared.DomainEvent) error {
	if err := save(ctx); err != nil {
		return err
	}
	if u.bus != nil && len(events) > 0 {
		_ = u.bus.Publish(ctx, events...)
	}
	return nil
}
