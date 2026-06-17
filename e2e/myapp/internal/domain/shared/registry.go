package shared

import (
	"encoding/json"
	"time"
)

// EventEnvelope is the wire/persisted shape of a DomainEvent. It carries
// the event name, occurrence time, and the JSON-marshaled body of the
// concrete event. Concrete event types are responsible for registering a
// factory that decodes an envelope back into a typed event.
type EventEnvelope struct {
	Name string          `json:"name"`
	Time time.Time       `json:"time"`
	Body json.RawMessage `json:"body,omitempty"`
}

// EventFactory decodes an EventEnvelope into a concrete DomainEvent.
// Concrete event types register a factory via RegisterEventType in an
// init block.
type EventFactory func(env EventEnvelope) (DomainEvent, error)

var eventFactories = map[string]EventFactory{}

// RegisterEventType installs a factory for a named event type. Calling
// RegisterEventType twice with the same name panics — it is intended to be
// called from package init blocks alongside the event type's definition.
func RegisterEventType(name string, factory EventFactory) {
	if _, exists := eventFactories[name]; exists {
		panic("shared: event type " + name + " already registered")
	}
	eventFactories[name] = factory
}

func lookupEventFactory(name string) (EventFactory, bool) {
	f, ok := eventFactories[name]
	return f, ok
}

// EnvelopeEvent is the generic DomainEvent returned by DecodeEvent when
// no concrete factory is registered. It satisfies the interface so
// transports can pass it through unchanged.
type EnvelopeEvent struct {
	Envelope EventEnvelope
}

// EventName implements DomainEvent.
func (e *EnvelopeEvent) EventName() string { return e.Envelope.Name }

// OccurredAt implements DomainEvent.
func (e *EnvelopeEvent) OccurredAt() time.Time { return e.Envelope.Time }

// DefaultFactory is the factory used when no concrete factory has been
// registered for an event name. It returns a generic EnvelopeEvent so the
// outbox worker can still publish the event to the bus.
func DefaultFactory(env EventEnvelope) (DomainEvent, error) {
	return &EnvelopeEvent{Envelope: env}, nil
}
