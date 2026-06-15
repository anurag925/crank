package user

import (
	"errors"
	"strings"
)

// Email is a value object for validated email addresses.
type Email struct {
	value string
}

// ErrInvalidEmail is returned when an email value object fails validation.
var ErrInvalidEmail = errors.New("invalid email")

// NewEmail constructs an Email value object. The value is trimmed and
// lower-cased; the at-sign and dot are checked but no exhaustive RFC 5322
// parsing is performed.
func NewEmail(s string) (Email, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" || !strings.Contains(v, "@") || !strings.Contains(v, ".") {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: v}, nil
}

// String returns the email address.
func (e Email) String() string { return e.value }
