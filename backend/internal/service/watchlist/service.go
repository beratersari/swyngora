package watchlist

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	maxItems       = 200
	defaultClient  = "default"
	maxClientIDLen = 128
)

// Service orchestrates watchlist use cases.
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
	norm, err := normalizeItems(items)
	if err != nil {
		return nil, err
	}
	return s.store.Set(ctx, clientID, norm)
}

// Add appends or updates one item.
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
	// Enforce maxItems on Add (Replace already uses normalizeItems).
	cur, err := s.store.Get(ctx, clientID)
	if err != nil {
		return nil, err
	}
	isNew := true
	if cur != nil {
		for _, it := range cur.Items {
			if it.Exchange == item.Exchange && it.Symbol == item.Symbol {
				isNew = false
				break
			}
		}
		if isNew && len(cur.Items) >= maxItems {
			return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, maxItems)
		}
	}
	return s.store.Add(ctx, clientID, item)
}

// Remove deletes one item.
func (s *Service) Remove(ctx context.Context, clientID, exchange, symbol string) (*domain.Watchlist, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		return nil, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
	}
	symbol = normalizeSymbol(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return s.store.Remove(ctx, clientID, ex, symbol)
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = defaultClient
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	// allow simple ids only
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
	ex := domain.ParseExchange(string(it.Exchange))
	if ex == "" {
		// empty exchange → default
		if strings.TrimSpace(string(it.Exchange)) == "" {
			ex = domain.DefaultExchange
		} else {
			return it, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
	}
	sym := normalizeSymbol(ex, it.Symbol)
	if sym == "" {
		return it, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	note := strings.TrimSpace(it.Note)
	if len(note) > 200 {
		note = note[:200]
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

func normalizeSymbol(ex domain.Exchange, symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	if ex == domain.ExchangeCoinbase {
		return strings.ToUpper(symbol)
	}
	return strings.ToUpper(strings.ReplaceAll(symbol, "-", ""))
}
