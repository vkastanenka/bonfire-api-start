package auth

import (
	"bonfire-api/internal/httpio"
	"context"
	"net/http"

	"github.com/google/uuid"
)

type RegisterService interface {
	EnableTOTP(ctx context.Context, userID uuid.UUID, secret string, code string) error
	ForgotPassword(ctx context.Context, email string) error
	GenerateTOTP(ctx context.Context, userID uuid.UUID) (string, string, error)
	GetDevices(ctx context.Context, userID uuid.UUID, currentRefreshToken string) ([]DeviceResponse, error)
	Login(ctx context.Context, req LoginRequest, userAgent string, clientIP string) (map[string]string, error)
	RefreshAccessToken(ctx context.Context, oldRefreshToken string) (map[string]string, error)
	Register(ctx context.Context, req RegisterRequest) error
	ResendVerificationEmail(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, tokenStr string, newPassword string) error
	RevokeAllOtherDevices(ctx context.Context, userID uuid.UUID, currentRefreshToken string) error
	RevokeAllOtherSessions(ctx context.Context, userID uuid.UUID, currentSessionID uuid.UUID) error
	RevokeDevice(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error
	ValidateMFAToken(tokenStr string) (uuid.UUID, error)
	VerifyEmail(ctx context.Context, tokenStr string) error
	VerifyLogin2FA(ctx context.Context, mfaToken string, code string, userAgent string, clientIP string) (map[string]string, error)
}

type RequestValidator interface {
	ValidateStruct(s interface{}) error
}

type AuthHandler struct {
	service RegisterService
	val     RequestValidator
}

func NewHandler(service RegisterService, val RequestValidator) *AuthHandler {
	return &AuthHandler{service: service, val: val}
}

// Ping confirms the auth routes are available
func (h *AuthHandler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})

	return nil
}
