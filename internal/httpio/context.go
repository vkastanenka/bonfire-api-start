package httpio

import (
	"context"
	"net/netip"

	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/token"

	"github.com/google/uuid"
)

type ctxKey int

const (
	ctxKeyClaims ctxKey = iota
	ctxKeyMeta
	ctxKeyReqID
	ctxKeyTraceID
)

// CtxGetMeta extracts ClientMeta from the context.
func CtxGetMeta(ctx context.Context) (ClientMeta, error) {
	meta, ok := ctx.Value(ctxKeyMeta).(ClientMeta)
	if !ok {
		err := errs.Internal("client metadata missing from request context").
			ErrorInfoReason("MISSING_CONTEXT_META")

		if reqID := CtxGetReqID(ctx); reqID != "" {
			if reqInfo, e := errs.NewRequestInfo(reqID, ""); e == nil {
				err.AddDetail(reqInfo)
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
	claims, ok := ctx.Value(ctxKeyClaims).(*token.Claims)
	if !ok || claims == nil {
		return nil, errs.Unauthenticated("authentication token missing or invalid").
			ErrorInfoReason("AUTH_TOKEN_MISSING")
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
	if v, ok := ctx.Value(ctxKeyReqID).(string); ok {
		return v
	}
	return ""
}

// CtxGetTraceID extracts the trace ID from the context if present.
func CtxGetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTraceID).(string); ok {
		return v
	}
	return ""
}
