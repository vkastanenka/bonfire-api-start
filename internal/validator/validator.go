package validator

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/presence.go"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

var (
	v           = goValidator.New()
	rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)
)

const (
	errValidationFailed       = "Validation failed for the request."
	errInvalidConstraintValue = "Invalid value for constraint: %s"
	errRequired               = "This field is required."
	errWhitespace             = "Cannot consist entirely of whitespace."
	errEmail                  = "Must be a valid email address."
	errAlphanum               = "Must contain only letters and numbers."
	errUsername               = "Must start and end with a letter or number. May contain only letters, numbers, and non-consecutive underscores or periods."
	errMinString              = "Must be at least %s characters."
	errMinNumeric             = "Must be %s or greater."
	errMinCollection          = "Must contain at least %s items."
	errMaxString              = "Cannot be longer than %s characters."
	errMaxNumeric             = "Must be %s or less."
	errMaxCollection          = "Cannot contain more than %s items."
)

func init() {
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

	v.RegisterValidation("valid_username", func(fl goValidator.FieldLevel) bool {
		return rgxUsername.MatchString(fl.Field().String())
	})

	v.RegisterValidation("presence", func(fl goValidator.FieldLevel) bool {
		switch val := fl.Field().Interface().(type) {
		case string:
			p := presence.Parse(val)
			return p.Valid()

		case *string:
			if val == nil {
				return true
			}
			p := presence.Parse(*val)
			return p.Valid()

		case presence.Presence:
			if val == presence.PresenceUnknown {
				return true
			}
			return val.Valid()

		case *presence.Presence:
			if val == nil {
				return true
			}
			if *val == presence.PresenceUnknown {
				return true
			}
			return val.Valid()

		default:
			return false
		}
	})

	v.RegisterAlias("identity_id", "required,uuid,len=36")
	v.RegisterAlias("identity_email", "required,email,max=255")
	v.RegisterAlias("identity_username", "required,min=3,max=32,valid_username")
	v.RegisterAlias("identity_password", "required,min=12,max=255")
	v.RegisterAlias("profile_display_name", "omitempty,min=3,max=32")
}

func Validate(s interface{}) error {
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return apperr.NewInternal(err, "")
	}

	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		invalidParams := make([]apperr.InvalidParam, 0, len(validationErrors))

		for _, fieldErr := range validationErrors {
			ns := fieldErr.Namespace()
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

		return apperr.NewInvalidInput(
			err,
			errValidationFailed,
			apperr.Params(invalidParams),
		)
	}

	return apperr.NewInternal(err, "")
}

func msgForFieldError(err goValidator.FieldError) string {
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
	case "valid_username":
		return errUsername
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
