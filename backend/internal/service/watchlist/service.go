package watchlist

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	maxItems       = domain.MaxWatchlistItems
	maxClientIDLen = 128
	maxNoteRunes   = 200
)

// Service orchestrates watchlist use cases.
// Client IDs are opaque client-supplied tenancy keys (no server auth yet).
// Empty clientId is rejected — there is no shared "default" bucket.
type Service struct {
	store domain.WatchlistPort
}

// New constructs a watchlist service.
func New(store domain.WatchlistPort) *Service {
	return &Service{store: store}
}

// Get returns the watchlist for a client (empty list if none).
func (s *Service) Get(ctx context.Context, clientID string) (*domain.Watchlist, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	wl, err := s.store.Get(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if wl == nil {
		return &domain.Watchlist{ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: time.Now().UTC()}, nil
	}
	return wl, nil
}

// Replace sets the full list.
func (s *Service) Replace(ctx context.Context, clientID string, items []domain.WatchlistItem) (*domain.Watchlist, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	norm, err := normalizeItems(items)
	if err != nil {
		return nil, err
	}
	return s.store.Set(ctx, clientID, norm)
}

// Add appends or updates one item. Max items is enforced in the store under lock.
func (s *Service) Add(ctx context.Context, clientID string, exchange, symbol, note string) (*domain.Watchlist, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	item, err := normalizeItem(domain.WatchlistItem{
		Exchange: domain.Exchange(exchange),
		Symbol:   symbol,
		Note:     note,
		AddedAt:  time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
	}
	return s.store.Add(ctx, clientID, item)
}

// Remove deletes one item.
func (s *Service) Remove(ctx context.Context, clientID, exchange, symbol string) (*domain.Watchlist, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: watchlist store not configured", domain.ErrUpstream)
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
	return s.store.Remove(ctx, clientID, ex, symbol)
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	// Reject the historical shared bucket name explicitly.
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
		// Truncate by runes, not bytes.
		runes := []rune(note)
		note = string(runes[:maxNoteRunes])
	}
	added := it.AddedAt
	if added.IsZero() {
		added = time.Now().UTC()
	}
	return domain.WatchlistItem{
		Exchange: ex,
		Symbol:   sym,
		Note:     note,
		AddedAt:  added.UTC(),
	}, nil
}
