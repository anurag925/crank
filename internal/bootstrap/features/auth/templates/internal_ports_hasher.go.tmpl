// Package ports: this file adds the Hasher port used by the auth feature.
// Application services depend only on this interface; the bcrypt adapter
// in internal/adapters/crypto provides the production implementation.
package ports

// Hasher is the abstraction for a password-hashing algorithm. The base
// feature ships no implementation; the auth feature wires the bcrypt
// adapter from internal/adapters/crypto.
type Hasher interface {
	// Hash returns a salted, one-way hash of plaintext. The returned string
	// is self-contained (carries its own salt and cost) so it can be stored
	// and verified without external state.
	Hash(plaintext string) (string, error)
	// Verify reports whether plaintext matches the supplied hash.
	Verify(hash, plaintext string) bool
}
