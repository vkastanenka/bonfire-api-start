package validator

import (
	"bonfire-api/internal/apperr"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

// Pre-compile the regex
// No consecutive periods/underscores, cannot start or end with a period/underscore
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)

// Validator wraps the core third-party validation engine
type Validator struct {
	engine *goValidator.Validate
}

// New initializes and returns a pre-configured Validator instance.
func New() *Validator {
	v := goValidator.New()

	// Use JSON tags for validation names
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register the custom validation tag handler
	v.RegisterValidation("valid_username", func(fl goValidator.FieldLevel) bool {
		return usernameRegex.MatchString(fl.Field().String())
	})

	return &Validator{engine: v}
}

// ValidateStruct validates an arbitrary struct against its defined validation tags.
func (v *Validator) ValidateStruct(s interface{}) error {
	err := v.engine.Struct(s)
	if err == nil {
		return nil
	}

	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return apperr.New(apperr.CodeInternal, "invalid validation target provided", apperr.WithErr(err))
	}

	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		var errs []apperr.ValidationError
		for _, fieldErr := range validationErrors {
			// Using StructNamespace provides the full path (e.g., "User.Address.City")
			// You can use a more robust path cleaner than strings.Cut
			ns := fieldErr.StructNamespace()
			// Drop the first part (the struct name)
			parts := strings.Split(ns, ".")
			jsonPath := strings.Join(parts[1:], ".")
			if jsonPath == "" {
				jsonPath = fieldErr.Field()
			}

			errs = append(errs, apperr.ValidationError{
				Field:   jsonPath,
				Message: msgForFieldError(fieldErr),
			})
		}

		return apperr.New(
			apperr.CodeInvalidInput,
			"validation failed for the request payload.",
			apperr.WithValidationErrors(errs),
			apperr.WithErr(err),
		)
	}

	return apperr.New(apperr.CodeInternal, "an unknown error occurred during validation", apperr.WithErr(err))
}

// msgForFieldError evaluates the field error and returns a contextual message.
func msgForFieldError(err goValidator.FieldError) string {
	// Custom handling for empty/whitespace string edge-cases caught by 'required'
	if err.Tag() == "required" {
		val := err.Value()

		// Guard against raw nil interface values before checking reflect.TypeOf(val)
		if val != nil && reflect.TypeOf(val).Kind() == reflect.Ptr {
			sv := reflect.ValueOf(val)
			if !sv.IsNil() {
				val = sv.Elem().Interface()
			}
		}

		if valStr, ok := val.(string); ok {
			if len(valStr) > 0 && strings.TrimSpace(valStr) == "" {
				return "This field cannot consist entirely of whitespace."
			}
		}
		return "This field is required."
	}

	switch err.Tag() {
	case "email":
		return "Invalid email format."
	case "alphanum":
		return "Must contain only letters and numbers."
	case "valid_username":
		return "Must contain only lowercase letters, numbers, underscores, or periods."
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
