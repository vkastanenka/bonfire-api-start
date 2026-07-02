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

// TODO: Refactor apperr and update file with new implementation.

// Validator wraps third-party validation engine.
type Validator struct {
	engine *goValidator.Validate
}

// Error messages.
const (
	ErrInvalidTarget          = "Invalid validation target provided."
	ErrValidationFailed       = "Validation failed for the request payload."
	ErrUnknownValidation      = "An unknown error occurred during validation."
	ErrRequired               = "This field is required."
	ErrWhitespace             = "This field cannot consist entirely of whitespace."
	ErrEmail                  = "Invalid email format."
	ErrAlphanum               = "Must contain only letters and numbers."
	ErrUsername               = "Must contain only letters, numbers, underscores, or periods."
	ErrMinString              = "Must be at least %s characters long."
	ErrMinNumeric             = "Must be %s or greater."
	ErrMinCollection          = "Must contain at least %s items."
	ErrMaxString              = "Cannot be longer than %s characters."
	ErrMaxNumeric             = "Must be %s or less."
	ErrMaxCollection          = "Cannot contain more than %s items."
	ErrInvalidConstraintValue = "Invalid value for constraint: %s"
)

// Regex expressions.
var (
	// No consecutive periods/underscores, cannot start or end with a period/underscore
	RgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)
)

// New initializes a Validator with JSON tag resolution and domain-specific aliases.
func New() *Validator {
	v := goValidator.New()

	setupTagNameResolution(v)
	registerCustomValidations(v)
	registerDomainAliases(v)

	return &Validator{engine: v}
}

// setupTagNameResolution configures the engine to use JSON tags for field names.
func setupTagNameResolution(v *goValidator.Validate) {
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tag := fld.Tag.Get("json")
		if tag == "" || tag == "-" {
			return ""
		}
		if idx := strings.IndexByte(tag, ','); idx != -1 {
			return tag[:idx]
		}
		return tag
	})
}

// registerCustomValidations binds custom validation rules to the engine.
func registerCustomValidations(v *goValidator.Validate) {
	v.RegisterValidation("valid_username", func(fl goValidator.FieldLevel) bool {
		return RgxUsername.MatchString(fl.Field().String())
	})
}

// registerDomainAliases bundles validation constraints under domain labels.
func registerDomainAliases(v *goValidator.Validate) {
	// Identity
	v.RegisterAlias("identity_id", "required,uuid,len=36")
	v.RegisterAlias("identity_email", "required,email,max=255")
	v.RegisterAlias("identity_username", "required,min=4,max=32,valid_username")
	v.RegisterAlias("identity_password", "required,min=12,max=128")

	// Security
	v.RegisterAlias("security_password", "required,min=12,max=128")

	// Profile
	v.RegisterAlias("profile_display_name", "omitempty,min=3,max=32")
}

// ValidateStruct validates a struct and maps failures to custom application errors.
func (v *Validator) ValidateStruct(s interface{}) error {
	// Validate struct with core engine and exit early when valid
	err := v.engine.Struct(s)
	if err == nil {
		return nil
	}

	// Handle invalid validator arg
	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return apperr.New(apperr.CodeInternal, ErrInvalidTarget, apperr.WithErr(err))
	}

	// Handle validation failures
	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		invalidParams := make([]apperr.InvalidParam, 0, len(validationErrors))

		// Loop through each validation failure
		for _, fieldErr := range validationErrors {
			ns := fieldErr.StructNamespace()
			var jsonPath string

			// Extract a clean field path by removing the root struct name (e.g., "User.Age" -> "Age")
			if idx := strings.Index(ns, "."); idx != -1 {
				jsonPath = ns[idx+1:]
			} else {
				jsonPath = fieldErr.Field()
			}

			// Append field and its error message to the list
			invalidParams = append(invalidParams, apperr.InvalidParam{
				Name:   jsonPath,
				Reason: msgForFieldError(fieldErr),
			})
		}

		// Return error with all validation errors
		return apperr.New(
			apperr.CodeInvalidInput,
			ErrValidationFailed,
			apperr.WithInvalidParams(invalidParams),
			apperr.WithErr(err),
		)
	}

	// Unexpected error fallback
	return apperr.New(apperr.CodeInternal, ErrUnknownValidation, apperr.WithErr(err))
}

// msgForFieldError returns an error message for a failed validation tag.
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

		// Check if the value is a string consisting only of whitespace characters
		if valStr, ok := val.(string); ok {
			if len(valStr) > 0 && strings.TrimSpace(valStr) == "" {
				return ErrWhitespace
			}
		}
		return ErrRequired
	}

	// Map validation tags to their error messages
	switch err.Tag() {
	case "email":
		return ErrEmail
	case "alphanum":
		return ErrAlphanum
	case "valid_username":
		return ErrUsername
	case "min":
		return formatRangeMessage(err, ErrMinString, ErrMinNumeric, ErrMinCollection)
	case "max":
		return formatRangeMessage(err, ErrMaxString, ErrMaxNumeric, ErrMaxCollection)
	default:
		return fmt.Sprintf(ErrInvalidConstraintValue, err.Tag())
	}
}

// formatRangeMessage formats range errors based on the field's data type.
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
