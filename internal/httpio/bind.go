package httpio

import (
	"net/http"

	"bonfire-api/internal/validator"
)

// BindJSON
func BindJSON[T any](w http.ResponseWriter, r *http.Request, validator *validator.Validator) (T, error) {
	var req T
	if err := DecodeJSON(w, r, &req); err != nil {
		return req, err
	}

	// If the struct implements our Sanitizable interface, run it
	if s, ok := any(&req).(Sanitizable); ok {
		s.Sanitize()
	}

	// Validate
	if err := validator.ValidateStruct(&req); err != nil {
		return req, err
	}

	return req, nil
}
