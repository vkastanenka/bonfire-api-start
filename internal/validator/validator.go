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

// --- VALIDATOR TYPES ---

// Validator
type Validator struct {
	engine *goValidator.Validate
}

// --- VALIDATOR CONSTANTS ---

// Errors
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

// Regex
var (
	// No consecutive periods/underscores, cannot start or end with a period/underscore
	RgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)
)

// --- VALIDATOR INITIALIZATION ---

// New
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
		return RgxUsername.MatchString(fl.Field().String())
	})

	// --- Identity Domain ---
	v.RegisterAlias("identity.email", "required,email,max=255")
	v.RegisterAlias("identity.username", "required,min=4,max=32,valid_username")

	// --- Security Domain ---
	v.RegisterAlias("security.password", "required,min=12,max=128")

	// --- Profile ---
	v.RegisterAlias("profile.display_name", "omitempty,min=3,max=32")

	return &Validator{engine: v}
}

// --- VALIDATOR METHODS ---

// ValidateStruct
func (v *Validator) ValidateStruct(s interface{}) error {
	err := v.engine.Struct(s)
	if err == nil {
		return nil
	}

	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return apperr.New(apperr.CodeInternal, ErrInvalidTarget, apperr.WithErr(err))
	}

	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		// Pre-allocate slice capacity for better performance
		invalidParams := make([]apperr.InvalidParam, 0, len(validationErrors))

		for _, fieldErr := range validationErrors {
			ns := fieldErr.StructNamespace()
			var jsonPath string
			if idx := strings.Index(ns, "."); idx != -1 {
				jsonPath = ns[idx+1:]
			} else {
				jsonPath = fieldErr.Field()
			}

			invalidParams = append(invalidParams, apperr.InvalidParam{
				Name:   jsonPath,
				Reason: msgForFieldError(fieldErr),
			})
		}

		return apperr.New(
			apperr.CodeInvalidInput,
			ErrValidationFailed,
			apperr.WithInvalidParams(invalidParams),
			apperr.WithErr(err),
		)
	}

	return apperr.New(apperr.CodeInternal, ErrUnknownValidation, apperr.WithErr(err))
}

// --- VALIDATOR HELPERS ---

// msgForFieldError
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
				return ErrWhitespace
			}
		}
		return ErrRequired
	}

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

// formatRangeMessage
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
