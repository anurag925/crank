// Package shared defines the DomainEvent interface every aggregate-emitted event
// must implement. Concrete events live alongside the aggregate that produces
// them (e.g. internal/domain/user/events.go).
package shared

import (
	"encoding/json"
	"fmt"
	"time"
)

// DomainEvent is the minimal contract for any event raised by a domain
// aggregate. The application service collects events from the aggregate via
// PullEvents and publishes them through the EventBus port.
type DomainEvent interface {
	// EventName returns a stable dotted identifier, e.g. "user.created".
	EventName() string
	// OccurredAt returns the wall-clock time the event was recorded.
	OccurredAt() time.Time
}

// EncodeEvent serialises a DomainEvent into JSON for transport (outbox,
// queue, or HTTP). The encoded form is an EventEnvelope carrying the
// event name, occurrence time, and a JSON body that the registered
// factory (if any) decodes back into the concrete type. The body is
// obtained by marshalling the event value itself; concrete event types
// that need richer control can implement json.Marshaler.
func EncodeEvent(ev DomainEvent) ([]byte, error) {
	body, err := json.Marshal(ev)
	if err != nil {
		return nil, fmt.Errorf("encode event %q: %w", ev.EventName(), err)
	}
	env := EventEnvelope{
		Name: ev.EventName(),
		Time: ev.OccurredAt(),
		Body: body,
	}
	return json.Marshal(env)
}

// DecodeEvent reverses EncodeEvent. The event name is recovered from the
// envelope, the registered factory (if any) is consulted, and the body
// is decoded into the concrete DomainEvent. If no factory is registered
// for the name, a generic EnvelopeEvent is returned so the caller can
// still publish the event downstream.
func DecodeEvent(raw []byte) (DomainEvent, error) {
	var env EventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode event envelope: %w", err)
	}
	factory, ok := lookupEventFactory(env.Name)
	if !ok {
		return DefaultFactory(env)
	}
	return factory(env)
}
