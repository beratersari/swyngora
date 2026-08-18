package bookhist

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Service writes and reads durable spot order-book samples.
type Service struct {
	Store  domain.BookHistoryStore
	Books  map[domain.Exchange]domain.MarketDataPort
	Logger *slog.Logger
	Seeds  []string

	mu   sync.Mutex
	seen map[string]time.Time
}

// Note records a pair that users have asked about.
func (s *Service) Note(exchange, symbol string) {
	if s == nil {
		return
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		ex = domain.DefaultExchange
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return
	}
	s.mu.Lock()
	if s.seen == nil {
		s.seen = map[string]time.Time{}
	}
	s.seen[string(ex)+"|"+symbol] = time.Now().UTC()
	s.mu.Unlock()
}

// Jobs is the worker universe: seeds on each venue plus recently seen pairs.
func (s *Service) Jobs() []bookJob {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]struct{}{}
	out := make([]bookJob, 0, 32)
	add := func(ex domain.Exchange, symbol string) {
		symbol = domain.CrossVenueSymbol(ex, symbol)
		if symbol == "" || s.Books[ex] == nil {
			return
		}
		key := string(ex) + "|" + symbol
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, bookJob{ex: ex, symbol: symbol})
	}
	for _, seed := range s.Seeds {
		for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit} {
			add(ex, seed)
		}
	}
	for key := range s.seen {
		ex, symbol, ok := splitJobKey(key)
		if ok {
			add(ex, symbol)
		}
	}
	return out
}

type bookJob struct {
	ex     domain.Exchange
	symbol string
}

func splitJobKey(key string) (domain.Exchange, string, bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			ex := domain.ParseExchange(key[:i])
			if ex == "" {
				return "", "", false
			}
			return ex, key[i+1:], true
		}
	}
	return "", "", false
}

// SnapshotAt returns the stored book nearest to at.
func (s *Service) SnapshotAt(ctx context.Context, exchange, symbol string, at time.Time) (*domain.BookHistorySnapshot, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("%w: order book history not configured", domain.ErrUpstream)
	}
	ex, err := parseBookExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if at.IsZero() {
		return nil, fmt.Errorf("%w: time is required", domain.ErrInvalidArgument)
	}
	got, err := s.Store.NearestAt(ctx, string(ex), symbol, at.UTC())
	if err != nil {
		return nil, err
	}
	if got == nil {
		return nil, fmt.Errorf("%w: no stored order book for %s on %s", domain.ErrNotFound, symbol, ex)
	}
	marked, ok := domain.NearestBookSnapshot([]domain.BookHistorySnapshot{*got}, at.UTC(), domain.DefaultBookHistorySlack)
	if ok {
		return &marked, nil
	}
	got.Complete = false
	return got, nil
}

// List returns newest-first summaries (no ladders).
func (s *Service) List(ctx context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("%w: order book history not configured", domain.ErrUpstream)
	}
	ex, err := parseBookExchange(q.Exchange)
	if err != nil {
		return nil, err
	}
	q.Exchange = string(ex)
	q.Symbol = domain.NormalizeSymbol(ex, q.Symbol)
	if q.Symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	q.Limit = domain.ParseBookHistoryLimit(q.Limit)
	rows, err := s.Store.ListSnapshots(ctx, q)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i] = domain.StripBookLevels(rows[i])
	}
	return rows, nil
}

// Compare loads the nearest books at from and to and diffs liquidity.
func (s *Service) Compare(ctx context.Context, exchange, symbol string, from, to time.Time) (*domain.BookHistoryDiff, error) {
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("%w: from and to times are required", domain.ErrInvalidArgument)
	}
	if to.Before(from) {
		from, to = to, from
	}
	a, err := s.SnapshotAt(ctx, exchange, symbol, from)
	if err != nil {
		return nil, err
	}
	b, err := s.SnapshotAt(ctx, exchange, symbol, to)
	if err != nil {
		return nil, err
	}
	diff := domain.CompareBookHistory(*a, *b)
	return &diff, nil
}

// SaveSymbol fetches one venue independently and upserts a sample.
func (s *Service) SaveSymbol(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) (bool, error) {
	if s == nil || s.Store == nil {
		return false, nil
	}
	p := s.Books[ex]
	if p == nil {
		return false, nil
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return false, nil
	}
	raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: symbol, Limit: domain.MaxOrderBookRawLimit})
	if err != nil {
		return false, err
	}
	if raw == nil || (len(raw.Bids) == 0 && len(raw.Asks) == 0) {
		return false, nil
	}
	if raw.Symbol == "" {
		raw.Symbol = symbol
	}
	snap := domain.CaptureBookHistory(ex, *raw, now)
	return s.Store.InsertSnapshot(ctx, snap)
}

func parseBookExchange(raw string) (domain.Exchange, error) {
	ex := domain.ParseExchange(raw)
	if ex == "" {
		return "", fmt.Errorf("%w: exchange must be binance, coinbase, or bybit", domain.ErrInvalidArgument)
	}
	return ex, nil
}
