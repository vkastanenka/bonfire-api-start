package handler

import (
	"encoding/json"
	"net/http"
	"bonfire-api/internal/repository"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	DB *repository.Queries // This links your handler to your sqlc queries
}

// Request payloads
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
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

	// TODO: Add database save logic here via repository

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