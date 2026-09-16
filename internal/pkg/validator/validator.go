package validator

import (
	"errors"
	"strings"

	"bonfire-api/internal/pkg/errs"

	goValidator "github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator and translates errors to appErr instances.
type Validator struct {
	validate *goValidator.Validate
}

// New creates and initializes a new Validator configured with default options and optional overrides.
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

// Validate validates a struct instance and returns an appErr on constraint violations or execution errors.
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
		appErr := errs.InvalidArgument("validation failed").
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

	return errs.Internal("unexpected validation error").Wrap(err)
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

// extractFieldPath strips the root struct type name from field error namespaces, returning JSON/nested dot paths.
func extractFieldPath(fieldErr goValidator.FieldError) string {
	ns := fieldErr.Namespace()

	if idx := strings.IndexAny(ns, ".["); idx != -1 {
		if ns[idx] == '.' {
			return ns[idx+1:]
		}
		return ns[idx:]
	}

	if field := fieldErr.Field(); field != "" {
		return field
	}

	return ns
}
