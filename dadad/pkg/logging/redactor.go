package logging

import (
	"context"
	"log/slog"
	"strings"
)

// sensitiveKeys is a set of lowercase keywords that trigger value redaction
// when they appear in an attribute key (case-insensitive substring match).
var sensitiveKeys = map[string]struct{}{
	"password":   {},
	"passwd":     {},
	"secret":     {},
	"token":      {},
	"api_key":    {},
	"apikey":     {},
	"auth":       {},
	"credential": {},
	"private":    {},
	"ssn":        {},
	"credit":     {},
	"card":       {},
}

// redactionMask is the replacement value for redacted fields.
const redactionMask = "[REDACTED]"

// redactionHandler wraps a slog.Handler and replaces values whose keys
// contain sensitive keywords with "[REDACTED]". It is safe for concurrent
// use — all mutable state is local to each Handle call.
type redactionHandler struct {
	handler slog.Handler
}

func (h *redactionHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *redactionHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create a new record with the same time/level/message but filtered attrs.
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(h.redactAttr(a))
		return true
	})
	return h.handler.Handle(ctx, nr)
}

func (h *redactionHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	filtered := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		filtered = append(filtered, h.redactAttr(a))
	}
	return &redactionHandler{handler: h.handler.WithAttrs(filtered)}
}

func (h *redactionHandler) WithGroup(name string) slog.Handler {
	return &redactionHandler{handler: h.handler.WithGroup(name)}
}

// redactAttr returns a copy of the attribute with its value redacted if the
// key contains a sensitive keyword. Group values are recursed into.
func (h *redactionHandler) redactAttr(a slog.Attr) slog.Attr {
	if isSensitive(a.Key) {
		a.Value = slog.StringValue(redactionMask)
		return a
	}

	// Recurse into group values.
	if a.Value.Kind() == slog.KindGroup {
		group := a.Value.Group()
		for i, g := range group {
			group[i] = h.redactAttr(g)
		}
		a.Value = slog.GroupValue(group...)
	}

	// Also scrub string values that look like they contain credentials
	// (e.g. "password=abc123" in a generic "msg" field).
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(redactValue(a.Value.String()))
	}

	return a
}

// isSensitive reports whether a key contains any sensitive keyword as a
// case-insensitive substring.
func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for kw := range sensitiveKeys {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// redactValue replaces embedded key=value pairs where the key is sensitive.
// For example, "connecting with password=hunter2" becomes
// "connecting with password=[REDACTED]".
func redactValue(s string) string {
	// Fast path: skip if no '=' present.
	if !strings.Contains(s, "=") {
		return s
	}

	for kw := range sensitiveKeys {
		idx := strings.Index(strings.ToLower(s), kw+"=")
		if idx < 0 {
			continue
		}
		// Find the start of the value (after "keyword=").
		valStart := idx + len(kw) + 1
		if valStart >= len(s) {
			continue
		}
		// The value extends to the next space, comma, or end of string.
		valEnd := len(s)
		for j := valStart; j < len(s); j++ {
			if s[j] == ' ' || s[j] == ',' || s[j] == ';' || s[j] == '&' {
				valEnd = j
				break
			}
		}
		s = s[:valStart] + redactionMask + s[valEnd:]
	}
	return s
}
