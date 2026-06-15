package user

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidPassword is returned when a password value object fails its
// invariant (e.g. the supplied plaintext is too short).
var ErrInvalidPassword = errors.New("invalid password")

// Password is a value object wrapping a bcrypt hash. Construct one with
// NewPassword; the plaintext never lives on the aggregate.
type Password struct {
	hash string
}

// NewPassword hashes the supplied plaintext with bcrypt and returns a
// Password value object. The cost defaults to bcrypt.DefaultCost.
func NewPassword(plaintext string) (Password, error) {
	if len(plaintext) < 8 {
		return Password{}, ErrInvalidPassword
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return Password{}, err
	}
	return Password{hash: string(h)}, nil
}

// PasswordFromHash wraps a pre-existing bcrypt hash (e.g. one rehydrated
// from the persistence layer) into a Password value object. It does not
// validate the hash — call Authenticate to verify a candidate plaintext.
func PasswordFromHash(hash string) Password {
	return Password{hash: hash}
}

// Hash returns the stored bcrypt hash.
func (p Password) Hash() string { return p.hash }

// Authenticate reports whether the supplied plaintext matches the stored
// hash. Returns false on any bcrypt error so callers can treat it as a
// simple predicate.
func (p Password) Authenticate(plaintext string) bool {
	if p.hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plaintext)) == nil
}
