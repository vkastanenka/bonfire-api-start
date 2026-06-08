package auth

import (
	"bonfire-api/internal/pkg/httpio"
	"bonfire-api/internal/pkg/validator"
	"net/http"
)

type AuthHandler struct {
	val *validator.Validator
}

func NewHandler(val *validator.Validator) *AuthHandler {
	return &AuthHandler{val: val}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	// Decode incoming JSON body into the struct
	if ok := httpio.DecodeJSON(w, r, &req); !ok {
		return
	}

	// Validate request body
	if validationErrs := h.val.ValidateStruct(req); validationErrs != nil {
		httpio.RespondJSON(w, http.StatusUnprocessableEntity, validationErrs)
		return
	}

	// err = h.auth.CreateUserAccount(r.Context(), req)
	// if err != nil {
	// 	w.Header().Set("Content-Type", "application/json")

	// 	// If your service returns a known error, handle it cleanly
	// 	switch err {
	// 	case ErrEmailTaken:
	// 		w.WriteHeader(http.StatusConflict) // 409 status code
	// 		json.NewEncoder(w).Encode(map[string]string{"email": "Email is already in use"})
	// 	case ErrUsernameTaken:
	// 		w.WriteHeader(http.StatusConflict) // 409 status code
	// 		json.NewEncoder(w).Encode(map[string]string{"username": "Username is already taken"})
	// 	default:
	// 		w.WriteHeader(http.StatusInternalServerError) // 500 status code
	// 		json.NewEncoder(w).Encode(map[string]string{"error": "An unexpected error occurred"})
	// 	}
	// 	return
	// }

	// Respond
	httpio.RespondJSON(w, http.StatusCreated, map[string]string{
		"message": "User registered successfully!",
	})
}
