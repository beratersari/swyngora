package domain

import (
	"context"
	"time"
)

// TagDelist is a synthetic product tag applied to pairs on the spot delist schedule.
// It appears in the Markets tag filter and in the Tags column.
const TagDelist = "Delist"

// SpotDelistEntry is a scheduled spot-pair delisting on a venue.
type SpotDelistEntry struct {
	// Exchange is the venue id (currently binance only).
	Exchange Exchange
	// Symbol is the native pair id (e.g. ADAUSDT).
	Symbol string
	// DelistTime is when trading ceases (UTC).
	DelistTime time.Time
}

// SpotDelistSchedulePort loads scheduled spot delistings from a venue (e.g. Binance).
type SpotDelistSchedulePort interface {
	// FetchSpotDelistSchedule returns upcoming (and recently announced) delist rows.
	// May require venue credentials (configured on the adapter).
	FetchSpotDelistSchedule(ctx context.Context) ([]SpotDelistEntry, error)
}

// SpotDelistStore holds the latest in-memory delist schedule for request enrichment.
type SpotDelistStore interface {
	// ReplaceAll swaps the full schedule for one exchange (call after a successful fetch).
	ReplaceAll(exchange Exchange, entries []SpotDelistEntry)
	// DelistTime returns the scheduled delist time for a symbol when known.
	DelistTime(exchange Exchange, symbol string) (time.Time, bool)
	// List returns a snapshot of all entries for an exchange (stable symbol order).
	List(exchange Exchange) []SpotDelistEntry
}
