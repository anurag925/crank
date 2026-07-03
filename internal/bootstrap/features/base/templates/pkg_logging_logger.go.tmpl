package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// contextKey is an unexported type to avoid key collisions.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// ContextHandler wraps a slog.Handler and automatically injects
// request_id and user_id from the Go context into every log record.
// This means any code calling slog.InfoContext, slog.WarnContext, etc.
// with a context.Context gets enrichment for free — no need to call a
// helper function. Third-party code and deferred goroutines also benefit
// automatically as long as the context is propagated.
type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if uid, ok := ctx.Value(userIDKey).(string); ok && uid != "" {
		r.AddAttrs(slog.String("user_id", uid))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// New creates a slog.Logger with a three-layer handler stack:
//
//	JSONHandler → redactionHandler → ContextHandler
//
// JSON emits structured output, the redaction handler scrubs sensitive
// values, and ContextHandler auto-injects request_id and user_id from
// the Go context. When addSource is true every log entry includes a
// compressed source location (e.g. "handler/user.go:42").
func New(level slog.Leveler, addSource bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(shortSource(src))
				}
			}
			return a
		},
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	redactHandler := &redactionHandler{handler: jsonHandler}
	ctxHandler := &ContextHandler{Handler: redactHandler}

	return slog.New(ctxHandler)
}

// ParseLevel converts a config string to a slog.Level. Defaults to Info.
func ParseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// shortSource compresses a full source path to the last two directory
// components plus the file name: "/app/internal/handler/user.go" →
// "handler/user.go". If the path is shorter than three segments it is
// returned unchanged.
func shortSource(src *slog.Source) string {
	parts := strings.Split(src.File, string(filepath.Separator))
	if len(parts) <= 2 {
		return src.File
	}
	short := filepath.Join(parts[len(parts)-2:]...)
	return short + ":" + itoa(src.Line)
}

// itoa converts a positive int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithUserID stores the authenticated user ID in the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// RequestIDFromContext returns the request ID stored in the context, or an
// empty string if none is present.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// UserIDFromContext returns the user ID stored in the context.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
