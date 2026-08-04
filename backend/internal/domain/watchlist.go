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

// WatchlistItemKey is the unique identity of a symbol on a list.
func WatchlistItemKey(exchange Exchange, symbol string) string {
	return string(exchange) + "|" + symbol
}

// ItemKey returns the unique key for this item.
func (it WatchlistItem) ItemKey() string {
	return WatchlistItemKey(it.Exchange, it.Symbol)
}

// SameContent reports equal exchange, symbol, and note (ignores AddedAt).
func (it WatchlistItem) SameContent(other WatchlistItem) bool {
	return it.Exchange == other.Exchange && it.Symbol == other.Symbol && it.Note == other.Note
}

// Watchlist is a client's list of items.
// Tenancy is opaque clientId only (no server auth in early versions) — clients
// must supply a non-empty unguessable id; empty/"default" is rejected.
// ClientID is the list owner.
// Version is a monotonic revision for multi-device optimistic concurrency (starts at 0).
type Watchlist struct {
	ClientID string
	Items    []WatchlistItem
	Updated  time.Time
	Version  int64
}

// WatchlistUnconditionalVersion skips optimistic concurrency checks (imports, migrations).
const WatchlistUnconditionalVersion int64 = -1

// WatchlistConflictType classifies a per-symbol sync conflict.
type WatchlistConflictType string

const (
	// ConflictUpdateVsUpdate: both sides changed the same symbol (e.g. different notes).
	ConflictUpdateVsUpdate WatchlistConflictType = "update_vs_update"
	// ConflictDeleteVsUpdate: client deleted; server still has a (possibly changed) item.
	ConflictDeleteVsUpdate WatchlistConflictType = "delete_vs_update"
	// ConflictUpdateVsDelete: client updated; server deleted the symbol.
	ConflictUpdateVsDelete WatchlistConflictType = "update_vs_delete"
)

// WatchlistConflictItem is one symbol the user must resolve.
type WatchlistConflictItem struct {
	Exchange   Exchange
	Symbol     string
	Type       WatchlistConflictType
	ServerItem *WatchlistItem // nil if deleted on server
	ClientItem *WatchlistItem // nil if deleted on client
}

// WatchlistSyncConflict is returned when a write cannot be auto-merged.
// Unwraps to ErrConflict (HTTP 409).
type WatchlistSyncConflict struct {
	BaseVersion   int64
	ServerVersion int64
	Server        Watchlist
	// ClientProposed is the client's intended full list when known (replace); nil for single-ops.
	ClientProposed []WatchlistItem
	// AutoMerged is the non-conflicting merge preview (server + auto-accepted client changes).
	AutoMerged []WatchlistItem
	Conflicts  []WatchlistConflictItem
}

func (e *WatchlistSyncConflict) Error() string {
	return "conflict: watchlist version mismatch; resolve symbol conflicts"
}

func (e *WatchlistSyncConflict) Unwrap() error { return ErrConflict }

// WatchlistVersionMismatch is a low-level CAS failure from the store (no merge attempted yet).
type WatchlistVersionMismatch struct {
	Current *Watchlist
}

func (e *WatchlistVersionMismatch) Error() string {
	return "conflict: watchlist version mismatch"
}

func (e *WatchlistVersionMismatch) Unwrap() error { return ErrConflict }

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
//
// expectedVersion is optimistic concurrency: when >= 0 it must match the stored
// version or the method returns *WatchlistVersionMismatch without writing.
// Use WatchlistUnconditionalVersion (-1) to skip the check (imports/tests).
// Successful writes increment Version by 1.
type WatchlistPort interface {
	Get(ctx context.Context, clientID string) (*Watchlist, error)
	// Set replaces the entire list for clientID (must reject len(items) > MaxWatchlistItems).
	Set(ctx context.Context, clientID string, items []WatchlistItem, expectedVersion int64) (*Watchlist, error)
	// Add upserts one item; must reject when adding a new symbol would exceed MaxWatchlistItems.
	Add(ctx context.Context, clientID string, item WatchlistItem, expectedVersion int64) (*Watchlist, error)
	Remove(ctx context.Context, clientID string, exchange Exchange, symbol string, expectedVersion int64) (*Watchlist, error)

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
