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
	// ErrSupplyUnmapped: base is not in the Binance marketing supply snapshot.
	ErrSupplyUnmapped = errors.New("supply unmapped")
	// ErrCatalogUnmapped: no CoinMarketCap id in the Binance marketing catalog.
	ErrCatalogUnmapped = errors.New("catalog unmapped")
	// ErrHoldersUnpublished: CMC has an id but no holder table for this asset.
	ErrHoldersUnpublished = errors.New("holders unpublished")
)
