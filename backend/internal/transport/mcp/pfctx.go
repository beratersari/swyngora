package mcp

import (
	"context"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
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

type situationReasonKey struct{}

type situationReasonValue struct {
	clauses []string
}

// WithSituationReason marks a buy's clause set on ctx. An empty list stores no read.
func WithSituationReason(ctx context.Context, clauses []string) context.Context {
	cp := append([]string(nil), clauses...)
	return context.WithValue(ctx, situationReasonKey{}, situationReasonValue{clauses: cp})
}

// SituationReasonFrom reports whether this call named the clauses a buy stores.
func SituationReasonFrom(ctx context.Context) (clauses []string, ok bool) {
	v, ok := ctx.Value(situationReasonKey{}).(situationReasonValue)
	if !ok {
		return nil, false
	}
	return append([]string(nil), v.clauses...), true
}

func applySituationReason(ctx context.Context, body map[string]any) {
	clauses, ok := SituationReasonFrom(ctx)
	if !ok {
		return
	}
	if clauses == nil {
		clauses = []string{}
	}
	body["reason"] = clauses
}

func reasonInput(ctx context.Context) domain.SituationReason {
	clauses, ok := SituationReasonFrom(ctx)
	if !ok {
		return domain.SituationReason{}
	}
	return domain.SituationReason{Set: true, Clauses: clauses}
}

// withToolReason reads a comma-separated clause list. "none" stores no read. Empty leaves the whole read.
func withToolReason(ctx context.Context, raw string) (context.Context, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ctx, nil
	}
	var clauses []string
	if raw != "none" {
		clauses = strings.Split(raw, ",")
	}
	reason, err := domain.ParseSituationReason(clauses)
	if err != nil {
		return ctx, err
	}
	return WithSituationReason(ctx, reason.Clauses), nil
}
