package validator

import (
	"fmt"
	"reflect"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

// Validator wraps the core third-party validation engine and exposes
// high-level methods to validate incoming HTTP request payloads.
type Validator struct {
	engine *goValidator.Validate
}

// New initializes and returns a pre-configured Validator instance.
//
// It overrides the default error formatting behavior to extract and prefer
// JSON tag names over Go struct field names when generating validation reports.
func New() *Validator {
	v := goValidator.New()

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{engine: v}
}

// ValidateStruct validates an arbitrary struct against its defined validation tags.
// It accepts any struct instance via the empty interface.
//
// If validation fails, it parses the underlying engine errors and returns a map
// where the keys are the JSON field names and the values are localized,
// user-friendly error messages.
//
// If the struct is entirely valid and no errors occur, it returns nil.
func (v *Validator) ValidateStruct(s interface{}) map[string]string {
	errors := make(map[string]string)

	err := v.engine.Struct(s)
	if err != nil {
		for _, err := range err.(goValidator.ValidationErrors) {
			errors[err.Field()] = msgForTag(err.Tag(), err.Param())
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// msgForTag translates a raw validation tag and its optional parameter
// into a localized, user-friendly error message string.
//
// It evaluates the failing validation rule (tag) against a predefined list
// of supported constraints and dynamically inserts parameters (param)
// for length-based validations like 'min' and 'max'.
func msgForTag(tag string, param string) string {
	switch tag {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Must be at least %s characters long", param)
	case "max":
		return fmt.Sprintf("Cannot be longer than %s characters", param)
	case "alphanum":
		return "Must contain only letters and numbers"
	default:
		return "Invalid value"
	}
}
