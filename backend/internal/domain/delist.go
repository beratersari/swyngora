package domain

import (
	"context"
	"time"
)

// TagDelist is a synthetic product tag applied to pairs on the spot delist schedule.
// It appears in the Markets tag filter and in the Tags column.
const TagDelist = "Delist"

// DelistVisibleHorizon is how far ahead a scheduled delist is promoted onto
// the default TRADING markets list (about one calendar month).
const DelistVisibleHorizon = 31 * 24 * time.Hour

// DelistPastHorizon is how long an already-halted pair stays on the Markets
// list with a Delist tag (last 30 days).
const DelistPastHorizon = 30 * 24 * time.Hour

// DelistCandleGrace is added when fetching last klines so a date-only midnight
// halt still includes the rest of that session (often a few hours later).
const DelistCandleGrace = 24 * time.Hour

// HaltCandleEnd is the kline endTime for last prints: scheduled halt + one day,
// not after now.
func HaltCandleEnd(halt, now time.Time) time.Time {
	if halt.IsZero() {
		return time.Time{}
	}
	end := halt.UTC().Add(DelistCandleGrace)
	n := now.UTC()
	if !n.IsZero() && end.After(n) {
		return n
	}
	return end
}

// SpotDelistEntry is a scheduled spot-pair delisting on a venue.
type SpotDelistEntry struct {
	// Exchange is the venue id (binance, bybit, …).
	Exchange Exchange
	// Symbol is the native pair id (e.g. ADAUSDT).
	Symbol string
	// DelistTime is when trading ceases (UTC).
	DelistTime time.Time
	// AnnouncedAt is when the venue published the delist notice (UTC). Zero if unknown.
	AnnouncedAt time.Time
}

// DelistVisibleOnTradingList is true when the pair should stay on the default
// markets list with a Delist tag: delist is within the next month, or in the
// last 30 days (already-removed pairs stay visible).
func DelistVisibleOnTradingList(delist, now time.Time) bool {
	if delist.IsZero() {
		return false
	}
	t := delist.UTC()
	n := now.UTC()
	return !t.Before(n.Add(-DelistPastHorizon)) && !t.After(n.Add(DelistVisibleHorizon))
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
	// Get returns the stored entry (halt time + announcement time) when known.
	Get(exchange Exchange, symbol string) (SpotDelistEntry, bool)
	// List returns a snapshot of all entries for an exchange (stable symbol order).
	List(exchange Exchange) []SpotDelistEntry
}
