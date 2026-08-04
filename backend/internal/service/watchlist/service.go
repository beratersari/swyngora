package watchlist

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	maxItems       = domain.MaxWatchlistItems
	maxClientIDLen = 128
	maxNoteRunes   = 200
)

// Service orchestrates watchlist use cases including sharing and audit.
// Client IDs are opaque client-supplied tenancy keys (no server auth yet).
type Service struct {
	store domain.WatchlistPort
}

// New constructs a watchlist service.
func New(store domain.WatchlistPort) *Service {
	return &Service{store: store}
}

// resolveOwner returns the list owner id; empty owner means actor owns the list.
func resolveOwner(actor, owner string) string {
	if strings.TrimSpace(owner) == "" {
		return actor
	}
	return owner
}

// Get returns a watchlist the actor may view (own or shared as viewer/editor).
// ownerClientID empty → actor's own list.
func (s *Service) Get(ctx context.Context, actorClientID, ownerClientID string) (*domain.WatchlistAccess, error) {
	actor, err := normalizeClientID(actorClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	owner, err := normalizeClientID(resolveOwner(actor, ownerClientID))
	if err != nil {
		return nil, err
	}
	role, err := s.accessRole(ctx, actor, owner)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, fmt.Errorf("%w: no access to this watchlist", domain.ErrForbidden)
	}
	wl, err := s.store.Get(ctx, owner)
	if err != nil {
		return nil, err
	}
	if wl == nil {
		wl = &domain.Watchlist{ClientID: owner, Items: []domain.WatchlistItem{}, Updated: time.Now().UTC()}
	}
	return &domain.WatchlistAccess{Watchlist: *wl, OwnerClientID: owner, Role: role}, nil
}

