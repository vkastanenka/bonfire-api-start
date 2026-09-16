package httpio

import (
	"net/http"

	"bonfire-api/internal/pkg/sanitize"
	"bonfire-api/internal/pkg/validator"
)

// Bind handles request decoding, string sanitization, and struct validation.
type Bind struct {
	validator *validator.Validator
}

// NewBind constructs a new Bind instance with the provided validator.
func NewBind(v *validator.Validator) *Bind {
	return &Bind{validator: v}
}

// JSON decodes the JSON request body into dest, then normalizes and validates the payload.
func (b *Bind) JSON(w http.ResponseWriter, r *http.Request, dest any) error {
	if err := decodeJSON(w, r, dest); err != nil {
		return err
	}

	return b.validate(dest)
}

// Query decodes URL query parameters into dest, then normalizes and validates the payload.
func (b *Bind) Query(r *http.Request, dest any) error {
	if err := decodeQuery(r, dest); err != nil {
		return err
	}

	return b.validate(dest)
}

// Path extracts URL path parameters into dest, then normalizes and validates the payload.
func (b *Bind) Path(r *http.Request, dest any) error {
	if err := decodePath(r, dest); err != nil {
		return err
	}

	return b.validate(dest)
}

// validate sanitizes input strings in req and executes field validation rules if a validator is present.
func (b *Bind) validate(req any) error {
	sanitize.Normalize(req)
	if b == nil || b.validator == nil {
		return nil
	}
	return b.validator.Validate(req)
}
