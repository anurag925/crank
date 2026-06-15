// Package shared defines the DomainEvent interface every aggregate-emitted event
// must implement. Concrete events live alongside the aggregate that produces
// them (e.g. internal/domain/user/events.go).
package shared

import "time"

// DomainEvent is the minimal contract for any event raised by a domain
// aggregate. The application service collects events from the aggregate via
// PullEvents and publishes them through the EventBus port.
type DomainEvent interface {
	// EventName returns a stable dotted identifier, e.g. "user.created".
	EventName() string
	// OccurredAt returns the wall-clock time the event was recorded.
	OccurredAt() time.Time
}
