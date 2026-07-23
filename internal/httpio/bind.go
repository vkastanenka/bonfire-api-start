package httpio

import (
	"net/http"

	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/validator"
)

type Bind struct {
	validator *validator.Validator
}

func NewBind(v *validator.Validator) *Bind {
	return &Bind{validator: v}
}

func (b *Bind) JSON(w http.ResponseWriter, r *http.Request, dest any) error {
	if err := decodeJSON(w, r, dest); err != nil {
		return err
	}

	return b.validate(dest)
}

func (b *Bind) Query(r *http.Request, dest any) error {
	if err := decodeQuery(r, dest); err != nil {
		return err
	}

	return b.validate(dest)
}

func (b *Bind) Path(r *http.Request, dest any) error {
	if err := decodePath(r, dest); err != nil {
		return err
	}

	return b.validate(dest)
}

func (b *Bind) validate(req any) error {
	sanitize.Normalize(req)
	if b == nil || b.validator == nil {
		return nil
	}
	return b.validator.Validate(req)
}
