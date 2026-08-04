package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MaxWatchlistItems is the hard cap on items per client (enforced in store + service).
const MaxWatchlistItems = 200

// MaxWatchlistSharesPerOwner caps how many grantees one owner can share with.
const MaxWatchlistSharesPerOwner = 50

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
// ClientID is the list owner.
type Watchlist struct {
	ClientID string
	Items    []WatchlistItem
	Updated  time.Time
}

// WatchlistShareRole is viewer (read-only) or editor (add/remove items only).
type WatchlistShareRole string

const (
	WatchlistRoleViewer WatchlistShareRole = "viewer"
	WatchlistRoleEditor WatchlistShareRole = "editor"
	// WatchlistRoleOwner is returned in access views; never stored as a share row.
	WatchlistRoleOwner WatchlistShareRole = "owner"
)

// WatchlistShare grants another client access to an owner's watchlist.
type WatchlistShare struct {
	OwnerClientID   string
	GranteeClientID string
	Role            WatchlistShareRole
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// WatchlistAuditAction names a mutation recorded for a list.
type WatchlistAuditAction string

const (
	WatchlistAuditShareGranted WatchlistAuditAction = "share_granted"
	WatchlistAuditShareUpdated WatchlistAuditAction = "share_updated"
	WatchlistAuditShareRevoked WatchlistAuditAction = "share_revoked"
	WatchlistAuditItemAdded    WatchlistAuditAction = "item_added"
	WatchlistAuditItemRemoved  WatchlistAuditAction = "item_removed"
	WatchlistAuditListReplaced WatchlistAuditAction = "list_replaced"
)

// WatchlistAuditEvent records who changed what and when.
type WatchlistAuditEvent struct {
	ID            string
	OwnerClientID string
	ActorClientID string
	Action        WatchlistAuditAction
	Exchange      string
	Symbol        string
	Detail        string // free-form (role, note, item counts, etc.)
	CreatedAt     time.Time
}

// WatchlistAccess is a list plus the caller's role on it.
type WatchlistAccess struct {
	Watchlist
	OwnerClientID string
	Role          WatchlistShareRole
}

// IsValidWatchlistShareRole reports viewer|editor.
func IsValidWatchlistShareRole(s string) bool {
	switch WatchlistShareRole(strings.ToLower(strings.TrimSpace(s))) {
	case WatchlistRoleViewer, WatchlistRoleEditor:
		return true
	default:
		return false
	}
}

// NormalizeWatchlistShareRole parses viewer|editor.
func NormalizeWatchlistShareRole(s string) (WatchlistShareRole, error) {
	r := WatchlistShareRole(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidWatchlistShareRole(string(r)) {
		return "", fmt.Errorf("%w: role must be viewer or editor", ErrInvalidArgument)
	}
	return r, nil
}

// WatchlistPort persists watchlists, shares, and audit history.
// Implementations must be safe for concurrent use and enforce MaxWatchlistItems
// under the same lock as mutations.
type WatchlistPort interface {
	Get(ctx context.Context, clientID string) (*Watchlist, error)
	// Set replaces the entire list for clientID (must reject len(items) > MaxWatchlistItems).
	Set(ctx context.Context, clientID string, items []WatchlistItem) (*Watchlist, error)
	// Add upserts one item; must reject when adding a new symbol would exceed MaxWatchlistItems.
	Add(ctx context.Context, clientID string, item WatchlistItem) (*Watchlist, error)
	Remove(ctx context.Context, clientID string, exchange Exchange, symbol string) (*Watchlist, error)

	// Shares: ownerClientID is the list owner; grantee is the shared-with client.
	// CreateShare must fail if the owner+grantee pair already exists (no double-share).
	// UpdateShareRole changes role on an existing share only.
	CreateShare(ctx context.Context, share WatchlistShare) (*WatchlistShare, error)
	UpdateShareRole(ctx context.Context, ownerClientID, granteeClientID string, role WatchlistShareRole, at time.Time) (*WatchlistShare, error)
	GetShare(ctx context.Context, ownerClientID, granteeClientID string) (*WatchlistShare, error)
	ListSharesByOwner(ctx context.Context, ownerClientID string) ([]WatchlistShare, error)
	ListSharesForGrantee(ctx context.Context, granteeClientID string) ([]WatchlistShare, error)
	DeleteShare(ctx context.Context, ownerClientID, granteeClientID string) error
	CountSharesByOwner(ctx context.Context, ownerClientID string) (int, error)

	// Audit
	AppendAudit(ctx context.Context, ev WatchlistAuditEvent) error
	ListAudit(ctx context.Context, ownerClientID string, limit, offset int) ([]WatchlistAuditEvent, error)
}
