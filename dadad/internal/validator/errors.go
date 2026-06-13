package validator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError is returned when struct validation fails. It carries a map
// of field-level error messages keyed by the JSON field name so API consumers
// can programmatically inspect which fields failed and why.
type ValidationError struct {
	HTTPStatus int               `json:"-"`
	Message    string            `json:"message"`
	Errors     map[string]string `json:"errors"`
}

func (e *ValidationError) Error() string {
	var parts []string
	for field, msg := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(parts, "; ")
}

// toValidationError converts go-playground/validator errors into our
// structured ValidationError. Unrecognised tags fall back to a generic
// "is invalid" message.
func toValidationError(err error) *ValidationError {
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return &ValidationError{
			HTTPStatus: http.StatusUnprocessableEntity,
			Message:    "validation failed",
			Errors:     map[string]string{"body": err.Error()},
		}
	}

	fields := make(map[string]string, len(errs))
	for _, fe := range errs {
		fields[fe.Field()] = humanMessage(fe)
	}

	return &ValidationError{
		HTTPStatus: http.StatusUnprocessableEntity,
		Message:    "validation failed",
		Errors:     fields,
	}
}

// humanMessage returns a readable sentence for a single field error.
func humanMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		if fe.Kind().String() == "string" {
			return fmt.Sprintf("must be at least %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		if fe.Kind().String() == "string" {
			return fmt.Sprintf("must be at most %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}

// ErrorResponse is the standard JSON envelope returned by API error handlers.
// For validation failures, Details carries the per-field error map.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}
