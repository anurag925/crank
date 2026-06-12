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
	loggerKey    contextKey = "logger"
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// New creates a slog.Logger configured with the given level and wrapped with
// the redaction handler. When addSource is true every log entry includes a
// compressed source location (e.g. "handler/user.go:42").
func New(level slog.Leveler, addSource bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Compress the source path to a project-relative short form.
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(shortSource(src))
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)

	// Wrap with the redaction handler to scrub sensitive keywords.
	return slog.New(&redactionHandler{handler: handler})
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

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns the logger stored in the context. If no logger is found
// the global default is returned.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithRequestID stores the request ID in the context and returns the updated
// context together with the logger enriched with the request_id attribute.
func WithRequestID(ctx context.Context, id string) (context.Context, *slog.Logger) {
	ctx = context.WithValue(ctx, requestIDKey, id)
	l := FromContext(ctx).With("request_id", id)
	return WithLogger(ctx, l), l
}

// WithUserID stores the authenticated user ID in the context and enriches
// the logger with a user_id attribute.
func WithUserID(ctx context.Context, id string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, id)
	l := FromContext(ctx).With("user_id", id)
	return WithLogger(ctx, l)
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
