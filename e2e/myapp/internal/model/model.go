// Package model holds cross-layer transport DTOs that don't fit cleanly into
// the domain or application layers. Today it contains the APIError envelope
// returned by every HTTP endpoint.
package model

// APIError is the standard JSON error envelope returned by all endpoints.
// For validation failures, Details contains per-field messages keyed by the
// JSON field name.
type APIError struct {
	Error   string            `json:"error"             example:"validation failed"`
	Details map[string]string `json:"details,omitempty"`
}
