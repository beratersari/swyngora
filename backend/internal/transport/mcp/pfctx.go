package mcp

import (
	"context"
	"strings"
)

type portfolioIDKey struct{}

// WithPortfolioID stores the selected paper book id on ctx for HTTP/in-process adapters.
func WithPortfolioID(ctx context.Context, id string) context.Context {
	id = strings.TrimSpace(id)
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, portfolioIDKey{}, id)
}

// PortfolioIDFrom returns the optional selected book id.
func PortfolioIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(portfolioIDKey{}).(string)
	return strings.TrimSpace(v)
}

type idempotencyKeyCtx struct{}

// WithIdempotencyKey stores a client retry key on ctx for place/close tools.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	key = strings.TrimSpace(key)
	if key == "" {
		return ctx
	}
	return context.WithValue(ctx, idempotencyKeyCtx{}, key)
}

// IdempotencyKeyFrom returns the optional retry key.
func IdempotencyKeyFrom(ctx context.Context) string {
	v, _ := ctx.Value(idempotencyKeyCtx{}).(string)
	return strings.TrimSpace(v)
}
