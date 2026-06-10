package auth

// RegisterData defines the input payload for creating a new user.
type RegisterData struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	DisplayName *string `json:"displayName" validate:"omitempty,min=3,max=32"`
	Username    string  `json:"username" validate:"required,alphanum,min=8,max=32"`
	Password    string  `json:"password" validate:"required,min=8,max=100"`
}

// LoginData defines the input payload for logging in a user.
type LoginData struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	Password    string  `json:"password" validate:"required,min=8,max=100"`
}
