package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is the singleton validator instance used across the application.
var validate *validator.Validate

// init creates and configures the default validator. It is safe to call
// Init() again after adding custom registrations.
func init() {
	Init()
}

// Init (re)creates the shared validator instance and applies the default
// configuration (JSON tag-based field names). Call this after registering
// custom validators or translations so they take effect.
func Init() {
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Use the JSON tag as the field name in error messages so API consumers
	// see the same casing they send in request bodies.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// ---------------------------------------------------------------
	// Register custom validators here.
	//
	// Example — a "notblank" validator that rejects whitespace-only strings:
	//
	//   validate.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
	//       return strings.TrimSpace(fl.Field().String()) != ""
	//   })
	//
	// See https://github.com/go-playground/validator for the full list of
	// built-in tags and the custom-validation API.
	// ---------------------------------------------------------------
}

// Validate returns the shared validator instance.
func Validate() *validator.Validate {
	return validate
}

// Struct validates the supplied struct and returns a structured ValidationError
// when the input is invalid. Returns nil on success.
func Struct(s any) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	return toValidationError(err)
}
