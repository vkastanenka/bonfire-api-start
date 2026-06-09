package auth

import (
	"bonfire-api/internal/pkg/httpio"
	"net/http"
)

type AuthHandler struct {
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// Ping confirms the auth routes are available
func (h *AuthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var data RegisterData

	// Decode incoming JSON body into the struct
	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	// Respond
	httpio.RespondJSON(w, http.StatusCreated, map[string]string{
		"message": "User registered successfully!",
	})

	return nil
}

// type AuthHandler struct {
// 	val *validator.Validator
// }

// func NewAuthHandler(val *validator.Validator) *AuthHandler {
// 	return &AuthHandler{val: val}
// }

// // Validate request body
// if validationErrs := h.val.ValidateStruct(req); validationErrs != nil {
// 	httpio.RespondJSON(w, http.StatusUnprocessableEntity, validationErrs)
// 	return
// }

// // Respond
// httpio.RespondJSON(w, http.StatusCreated, map[string]string{
// 	"message": "User registered successfully!",
// })

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
