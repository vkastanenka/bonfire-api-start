package cache

import "fmt"

// AuthSessionKey tracks user session metadata for fast retrieval.
func AuthSessionKey(sessionID string) string {
	return fmt.Sprintf("auth:session:{%s}", sessionID)
}

// AuthCooldownForgotPassword restricts password reset emails.
func AuthCooldownForgotPasswordKey(email string) string {
	return fmt.Sprintf("auth:cooldown:forgot-password:{%s}", email)
}

// AuthLoginFailuresKey tracks consecutive failed login attempts.
func AuthLoginFailuresKey(email string) string {
	return fmt.Sprintf("auth:login:failures:{%s}", email)
}

// AuthLoginLockoutKey actively blocks login attempts after exceeding failure thresholds.
func AuthLoginLockoutKey(email string) string {
	return fmt.Sprintf("auth:login:lockout:{%s}", email)
}

// AuthCooldownResendVerificationKey restricts how often a user can request a verification email.
func AuthCooldownResendVerificationKey(email string) string {
	return fmt.Sprintf("auth:cooldown:resend-verification:{%s}", email)
}

// PresenceUserKey tracks if a user is actively connected.
func PresenceUserKey(userID string) string {
	return fmt.Sprintf("presence:user:{%s}", userID)
}

// StatusUserKey tracks a user's custom status choice.
func StatusUserKey(userID string) string {
	return fmt.Sprintf("status:user:{%s}", userID)
}

// EventsGuildKey routes real-time chat messages to servers.
func EventsGuildKey(guildID string) string {
	return fmt.Sprintf("events:guild:{%s}", guildID)
}
