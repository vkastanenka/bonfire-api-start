package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *AuthService) RevokeAllOtherSessions(ctx context.Context, userID uuid.UUID, currentSessionID uuid.UUID) error {
	// You'll need to add DeleteAllSessionsExcept to your interface
	return s.store.UserSessionDeleteAllExcept(ctx, repository.UserSessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: currentSessionID, Valid: true},
	})
}

type DeviceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	IPAddress string    `json:"ip_address"`
	IsCurrent bool      `json:"is_current"`
	LastSeen  time.Time `json:"last_seen_at"`
}

// GetDevices retrieves all active sessions and flags the current one based on the refresh token.
func (s *AuthService) GetDevices(ctx context.Context, userID uuid.UUID, currentRefreshToken string) ([]DeviceResponse, error) {
	// 1. Convert uuid.UUID to pgtype.UUID
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	// 2. Fetch sessions using the generated sqlc method
	sessions, err := s.store.UserSessionListByUser(ctx, pgUserID)
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, "Failed to fetch active devices.", apperr.WithErr(err))
	}

	var devices []DeviceResponse
	for _, sess := range sessions {
		// 3. Convert netip.Addr to string for display
		ipStr := sess.ClientIp.String()

		location := "Unknown Location"
		if sess.ClientIp.IsLoopback() {
			location = "Localhost"
		}

		devices = append(devices, DeviceResponse{
			ID:        uuid.UUID(sess.ID.Bytes).String(),
			Name:      parseDeviceName(sess.UserAgent),
			Location:  location,
			IPAddress: ipStr,
			IsCurrent: sess.RefreshToken == currentRefreshToken,
			LastSeen:  sess.LastSeenAt.Time,
		})
	}

	return devices, nil
}

// RevokeDevice deletes a specific session, ensuring it belongs to the authenticated user.
func (s *AuthService) RevokeDevice(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	err := s.store.UserSessionDelete(ctx, repository.UserSessionDeleteParams{
		ID:     pgtype.UUID{Bytes: sessionID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return apperr.New(apperr.CodeInternal, "Failed to log out of device.", apperr.WithErr(err))
	}
	return nil
}

// RevokeAllOtherDevices deletes all sessions except the one associated with the provided refresh token.
func (s *AuthService) RevokeAllOtherDevices(ctx context.Context, userID uuid.UUID, currentRefreshToken string) error {
	// 1. Fetch the current session to get its ID
	currentSession, err := s.store.UserSessionGet(ctx, currentRefreshToken)
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Current session invalid or already logged out.")
	}

	// 2. Delete everything else
	err = s.store.UserSessionDeleteAllExcept(ctx, repository.UserSessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     currentSession.ID,
	})
	if err != nil {
		return apperr.New(apperr.CodeInternal, "Failed to log out of other devices.", apperr.WithErr(err))
	}

	return nil
}

// Simple parser
func parseDeviceName(userAgent string) string {
	ua := strings.ToLower(userAgent)
	os := "Unknown OS"
	browser := "Unknown Browser"

	if strings.Contains(ua, "windows") {
		os = "Windows"
	} else if strings.Contains(ua, "mac os") {
		os = "macOS"
	} else if strings.Contains(ua, "linux") {
		os = "Linux"
	} else if strings.Contains(ua, "android") {
		os = "Android"
	} else if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		os = "iOS"
	}

	if strings.Contains(ua, "bonfire-client") { // Assuming you have a desktop client
		browser = "Bonfire Client"
	} else if strings.Contains(ua, "firefox") {
		browser = "Firefox"
	} else if strings.Contains(ua, "chrome") {
		browser = "Chrome"
	} else if strings.Contains(ua, "safari") {
		browser = "Safari"
	} else if strings.Contains(ua, "edg") {
		browser = "Edge"
	}

	return os + " (" + browser + ")"
}
