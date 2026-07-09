package cache

import (
	"fmt"

	"github.com/google/uuid"
)

func AuthSessionKey(sessionID string) string {
	return fmt.Sprintf("auth:session:{%s}", sessionID)
}

func AuthCooldownForgotPasswordKey(email string) string {
	return fmt.Sprintf("auth:cooldown:forgot-password:{%s}", email)
}

func AuthLoginFailuresKey(email string) string {
	return fmt.Sprintf("auth:login:failures:{%s}", email)
}

func AuthLoginLockoutKey(email string) string {
	return fmt.Sprintf("auth:login:lockout:{%s}", email)
}

func AuthCooldownResendVerificationKey(email string) string {
	return fmt.Sprintf("auth:cooldown:resend-verification:{%s}", email)
}

func UserPresenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:{%s}:presence", userID.String())
}
