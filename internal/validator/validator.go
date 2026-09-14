package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"bonfire-api/internal/errs"

	goValidator "github.com/go-playground/validator/v10"
)

var (
	rgxHexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)
	rgxVerCode  = regexp.MustCompile(`^[2-9A-HJ-NP-Z]{6}$`)
)

type Validator struct {
	validate *goValidator.Validate
}

func New() *Validator {
	v := goValidator.New()

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tagKey := range []string{"json", "form", "path"} {
			if tag := fld.Tag.Get(tagKey); tag != "" && tag != "-" {
				if idx := strings.IndexByte(tag, ','); idx != -1 {
					return tag[:idx]
				}
				return tag
			}
		}
		return fld.Name
	})

	v.RegisterAlias("token", "max=1024")

	_ = v.RegisterValidation("hexcolor", func(fl goValidator.FieldLevel) bool {
		str := fl.Field().String()
		return str == "" || rgxHexColor.MatchString(str)
	})

	_ = v.RegisterValidation("vercode", func(fl goValidator.FieldLevel) bool {
		str := fl.Field().String()
		return str == "" || rgxVerCode.MatchString(str)
	})

	return &Validator{
		validate: v,
	}
}

func (v *Validator) Validate(s any) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return errs.Internal("Failed to execute struct validation.").Wrap(err)
	}

	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		appErr := errs.InvalidArgument(errValidationFailed).
			Reason("VALIDATION_FAILED").
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
	case "vercode":
		return errVerCode
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
