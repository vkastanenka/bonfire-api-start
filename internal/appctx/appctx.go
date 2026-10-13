package appctx

import (
	"context"
	"errors"
	"net/netip"

	"bonfire-api/internal/token"
)

var (
	ErrClientMetaNotFound = errors.New("client metadata missing from context")
	ErrClaimsNotFound     = errors.New("auth claims missing or invalid from context")
)

type contextKey int

const (
	keyClaims contextKey = iota
	keyMeta
	keyReqID
	keyTraceID
)

// ClientMeta holds client connection metadata extracted from an HTTP request.
type ClientMeta struct {
	IP        netip.Addr
	UserAgent string
	OS        string
	Client    string
}

func WithClientMeta(ctx context.Context, meta ClientMeta) context.Context {
	return context.WithValue(ctx, keyMeta, meta)
}

func WithClaims(ctx context.Context, claims *token.Claims) context.Context {
	return context.WithValue(ctx, keyClaims, claims)
}

func WithReqID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, keyReqID, reqID)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, keyTraceID, traceID)
}

// GetClientMeta extracts ClientMeta from the context.
func GetClientMeta(ctx context.Context) (ClientMeta, error) {
	meta, ok := ctx.Value(keyMeta).(ClientMeta)
	if !ok {
		return ClientMeta{}, ErrClientMetaNotFound
	}
	return meta, nil
}

// GetClaims extracts token claims from the context.
func GetClaims(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(keyClaims).(*token.Claims)
	if !ok {
		return nil, ErrClaimsNotFound
	}
	return claims, nil
}

// GetReqID extracts the request ID from the context if present.
func GetReqID(ctx context.Context) string {
	if v, ok := ctx.Value(keyReqID).(string); ok {
		return v
	}
	return ""
}

// GetTraceID extracts the trace ID from the context if present.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(keyTraceID).(string); ok {
		return v
	}
	return ""
}
