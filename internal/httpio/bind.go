package httpio

import (
	"net/http"

	"bonfire-api/internal/validator"
)

func BindJSON[T any](w http.ResponseWriter, r *http.Request, validator *validator.Validator) (T, error) {
	var req T
	if err := DecodeJSON(w, r, &req); err != nil {
		return req, err
	}

	runLifecycleHooks(&req)

	if err := validator.ValidateStruct(&req); err != nil {
		return req, err
	}

	return req, nil
}

func runLifecycleHooks(v any) {
	if s, ok := v.(Sanitizable); ok {
		s.Sanitize()
	}
}
