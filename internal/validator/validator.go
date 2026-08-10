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

const (
	errValidationFailed       = "Validation failed for the request."
	errInvalidConstraintValue = "Invalid value for constraint: %s"
	errRequired               = "This field is required."
	errWhitespace             = "Cannot consist entirely of whitespace."
	errEmail                  = "Must be a valid email address."
	errAlphanum               = "Must contain only letters and numbers."
	errHexColor               = "Must be a valid hex code (e.g., #FF5733)."
	errVerCode                = "Must be 6 uppercase alphanumeric characters."
	errMinString              = "Must be at least %s characters."
	errMinNumeric             = "Must be %s or greater."
	errMinCollection          = "Must contain at least %s items."
	errMaxString              = "Cannot be longer than %s characters."
	errMaxNumeric             = "Must be %s or less."
	errMaxCollection          = "Cannot contain more than %s items."
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
		if tag := fld.Tag.Get("json"); tag != "" && tag != "-" {
			if idx := strings.IndexByte(tag, ','); idx != -1 {
				return tag[:idx]
			}
			return tag
		}
		if tag := fld.Tag.Get("form"); tag != "" && tag != "-" {
			return tag
		}
		if tag := fld.Tag.Get("path"); tag != "" && tag != "-" {
			return tag
		}
		return ""
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

	instance := &Validator{
		validate: v,
	}

	return instance
}

func (v *Validator) Validate(s interface{}) error {
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
			ns := fieldErr.Namespace()
			var jsonPath string

			if idx := strings.IndexByte(ns, '.'); idx != -1 {
				jsonPath = ns[idx+1:]
			} else {
				jsonPath = fieldErr.Field()
			}

			appErr.FieldViolation(
				jsonPath,
				v.msgForFieldError(fieldErr),
				strings.ToUpper(fieldErr.ActualTag()),
			)
		}

		return appErr
	}

	return errs.Internal("Unexpected validation error.").Wrap(err)
}

func (v *Validator) msgForFieldError(err goValidator.FieldError) string {
	if err.ActualTag() == "required" {
		val := err.Value()

		if val != nil && reflect.TypeOf(val).Kind() == reflect.Ptr {
			sv := reflect.ValueOf(val)
			if !sv.IsNil() {
				val = sv.Elem().Interface()
			}
		}

		if valStr, ok := val.(string); ok {
			if len(valStr) > 0 && strings.TrimSpace(valStr) == "" {
				return errWhitespace
			}
		}
		return errRequired
	}

	switch err.ActualTag() {
	case "email":
		return errEmail
	case "alphanum":
		return errAlphanum
	case "hexcolor":
		return errHexColor
	case "vercode":
		return errVerCode
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
