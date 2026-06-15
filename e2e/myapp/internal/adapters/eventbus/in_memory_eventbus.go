package eventbus

import (
	"context"
	"log/slog"

	"myapp/internal/domain/shared"
	"myapp/internal/ports"
)

// InMemory is the in-process EventBus implementation. It is safe for
// concurrent use, dispatches events synchronously to a list of in-process
// subscribers, and is intended for development, tests, and projects that
// do not require durable event distribution.
type InMemory struct {
	subs []func(ctx context.Context, ev shared.DomainEvent) error
}

// NewInMemory constructs an InMemory bus with no subscribers.
func NewInMemory() *InMemory {
	return &InMemory{}
}

// Subscribe registers a handler that receives every published event. Handlers
// are called in registration order. Subscribe returns the bus so it can be
// chained at the composition root.
func (b *InMemory) Subscribe(fn func(ctx context.Context, ev shared.DomainEvent) error) *InMemory {
	b.subs = append(b.subs, fn)
	return b
}

// Publish dispatches each event to every subscriber. A subscriber's error is
// logged but does not stop the remaining subscribers from being called.
// Returns nil unless a subscriber returns an error that requires aborting
// (the current in-memory implementation never does).
func (b *InMemory) Publish(ctx context.Context, evs ...shared.DomainEvent) error {
	for _, ev := range evs {
		for _, fn := range b.subs {
			if err := fn(ctx, ev); err != nil {
				slog.Default().Error("event bus subscriber failed",
					"event", ev.EventName(), "error", err)
			}
		}
	}
	return nil
}

// Compile-time assertion that InMemory satisfies the EventBus port.
var _ ports.EventBus = (*InMemory)(nil)
