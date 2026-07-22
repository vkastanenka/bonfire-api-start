package auth

const (
	EventForgotPassword     = "auth.forgot-password"
	EventRegister           = "auth.register"
	EventResendVerification = "auth.retry-verification"
)

type ForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type RegisterPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ResendVerificationPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}
