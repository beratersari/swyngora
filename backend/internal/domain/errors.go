package domain

import "errors"

// Domain-level sentinel errors. Transport maps these to HTTP status codes.
var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	// ErrConflict is a state conflict (e.g. export already running for this client).
	ErrConflict = errors.New("conflict")
	// ErrIdempotencyHit means this write collided with an existing idempotency key (replay).
	ErrIdempotencyHit = errors.New("idempotency key already used")
	ErrUpstream       = errors.New("upstream data source error")
	ErrRateLimited    = errors.New("rate limited by upstream")
)
