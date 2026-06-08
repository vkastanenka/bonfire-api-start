package auth

// RegisterRequest defines the input payload for creating a new user.
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email,max=255"`
	DisplayName string `json:"displayName" validate:"min=3,max=32"`
	Username    string `json:"username" validate:"required,alphanum,min=8,max=32"`
	Password    string `json:"password" validate:"required,min=8,max=100"`
}

// AuthResponse defines the standard payload sent back to successful clients.
type AuthResponse struct {
	Token string `json:"token"`
}
