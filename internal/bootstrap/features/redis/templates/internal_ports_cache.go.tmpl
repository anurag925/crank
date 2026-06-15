package ports

import (
	"context"
	"time"
)

// Cache is the abstraction over a key/value store. The redis feature
// provides a production implementation; tests and local development can
// supply an in-memory implementation that satisfies this interface.
type Cache interface {
	// Get returns the value stored at key, or ("", false, nil) when the
	// key is missing.
	Get(ctx context.Context, key string) (string, bool, error)
	// Set stores value at key with the supplied TTL. A zero TTL means
	// "no expiry".
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Del removes the supplied keys. Returns the number of keys that
	// were present and removed.
	Del(ctx context.Context, keys ...string) (int64, error)
}
