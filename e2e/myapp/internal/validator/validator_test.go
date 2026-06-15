package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStruct_ValidInput(t *testing.T) {
	type input struct {
		Name  string `json:"name"  validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	err := Struct(input{Name: "Alice", Email: "alice@example.com"})
	assert.NoError(t, err)
}

func TestStruct_MissingRequired(t *testing.T) {
	type input struct {
		Name string `json:"name" validate:"required"`
	}

	err := Struct(input{})
	require.Error(t, err)

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Equal(t, 422, ve.HTTPStatus)
	assert.Contains(t, ve.Errors, "name")
	assert.Equal(t, "is required", ve.Errors["name"])
}

func TestStruct_InvalidEmail(t *testing.T) {
	type input struct {
		Email string `json:"email" validate:"required,email"`
	}

	err := Struct(input{Email: "not-an-email"})
	require.Error(t, err)

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, ve.Errors, "email")
}

func TestStruct_MultipleErrors(t *testing.T) {
	type input struct {
		Name  string `json:"name"  validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	err := Struct(input{})
	require.Error(t, err)

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Len(t, ve.Errors, 2)
}

func TestStruct_MinLength(t *testing.T) {
	type input struct {
		Name string `json:"name" validate:"required,min=3"`
	}

	err := Struct(input{Name: "ab"})
	require.Error(t, err)

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, ve.Errors["name"], "at least 3")
}

func TestStruct_MaxLength(t *testing.T) {
	type input struct {
		Name string `json:"name" validate:"required,max=5"`
	}

	err := Struct(input{Name: "toolongname"})
	require.Error(t, err)

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, ve.Errors["name"], "at most 5")
}

func TestHumanMessage_UnknownTag(t *testing.T) {
	// Build a ValidationError with an unrecognised tag to exercise the default
	// branch of humanMessage(). We cannot rely on struct tags because
	// go-playground/validator panics on unregistered custom tags.
	ve := &ValidationError{
		HTTPStatus: 422,
		Message:    "validation failed",
		Errors:     map[string]string{"val": "failed validation: customtag"},
	}

	msg := ve.Error()
	assert.Contains(t, msg, "failed validation: customtag")
}

func TestValidationError_ErrorString(t *testing.T) {
	ve := &ValidationError{
		HTTPStatus: 422,
		Message:    "validation failed",
		Errors:     map[string]string{"name": "is required", "email": "must be a valid email address"},
	}

	msg := ve.Error()
	assert.Contains(t, msg, "name: is required")
	assert.Contains(t, msg, "email: must be a valid email address")
}

func TestValidate_ReturnsSingleton(t *testing.T) {
	v := Validate()
	assert.NotNil(t, v)
}
