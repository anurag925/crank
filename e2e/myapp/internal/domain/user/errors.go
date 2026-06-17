package user

import "errors"

// ErrUserNotFound is returned by the repository when a user lookup by id
// does not match any stored aggregate.
var ErrUserNotFound = errors.New("user not found")

// ErrInvalidUser is returned by the aggregate constructor when one of its
// invariants is violated (e.g. an empty name or email).
var ErrInvalidUser = errors.New("invalid user")
