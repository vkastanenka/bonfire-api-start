package cache

import (
	"fmt"

	"github.com/google/uuid"
)

func AuthLoginFailuresKey(email string) string {
	return fmt.Sprintf("auth:login:failures:{%s}", email)
}

func AuthLoginLockoutKey(email string) string {
	return fmt.Sprintf("auth:login:lockout:{%s}", email)
}

func SessionKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("session:{%s}", sessionID.String())
}

func TokenBlacklistKey(jti string) string {
	return fmt.Sprintf("token:blacklist:{%s}", jti)
}

func UserPresenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:{%s}:presence", userID.String())
}

func WSTicketKey(ticket uuid.UUID) string {
	return fmt.Sprintf("ws:ticket:{%s}", ticket.String())
}

// func AuthCooldownForgotPasswordKey(email string) string {
// 	return fmt.Sprintf("auth:cooldown:forgot-password:{%s}", email)
// }

// func AuthCooldownResendVerificationKey(email string) string {
// 	return fmt.Sprintf("auth:cooldown:resend-verification:{%s}", email)
// }
