package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"bonfire-api/internal/pkg/errs"

	goValidator "github.com/go-playground/validator/v10"
)

var (
	rgxHexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)
	rgxVerCode  = regexp.MustCompile(`^[2-9A-HJ-NP-Z]{6}$`)
)

type Validator struct {
	validate *goValidator.Validate
}

func New(opts ...Option) *Validator {
	val := &Validator{
		validate: goValidator.New(),
	}

	val.applyDefaults()

	for _, opt := range opts {
		opt(val)
	}

	return val
}

// applyDefaults registers infrastructure-level tag extractors, aliases, and base validations.
func (v *Validator) applyDefaults() {
	v.validate.RegisterTagNameFunc(defaultTagNameFunc)

	for alias, tags := range defaultAliases() {
		v.validate.RegisterAlias(alias, tags)
	}

	for tag, fn := range defaultValidations() {
		_ = v.validate.RegisterValidation(tag, fn)
	}
}

func (v *Validator) Validate(s any) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return errs.Internal("failed to execute struct validation").Wrap(err)
	}

	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		appErr := errs.InvalidArgument(errValidationFailed).
			ErrorInfoReason("VALIDATION_FAILED").
			Wrap(err)

		for _, fieldErr := range validationErrors {
			appErr.FieldViolation(
				extractFieldPath(fieldErr),
				msgForFieldError(fieldErr),
				strings.ToUpper(fieldErr.ActualTag()),
			)
		}

		return appErr
	}

	return errs.Internal("Unexpected validation error.").Wrap(err)
}

func extractFieldPath(fieldErr goValidator.FieldError) string {
	ns := fieldErr.Namespace()
	if idx := strings.IndexByte(ns, '.'); idx != -1 {
		return ns[idx+1:]
	}
	if fieldErr.Field() != "" {
		return fieldErr.Field()
	}
	return ns
}

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
