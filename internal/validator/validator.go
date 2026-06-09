package validator

import (
	"bonfire-api/internal/apperr"
	"errors"
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
// If validation fails, it returns a structured *apperr.Error containing field-level
// details, which can be safely bubbled up through the HTTP layer.
func (v *Validator) ValidateStruct(s interface{}) *apperr.Error {
	err := v.engine.Struct(s)
	if err == nil {
		return nil
	}

	// Guard against initialization or type errors to prevent panics
	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return apperr.NewInternal("Invalid validation target provided", apperr.WithErr(err))
	}

	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		errsMap := make(map[string]string, len(validationErrors))
		for _, fieldErr := range validationErrors {
			errsMap[fieldErr.Field()] = msgForFieldError(fieldErr)
		}

		// Wrap the validation map cleanly into your domain error model
		return apperr.NewInvalidInput(
			"Validation failed for the request payload.",
			apperr.WithDetails(errsMap),
			apperr.WithErr(err),
		)
	}

	return apperr.NewInternal("An unknown error occurred during validation", apperr.WithErr(err))
}

// msgForFieldError evaluates the field error and returns a contextual message.
func msgForFieldError(err goValidator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required."
	case "email":
		return "Invalid email format."
	case "alphanum":
		return "Must contain only letters and numbers."
	case "min":
		return formatRangeMessage(err, "Must be at least %s characters long.", "Must be %s or greater.", "Must contain at least %s items.")
	case "max":
		return formatRangeMessage(err, "Cannot be longer than %s characters.", "Must be %s or less.", "Cannot contain more than %s items.")
	default:
		return fmt.Sprintf("Invalid value for constraint: %s", err.Tag())
	}
}

// formatRangeMessage differentiates between string length and numeric value boundaries.
func formatRangeMessage(err goValidator.FieldError, stringTmpl, numericTmpl, collectionTmpl string) string {
	switch err.Kind() {
	case reflect.String:
		return fmt.Sprintf(stringTmpl, err.Param())
	case reflect.Slice, reflect.Map, reflect.Array:
		return fmt.Sprintf(collectionTmpl, err.Param())
	default:
		return fmt.Sprintf(numericTmpl, err.Param())
	}
}
