package appctx

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/token"
	"context"
	"net/netip"

	"github.com/google/uuid"
)

type Key int

const (
	KeyClaims Key = iota
	KeyMeta
	KeyReqID
	KeyTraceID
)

// GetMeta extracts ClientMeta from the context.
func GetMeta(ctx context.Context) (httpio.ClientMeta, error) {
	meta, ok := ctx.Value(KeyMeta).(httpio.ClientMeta)
	if !ok {
		err := errs.Internal("client metadata missing from request context").
			Reason("MISSING_CONTEXT_META")

		if reqID := GetReqID(ctx); reqID != "" {
			if reqInfo, e := errs.NewRequestInfo(reqID, ""); e == nil {
				err.Detail(reqInfo)
			}
		}
		return httpio.ClientMeta{}, err
	}
	return meta, nil
}

// GetIP extracts the client IP address from the context metadata.
func GetIP(ctx context.Context) (netip.Addr, error) {
	meta, err := GetMeta(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	return meta.IP, nil
}

// GetClaims extracts token claims from the context.
func GetClaims(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(KeyClaims).(*token.Claims)
	if !ok || claims == nil {
		return nil, errs.Unauthenticated("authentication token missing or invalid").
			Reason("AUTH_TOKEN_MISSING")
	}
	return claims, nil
}

// GetUserID extracts the authenticated user ID from token claims in the context.
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	claims, err := GetClaims(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	return claims.UserID, nil
}

// GetSessionID extracts the authenticated session ID from token claims in the context.
func GetSessionID(ctx context.Context) (uuid.UUID, error) {
	claims, err := GetClaims(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	return claims.SessionID, nil
}

// GetReqID extracts the request ID from the context if present.
func GetReqID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyReqID).(string); ok {
		return v
	}
	return ""
}

// GetTraceID extracts the trace ID from the context if present.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyTraceID).(string); ok {
		return v
	}
	return ""
}