// Replace sets the full list. Owner only (not editors).
// baseVersion is the version the client last loaded; baseItems (optional) enables accurate 3-way merge.
// Pass domain.WatchlistUnconditionalVersion to skip concurrency checks.
func (s *Service) Replace(ctx context.Context, actorClientID, ownerClientID string, items []domain.WatchlistItem, baseVersion int64, baseItems []domain.WatchlistItem) (*domain.WatchlistAccess, error) {
	actor, err := normalizeClientID(actorClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	owner, err := normalizeClientID(resolveOwner(actor, ownerClientID))
	if err != nil {
		return nil, err
	}
	if actor != owner {
		return nil, fmt.Errorf("%w: only the owner can replace the watchlist", domain.ErrForbidden)
	}
	norm, err := normalizeItems(items)
	if err != nil {
		return nil, err
	}
	var baseNorm []domain.WatchlistItem
	if baseItems != nil {
		baseNorm, err = normalizeItems(baseItems)
		if err != nil {
			return nil, err
		}
	}
	wl, err := s.store.Set(ctx, owner, norm, baseVersion)
	if err != nil {
		var mismatch *domain.WatchlistVersionMismatch
		if errors.As(err, &mismatch) && mismatch.Current != nil {
			merged, conflicts := domain.MergeWatchlistReplace(baseNorm, norm, mismatch.Current.Items)
			if len(conflicts) > 0 {
				return nil, &domain.WatchlistSyncConflict{
					BaseVersion: baseVersion, ServerVersion: mismatch.Current.Version,
					Server: *mismatch.Current, ClientProposed: norm, AutoMerged: merged, Conflicts: conflicts,
				}
			}
			// Auto-merge fully succeeded — write with current server version.
			wl, err = s.store.Set(ctx, owner, merged, mismatch.Current.Version)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	_ = s.store.AppendAudit(ctx, domain.WatchlistAuditEvent{
		ID: uuid.NewString(), OwnerClientID: owner, ActorClientID: actor,
		Action: domain.WatchlistAuditListReplaced, Detail: fmt.Sprintf("items=%d version=%d", len(wl.Items), wl.Version),
		CreatedAt: time.Now().UTC(),
	})
	return &domain.WatchlistAccess{Watchlist: *wl, OwnerClientID: owner, Role: domain.WatchlistRoleOwner}, nil
}

// Add appends or updates one item. Owner or editor.
// baseVersion is the version the client last loaded (or WatchlistUnconditionalVersion).
func (s *Service) Add(ctx context.Context, actorClientID, ownerClientID, exchange, symbol, note string, baseVersion int64) (*domain.WatchlistAccess, error) {
	actor, err := normalizeClientID(actorClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	owner, err := normalizeClientID(resolveOwner(actor, ownerClientID))
	if err != nil {
		return nil, err
	}
	role, err := s.accessRole(ctx, actor, owner)
	if err != nil {
		return nil, err
	}
	if role != domain.WatchlistRoleOwner && role != domain.WatchlistRoleEditor {
		return nil, fmt.Errorf("%w: view-only access; cannot add symbols", domain.ErrForbidden)
	}
	item, err := normalizeItem(domain.WatchlistItem{
		Exchange: domain.Exchange(exchange), Symbol: symbol, Note: note, AddedAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	wl, err := s.store.Add(ctx, owner, item, baseVersion)
	if err != nil {
		var mismatch *domain.WatchlistVersionMismatch
		if errors.As(err, &mismatch) && mismatch.Current != nil {
			noop, conf, auto, applyItem := domain.MergeWatchlistAdd(*mismatch.Current, item)
			if conf != nil {
				return nil, &domain.WatchlistSyncConflict{
					BaseVersion: baseVersion, ServerVersion: mismatch.Current.Version,
					Server: *mismatch.Current, Conflicts: []domain.WatchlistConflictItem{*conf},
					AutoMerged: mismatch.Current.Items,
				}
			}
			if noop != nil {
				return &domain.WatchlistAccess{Watchlist: *noop, OwnerClientID: owner, Role: role}, nil
			}
			if auto {
				wl, err = s.store.Add(ctx, owner, applyItem, mismatch.Current.Version)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	_ = s.store.AppendAudit(ctx, domain.WatchlistAuditEvent{
		ID: uuid.NewString(), OwnerClientID: owner, ActorClientID: actor,
		Action: domain.WatchlistAuditItemAdded, Exchange: string(item.Exchange), Symbol: item.Symbol,
		Detail: item.Note, CreatedAt: time.Now().UTC(),
	})
	return &domain.WatchlistAccess{Watchlist: *wl, OwnerClientID: owner, Role: role}, nil
}

// Remove deletes one item. Owner or editor.
func (s *Service) Remove(ctx context.Context, actorClientID, ownerClientID, exchange, symbol string, baseVersion int64) (*domain.WatchlistAccess, error) {
	actor, err := normalizeClientID(actorClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	owner, err := normalizeClientID(resolveOwner(actor, ownerClientID))
	if err != nil {
		return nil, err
	}
	role, err := s.accessRole(ctx, actor, owner)
	if err != nil {
		return nil, err
	}
	if role != domain.WatchlistRoleOwner && role != domain.WatchlistRoleEditor {
		return nil, fmt.Errorf("%w: view-only access; cannot remove symbols", domain.ErrForbidden)
	}
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return nil, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(rawEx)
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	wl, err := s.store.Remove(ctx, owner, ex, symbol, baseVersion)
	if err != nil {
		var mismatch *domain.WatchlistVersionMismatch
		if errors.As(err, &mismatch) && mismatch.Current != nil {
			noop, conf, _ := domain.MergeWatchlistRemove(*mismatch.Current, ex, symbol)
			if conf != nil {
				return nil, &domain.WatchlistSyncConflict{
					BaseVersion: baseVersion, ServerVersion: mismatch.Current.Version,
					Server: *mismatch.Current, Conflicts: []domain.WatchlistConflictItem{*conf},
					AutoMerged: mismatch.Current.Items,
				}
			}
			if noop != nil {
				return &domain.WatchlistAccess{Watchlist: *noop, OwnerClientID: owner, Role: role}, nil
			}
		}
		return nil, err
	}
	_ = s.store.AppendAudit(ctx, domain.WatchlistAuditEvent{
		ID: uuid.NewString(), OwnerClientID: owner, ActorClientID: actor,
		Action: domain.WatchlistAuditItemRemoved, Exchange: string(ex), Symbol: symbol,
		CreatedAt: time.Now().UTC(),
	})
	return &domain.WatchlistAccess{Watchlist: *wl, OwnerClientID: owner, Role: role}, nil
}

// Share grants viewer or editor access. Owner only. Same grantee cannot be shared twice.
func (s *Service) Share(ctx context.Context, ownerClientID, granteeClientID, role string) (*domain.WatchlistShare, error) {
	owner, err := normalizeClientID(ownerClientID)
	if err != nil {
		return nil, err
	}
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return nil, err
	}
	if owner == grantee {
		return nil, fmt.Errorf("%w: cannot share a watchlist with yourself", domain.ErrInvalidArgument)
	}
	r, err := domain.NormalizeWatchlistShareRole(role)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	n, err := s.store.CountSharesByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxWatchlistSharesPerOwner {
		return nil, fmt.Errorf("%w: max %d shares per watchlist", domain.ErrInvalidArgument, domain.MaxWatchlistSharesPerOwner)
	}
	now := time.Now().UTC()
	share, err := s.store.CreateShare(ctx, domain.WatchlistShare{
		OwnerClientID: owner, GranteeClientID: grantee, Role: r, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	_ = s.store.AppendAudit(ctx, domain.WatchlistAuditEvent{
		ID: uuid.NewString(), OwnerClientID: owner, ActorClientID: owner,
		Action: domain.WatchlistAuditShareGranted, Detail: fmt.Sprintf("grantee=%s role=%s", grantee, r),
		CreatedAt: now,
	})
	return share, nil
}

// UpdateShareRole changes an existing share's role. Owner only.
func (s *Service) UpdateShareRole(ctx context.Context, ownerClientID, granteeClientID, role string) (*domain.WatchlistShare, error) {
	owner, err := normalizeClientID(ownerClientID)
	if err != nil {
		return nil, err
	}
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return nil, err
	}
	r, err := domain.NormalizeWatchlistShareRole(role)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	now := time.Now().UTC()
	share, err := s.store.UpdateShareRole(ctx, owner, grantee, r, now)
	if err != nil {
		return nil, err
	}
	_ = s.store.AppendAudit(ctx, domain.WatchlistAuditEvent{
		ID: uuid.NewString(), OwnerClientID: owner, ActorClientID: owner,
		Action: domain.WatchlistAuditShareUpdated, Detail: fmt.Sprintf("grantee=%s role=%s", grantee, r),
		CreatedAt: now,
	})
	return share, nil
}

// RevokeShare removes access. Owner only.
func (s *Service) RevokeShare(ctx context.Context, ownerClientID, granteeClientID string) error {
	owner, err := normalizeClientID(ownerClientID)
	if err != nil {
		return err
	}
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return err
	}
	if s.store == nil {
		return fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	if err := s.store.DeleteShare(ctx, owner, grantee); err != nil {
		return err
	}
	_ = s.store.AppendAudit(ctx, domain.WatchlistAuditEvent{
		ID: uuid.NewString(), OwnerClientID: owner, ActorClientID: owner,
		Action: domain.WatchlistAuditShareRevoked, Detail: fmt.Sprintf("grantee=%s", grantee),
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

// ListShares returns shares for the owner's list.
func (s *Service) ListShares(ctx context.Context, ownerClientID string) ([]domain.WatchlistShare, error) {
	owner, err := normalizeClientID(ownerClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	return s.store.ListSharesByOwner(ctx, owner)
}

// ListSharedWithMe returns lists shared with the actor.
func (s *Service) ListSharedWithMe(ctx context.Context, actorClientID string) ([]domain.WatchlistShare, error) {
	actor, err := normalizeClientID(actorClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	return s.store.ListSharesForGrantee(ctx, actor)
}

// ListAudit returns change history for the owner's list (owner only).
func (s *Service) ListAudit(ctx context.Context, ownerClientID string, limit, offset int) ([]domain.WatchlistAuditEvent, error) {
	owner, err := normalizeClientID(ownerClientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListAudit(ctx, owner, limit, offset)
}

// accessRole returns owner/editor/viewer or empty if no access.
func (s *Service) accessRole(ctx context.Context, actor, owner string) (domain.WatchlistShareRole, error) {
	if actor == owner {
		return domain.WatchlistRoleOwner, nil
	}
	sh, err := s.store.GetShare(ctx, owner, actor)
	if err == domain.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return sh.Role, nil
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	if strings.EqualFold(id, "default") {
		return "", fmt.Errorf("%w: clientId must not be the shared name \"default\"", domain.ErrInvalidArgument)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: clientId has invalid characters", domain.ErrInvalidArgument)
	}
	return id, nil
}

func normalizeItems(items []domain.WatchlistItem) ([]domain.WatchlistItem, error) {
	if len(items) > maxItems {
		return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, maxItems)
	}
	seen := map[string]struct{}{}
	out := make([]domain.WatchlistItem, 0, len(items))
	for _, it := range items {
		n, err := normalizeItem(it)
		if err != nil {
			return nil, err
		}
		key := string(n.Exchange) + "|" + n.Symbol
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, n)
	}
	return out, nil
}

func normalizeItem(it domain.WatchlistItem) (domain.WatchlistItem, error) {
	rawEx := strings.TrimSpace(string(it.Exchange))
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return it, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, it.Symbol)
	if sym == "" {
		return it, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	note := strings.TrimSpace(it.Note)
	if utf8.RuneCountInString(note) > maxNoteRunes {
		runes := []rune(note)
		note = string(runes[:maxNoteRunes])
	}
	added := it.AddedAt
	if added.IsZero() {
		added = time.Now().UTC()
	}
	return domain.WatchlistItem{
		Exchange: ex, Symbol: sym, Note: note, AddedAt: added.UTC(),
	}, nil
}
