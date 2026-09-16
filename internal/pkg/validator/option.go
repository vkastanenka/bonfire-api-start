package validator

import (
	goValidator "github.com/go-playground/validator/v10"
)

// Option configures the Validator instance.
type Option func(*Validator)

// WithAlias registers a struct tag alias (e.g., alias="password", tags="min=12,max=255").
func WithAlias(alias, tags string) Option {
	return func(v *Validator) {
		v.validate.RegisterAlias(alias, tags)
	}
}

// WithValidation registers a custom validation tag and function.
func WithValidation(tag string, fn goValidator.Func) Option {
	return func(v *Validator) {
		_ = v.validate.RegisterValidation(tag, fn)
	}
}

// WithTagNameFunc registers a custom struct field name extractor for error reporting.
func WithTagNameFunc(fn goValidator.TagNameFunc) Option {
	return func(v *Validator) {
		v.validate.RegisterTagNameFunc(fn)
	}
}
