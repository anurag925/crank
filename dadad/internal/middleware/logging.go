package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"

	"dadad/pkg/logging"
)

const (
	// RequestIDHeader is the HTTP header used to propagate request IDs.
	RequestIDHeader = "X-Request-ID"
)

// RequestLogger returns Echo middleware that:
//   - Generates a unique request ID (or uses the one from X-Request-ID)
//   - Stores the request ID and a request-scoped logger in context
//   - Adds the request ID as a response header
//   - Logs a structured "request completed" entry at the end with:
//     request_id, method, path, status, latency, client_ip, bytes_out
//
// Downstream handlers can retrieve the request-scoped logger via
// logging.FromContext(c.Request().Context()) to automatically include
// the request_id in every log entry.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()

			// Use the inbound request ID if present, otherwise generate one.
			id := req.Header.Get(RequestIDHeader)
			if id == "" {
				id = generateID()
			}

			// Store request ID in context and enrich the logger.
			ctx, reqLogger := logging.WithRequestID(req.Context(), id)
			c.SetRequest(req.WithContext(ctx))

			// Propagate the request ID to the response.
			res.Header().Set(RequestIDHeader, id)

			// Execute downstream handlers.
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			latency := time.Since(start)

			attrs := []slog.Attr{
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.String("latency", latency.String()),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.String("client_ip", c.RealIP()),
				slog.Int64("bytes_out", res.Size),
			}

			if query := req.URL.RawQuery; query != "" {
				attrs = append(attrs, slog.String("query", query))
			}

			// Log at the appropriate level based on status code.
			level := slog.LevelInfo
			msg := "request completed"
			if res.Status >= 500 {
				level = slog.LevelError
				msg = "request completed with server error"
			} else if res.Status >= 400 {
				level = slog.LevelWarn
				msg = "request completed with client error"
			}

			reqLogger.LogAttrs(ctx, level, msg, attrs...)
			return nil
		}
	}
}

// generateID creates a 16-byte random hex string (32 characters).
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
