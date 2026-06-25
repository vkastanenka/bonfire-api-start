package cache

import "fmt"

// Hardcoded strings like "user:online:123" scattered across your app will cause bugs.
// By defining them here as pure functions, your IDE will catch typos, and you guarantee perfect namespacing.

// ForgotPasswordCooldownKey restricts password reset emails (Pillar 1)
func ForgotPasswordCooldownKey(email string) string {
	return fmt.Sprintf("auth:cooldown:forgot-password:%s", email)
}

// UserPresenceKey tracks if a user is actively connected (Pillar 2)
func UserPresenceKey(userID string) string {
	return fmt.Sprintf("presence:user:%s", userID)
}

// GuildEventChannel routes real-time chat messages to servers (Pillar 3)
func GuildEventChannel(guildID string) string {
	return fmt.Sprintf("events:guild:%s", guildID)
}
