package user

import "errors"

// UserID is a typed identifier for application users. Wrapping the primitive
// string in a named type makes it impossible to accidentally pass another
// resource's id where a user id is expected.
type UserID string

// ErrInvalidUserID is returned when a user id is empty.
var ErrInvalidUserID = errors.New("invalid user id")

// NewUserID constructs a UserID from a raw string, rejecting empty values.
func NewUserID(s string) (UserID, error) {
	if s == "" {
		return "", ErrInvalidUserID
	}
	return UserID(s), nil
}

// String returns the underlying string form of the id.
func (id UserID) String() string { return string(id) }

// IsZero reports whether the id is the zero value.
func (id UserID) IsZero() bool { return id == "" }
