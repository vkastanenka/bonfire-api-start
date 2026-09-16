package validator

import (
	"fmt"
	"reflect"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

const (
	errAlphanum               = "Must contain only letters and numbers."
	errEmail                  = "Must be a valid email address."
	errHexColor               = "Must be a valid hex code (e.g., #FF5733)."
	errInvalidConstraintValue = "Invalid value for constraint: %s"
	errMaxCollection          = "Cannot contain more than %s items."
	errMaxNumeric             = "Must be %s or less."
	errMaxString              = "Cannot be longer than %s characters."
	errMinCollection          = "Must contain at least %s items."
	errMinNumeric             = "Must be %s or greater."
	errMinString              = "Must be at least %s characters."
	errRequired               = "This field is required."
	errURL                    = "Must be a valid URL."
	errUUID                   = "Must be a valid UUID."
	errWhitespace             = "Cannot consist entirely of whitespace."
)

// msgForFieldError maps a single FieldError to a user-friendly error message.
func msgForFieldError(err goValidator.FieldError) string {
	switch err.ActualTag() {
	case "required":
		val := err.Value()
		if val != nil && reflect.TypeOf(val).Kind() == reflect.Ptr {
			sv := reflect.ValueOf(val)
			if !sv.IsNil() {
				val = sv.Elem().Interface()
			}
		}
		if valStr, ok := val.(string); ok && len(valStr) > 0 && strings.TrimSpace(valStr) == "" {
			return errWhitespace
		}
		return errRequired

	case "email":
		return errEmail
	case "alphanum":
		return errAlphanum
	case "hexcolor":
		return errHexColor
	case "uuid":
		return errUUID
	case "url":
		return errURL
	case "eqfield":
		return fmt.Sprintf("Must match field %s.", err.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of: %s.", strings.ReplaceAll(err.Param(), " ", ", "))
	case "ne":
		return fmt.Sprintf("Value cannot be %s.", err.Param())
	case "min":
		return formatRangeMessage(err, errMinString, errMinNumeric, errMinCollection)
	case "max":
		return formatRangeMessage(err, errMaxString, errMaxNumeric, errMaxCollection)
	default:
		return fmt.Sprintf(errInvalidConstraintValue, err.ActualTag())
	}
}

// formatRangeMessage formats range messages (min/max) dynamically based on the field's reflect.Kind.
func formatRangeMessage(err goValidator.FieldError, stringTmpl, numericTmpl, collectionTmpl string) string {
	switch err.Kind() {
	case reflect.String:
		return fmt.Sprintf(stringTmpl, err.Param())
	case reflect.Slice, reflect.Map, reflect.Array:
		return fmt.Sprintf(collectionTmpl, err.Param())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return fmt.Sprintf(numericTmpl, err.Param())
	default:
		return fmt.Sprintf(stringTmpl, err.Param())
	}
}
