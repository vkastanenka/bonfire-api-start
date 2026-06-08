package handler

import (
	"bonfire-api/internal/repository"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

/*
TODO:
Format go files on save
*/

type UserHandler struct {
	DB *repository.Queries // This links your handler to your sqlc queries
}

/*
Register logic:

* Validate request body
2. Validate email availability
3. Validate user availability
TRANSACTION:
4. Create a new user

5. Success response
*/

// Request payloads
type RegisterRequest struct {
    Email       string `json:"email"`
    DisplayName string `json:"displayName"`
    Username    string `json:"username"`
    Password    string `json:"password"`
}

// Register handles user registration
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	
	// Decode incoming JSON body into the struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// TODO: Validate request body, respond with error if not valid
	// TODO: Check email availability, respond with error if not available.
	// TODO: Check username availability, respond with error if not available.
	// TODO: Create transaction where we create user and user profile.

	// Respond back
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully!"})
}

// Login handles user authentication
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	// For now, a placeholder stub
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": "mock-jwt-token"})
}

// GetProfile handles fetching a single user by ID
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// How to grab URL params using Chi
	userID := chi.URLParam(r, "id")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":       userID,
		"username": "GoBeginner101",
		"status":   "Offline",
	})
}