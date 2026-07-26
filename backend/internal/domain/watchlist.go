package domain

import (
	"context"
	"time"
)

// MaxWatchlistItems is the hard cap on items per client (enforced in store + service).
const MaxWatchlistItems = 200

// WatchlistItem is a saved market the user wants to track.
// JSON tags are for convenience in tests; HTTP handlers use dedicated DTOs.
type WatchlistItem struct {
	Exchange Exchange
	Symbol   string
	Note     string
	AddedAt  time.Time
}

// Watchlist is a client's list of items.
// Tenancy is opaque clientId only (no server auth in early versions) — clients
// must supply a non-empty unguessable id; empty/"default" is rejected.
type Watchlist struct {
	ClientID string
	Items    []WatchlistItem
	Updated  time.Time
}

// WatchlistPort persists watchlists. Implementations must be safe for concurrent use
// and enforce MaxWatchlistItems under the same lock as mutations.
type WatchlistPort interface {
	Get(ctx context.Context, clientID string) (*Watchlist, error)
	// Set replaces the entire list for clientID (must reject len(items) > MaxWatchlistItems).
	Set(ctx context.Context, clientID string, items []WatchlistItem) (*Watchlist, error)
	// Add upserts one item; must reject when adding a new symbol would exceed MaxWatchlistItems.
	Add(ctx context.Context, clientID string, item WatchlistItem) (*Watchlist, error)
	Remove(ctx context.Context, clientID string, exchange Exchange, symbol string) (*Watchlist, error)
}
