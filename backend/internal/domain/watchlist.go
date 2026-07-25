package domain

import (
	"context"
	"time"
)

// WatchlistItem is a saved market the user wants to track.
type WatchlistItem struct {
	Exchange Exchange  `json:"exchange"`
	Symbol   string    `json:"symbol"`
	Note     string    `json:"note,omitempty"`
	AddedAt  time.Time `json:"addedAt"`
}

// Watchlist is a client's list of items (no auth — keyed by client id).
type Watchlist struct {
	ClientID string          `json:"clientId"`
	Items    []WatchlistItem `json:"items"`
	Updated  time.Time       `json:"updatedAt"`
}

// WatchlistPort persists watchlists. Implementations must be safe for concurrent use.
type WatchlistPort interface {
	Get(ctx context.Context, clientID string) (*Watchlist, error)
	// Set replaces the entire list for clientID.
	Set(ctx context.Context, clientID string, items []WatchlistItem) (*Watchlist, error)
	Add(ctx context.Context, clientID string, item WatchlistItem) (*Watchlist, error)
	Remove(ctx context.Context, clientID string, exchange Exchange, symbol string) (*Watchlist, error)
}
