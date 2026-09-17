package httpio

import (
	"context"
	"net/netip"

	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/token"

	"github.com/google/uuid"
)

type CtxKey int

const (
	CtxKeyClaims CtxKey = iota
	CtxKeyMeta
	CtxKeyReqID
	CtxKeyTraceID
)

// CtxGetMeta extracts ClientMeta from the context.
func CtxGetMeta(ctx context.Context) (ClientMeta, error) {
	meta, ok := ctx.Value(CtxKeyMeta).(ClientMeta)
	if !ok {
		err := errs.Internal("client metadata missing from request context").
			Reason("MISSING_CONTEXT_META")

		if reqID := CtxGetReqID(ctx); reqID != "" {
			if reqInfo, e := errs.NewRequestInfo(reqID, ""); e == nil {
				err.Detail(reqInfo)
			}
		}
		return ClientMeta{}, err
	}
	return meta, nil
}

// CtxGetIP extracts the client IP address from the context metadata.
func CtxGetIP(ctx context.Context) (netip.Addr, error) {
	meta, err := CtxGetMeta(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	return meta.IP, nil
}

// CtxGetClaims extracts token claims from the context.
func CtxGetClaims(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(CtxKeyClaims).(*token.Claims)
	if !ok || claims == nil {
		return nil, errs.Unauthenticated("authentication token missing or invalid").
			Reason("AUTH_TOKEN_MISSING")
	}
	return claims, nil
}

// CtxGetUserID extracts the authenticated user ID from token claims in the context.
func CtxGetUserID(ctx context.Context) (uuid.UUID, error) {
	claims, err := CtxGetClaims(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	return claims.UserID, nil
}

// CtxGetSessionID extracts the authenticated session ID from token claims in the context.
func CtxGetSessionID(ctx context.Context) (uuid.UUID, error) {
	claims, err := CtxGetClaims(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	return claims.SessionID, nil
}

// CtxGetReqID extracts the request ID from the context if present.
func CtxGetReqID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyReqID).(string); ok {
		return v
	}
	return ""
}

// CtxGetTraceID extracts the trace ID from the context if present.
func CtxGetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyTraceID).(string); ok {
		return v
	}
	return ""
}
