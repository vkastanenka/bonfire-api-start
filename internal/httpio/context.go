package httpio

import (
	"context"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/token"

	"net/netip"

	"github.com/google/uuid"
)

type CtxKey string

const (
	CtxClaimsKey  CtxKey = "claims"
	CtxMetaKey    CtxKey = "meta"
	CtxReqIDKey   CtxKey = "requestId"
	CtxTraceIDKey CtxKey = "traceId"
)

func CtxGetMeta(ctx context.Context) (ClientMeta, error) {
	meta, ok := ctx.Value(CtxMetaKey).(ClientMeta)
	if !ok {
		err := errs.Internal("client metadata missing from request context").
			Reason("MISSING_CONTEXT_META")

		if reqID := CtxGetReqID(ctx); reqID != "" {
			if reqInfo, e := errs.NewRequestInfo(reqID, ""); e == nil {
				err.Detail(reqInfo)
			}
		}
		return ClientMeta{IP: netip.IPv4Unspecified()}, err
	}
	return meta, nil
}

func CtxGetIP(ctx context.Context) (netip.Addr, error) {
	meta, err := CtxGetMeta(ctx)
	if err != nil {
		return netip.IPv4Unspecified(), err
	}
	return meta.IP, nil
}

func CtxGetClaims(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(CtxClaimsKey).(*token.Claims)
	if !ok {
		return nil, errs.Unauthenticated("authentication token missing or invalid").
			Reason("AUTH_TOKEN_MISSING")
	}
	return claims, nil
}

func CtxGetUserID(ctx context.Context) (uuid.UUID, error) {
	claims, err := CtxGetClaims(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func CtxGetReqID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxReqIDKey).(string); ok {
		return v
	}
	return ""
}

func CtxGetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxTraceIDKey).(string); ok {
		return v
	}
	return ""
}
