package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxIdempotencyKeyLen is the max length of a client-supplied key.
	MaxIdempotencyKeyLen = 128
	// DefaultIdempotencyTTL is how long a successful key can be replayed.
	DefaultIdempotencyTTL = 24 * time.Hour
)

// IdempotencyKind classifies the cached result.
const (
	IdempotencyKindTrade       = "trade"
	IdempotencyKindPending     = "pending"
	IdempotencyKindOCO         = "oco"
	IdempotencyKindBracket     = "bracket"
	IdempotencyKindMarginOpen  = "margin_open"
	IdempotencyKindMarginClose = "margin_close"
)

// IdempotencyRecord stores one client key for a paper book.
type IdempotencyRecord struct {
	ClientID    string // book id
	Key         string
	RequestHash string
	Kind        string
	ResultJSON  string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type idempotencyCtxKey struct{}

// ContextWithIdempotency attaches a record so the store can insert it in the same write tx.
func ContextWithIdempotency(ctx context.Context, rec *IdempotencyRecord) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if rec == nil || rec.Key == "" {
		return ctx
	}
	return context.WithValue(ctx, idempotencyCtxKey{}, rec)
}

// IdempotencyFromContext returns a record attached for the current write, if any.
func IdempotencyFromContext(ctx context.Context) *IdempotencyRecord {
	if ctx == nil {
		return nil
	}
	rec, _ := ctx.Value(idempotencyCtxKey{}).(*IdempotencyRecord)
	return rec
}

// NormalizeIdempotencyKey trims and validates. Empty is allowed (no idempotency).
func NormalizeIdempotencyKey(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) > MaxIdempotencyKeyLen {
		return "", fmt.Errorf("%w: idempotency key must be at most %d characters", ErrInvalidArgument, MaxIdempotencyKeyLen)
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return "", fmt.Errorf("%w: idempotency key has invalid characters", ErrInvalidArgument)
	}
	return s, nil
}

// IdempotencyRequestHash is a stable fingerprint of the mutating request.
func IdempotencyRequestHash(parts ...string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(strings.TrimSpace(p))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
