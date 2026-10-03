package bookhist

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// bookFlashDepth is the top of book sampled between minute snapshots.
// Binance charges 5 weight for 100 levels and 50 weight for 1,000.
const bookFlashDepth = 100

// Service writes and reads durable spot order-book samples.
type Service struct {
	Store  domain.BookHistoryStore
	Books  map[domain.Exchange]domain.MarketDataPort
	Logger *slog.Logger
	Seeds  []string

	mu     sync.Mutex
	seen   map[string]time.Time
	liquid []string

	watch domain.BookFlashWatch
	moves domain.BookMoveWatch
}

// binanceLiquidGap is how often Binance books outside the seed list are snapshotted.
// Flashes stay on seeds. Wide names use the top 100 levels.
const binanceLiquidGap = 2 * time.Minute

// SetLiquid adds USDT pairs whose spot books are snapshotted beside the seeds.
// Flashes stay on seeds and recently opened pairs. Binance wide names are sampled
// at the top 100 levels, and less often than Bybit and Coinbase.
func (s *Service) SetLiquid(symbols []string) {
	if s == nil {
		return
	}
	cp := append([]string{}, symbols...)
	s.mu.Lock()
	s.liquid = cp
	s.mu.Unlock()
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

// snapJob is one minute snapshot. depth 0 keeps the full ladder.
// minGap slows Binance samples that are outside the seed list.
type snapJob struct {
	ex     domain.Exchange
	symbol string
	depth  int
	minGap time.Duration
}

// SnapshotJobs is the minute-snapshot universe. Seeds and recently opened
// pairs keep a full ladder. Extra liquid names use the top 100 levels.
func (s *Service) SnapshotJobs() []snapJob {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]struct{}{}
	out := make([]snapJob, 0, 48)
	add := func(ex domain.Exchange, symbol string, depth int, gap time.Duration) {
		symbol = domain.CrossVenueSymbol(ex, symbol)
		if symbol == "" || s.Books[ex] == nil {
			return
		}
		key := string(ex) + "|" + symbol
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, snapJob{ex: ex, symbol: symbol, depth: depth, minGap: gap})
	}
	venues := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit}
	for _, seed := range s.Seeds {
		for _, ex := range venues {
			add(ex, seed, 0, 0)
		}
	}
	for key := range s.seen {
		ex, symbol, ok := splitJobKey(key)
		if ok {
			add(ex, symbol, 0, 0)
		}
	}
	for _, sym := range s.liquid {
		for _, ex := range venues {
			gap := time.Duration(0)
			if ex == domain.ExchangeBinance {
				gap = binanceLiquidGap
			}
			add(ex, sym, bookFlashDepth, gap)
		}
	}
	return out
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
		s.attachFlashes(ctx, &marked)
		s.attachMoves(ctx, &marked)
		return &marked, nil
	}
	got.Complete = false
	s.attachFlashes(ctx, got)
	s.attachMoves(ctx, got)
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

// ListLadders returns snapshots in the range with bid/ask levels kept.
func (s *Service) ListLadders(ctx context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
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
	q.Limit = domain.ClampBookLadderLimit(q.Limit)
	return s.Store.ListSnapshots(ctx, q)
}

// ListFlashes returns brief walls that overlap the window.
func (s *Service) ListFlashes(ctx context.Context, exchange, symbol string, from, to time.Time) ([]domain.BookHistoryFlash, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("%w: order book history not configured", domain.ErrUpstream)
	}
	return s.Store.ListFlashes(ctx, exchange, symbol, from, to)
}

// ListMoves returns non-wall size changes that overlap the window.
func (s *Service) ListMoves(ctx context.Context, exchange, symbol string, from, to time.Time) ([]domain.BookHistoryMove, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("%w: order book history not configured", domain.ErrUpstream)
	}
	return s.Store.ListMoves(ctx, exchange, symbol, from, to)
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

type bookEarliest interface {
	EarliestBook(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, bool, error)
}

// RecordingSince is the oldest stored spot-book sample per venue.
func (s *Service) RecordingSince(ctx context.Context, symbol string, now time.Time) ([]domain.RecordedSeries, error) {
	if s == nil || s.Store == nil {
		return nil, nil
	}
	ear, ok := s.Store.(bookEarliest)
	if !ok {
		return nil, nil
	}
	var out []domain.RecordedSeries
	for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit} {
		sym := domain.NormalizeSymbol(ex, symbol)
		if sym == "" {
			continue
		}
		since, found, err := ear.EarliestBook(ctx, ex, sym)
		if err != nil {
			return nil, err
		}
		if !found || since.IsZero() {
			continue
		}
		out = append(out, domain.RecordedSeries{
			Kind:     domain.RecordedBook,
			Exchange: ex,
			Since:    since.UTC(),
		})
	}
	return out, nil
}

type bookSpanBatch interface {
	EarliestBookSpans(context.Context, []string) ([]domain.StoredSince, error)
}

// RecordingSpans is the oldest spot-book sample for each logical USDT symbol.
func (s *Service) RecordingSpans(ctx context.Context, symbols []string, now time.Time) (map[string][]domain.RecordedSeries, error) {
	symbols = domain.SpanQuerySymbols(symbols)
	out := map[string][]domain.RecordedSeries{}
	if s == nil || s.Store == nil || len(symbols) == 0 {
		return out, nil
	}
	batch, ok := s.Store.(bookSpanBatch)
	if !ok {
		for _, sym := range symbols {
			rows, err := s.RecordingSince(ctx, sym, now)
			if err != nil {
				return nil, err
			}
			if len(rows) > 0 {
				out[sym] = rows
			}
		}
		return out, nil
	}
	stored := make([]string, 0, len(symbols)*3)
	back := map[string]string{}
	for _, logical := range symbols {
		for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit} {
			form := domain.CrossVenueSymbol(ex, logical)
			if form == "" {
				continue
			}
			stored = append(stored, form)
			back[string(ex)+"|"+form] = logical
		}
	}
	rows, err := batch.EarliestBookSpans(ctx, stored)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		logical := back[string(row.Exchange)+"|"+row.Symbol]
		if logical == "" || row.Since.IsZero() {
			continue
		}
		out[logical] = append(out[logical], domain.RecordedSeries{
			Kind: domain.RecordedBook, Exchange: row.Exchange, Since: row.Since.UTC(),
		})
	}
	return out, nil
}

// SaveSymbol fetches one venue independently and upserts a full ladder.
func (s *Service) SaveSymbol(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) (bool, error) {
	return s.SaveSymbolDepth(ctx, ex, symbol, now, 0)
}

// SaveSymbolDepth is SaveSymbol with a level cap. depth <= 0 uses the full ladder.
func (s *Service) SaveSymbolDepth(ctx context.Context, ex domain.Exchange, symbol string, now time.Time, depth int) (bool, error) {
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
	limit := domain.MaxOrderBookRawLimit
	if depth > 0 && depth < limit {
		limit = depth
	}
	raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: symbol, Limit: limit})
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
	s.keepFlashes(ctx, ex, symbol, now, snap.Walls, true)
	s.keepMoves(ctx, ex, symbol, now, domain.BookMoveSamples(*raw), true)
	return s.Store.InsertSnapshot(ctx, snap)
}

// SaveFlashes reads the live book and stores walls that vanish before the next minute sample.
func (s *Service) SaveFlashes(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) (int, error) {
	if s == nil || s.Store == nil {
		return 0, nil
	}
	p := s.Books[ex]
	if p == nil {
		return 0, nil
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return 0, nil
	}
	raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: symbol, Limit: bookFlashDepth})
	if err != nil {
		return 0, err
	}
	if raw == nil {
		return 0, nil
	}
	n := s.keepFlashes(ctx, ex, symbol, now, domain.BookWalls(*raw), false)
	n += s.keepMoves(ctx, ex, symbol, now, domain.BookMoveSamples(*raw), false)
	return n, nil
}

func (s *Service) keepFlashes(ctx context.Context, ex domain.Exchange, symbol string, now time.Time, walls []domain.BookHistoryWall, minute bool) int {
	if s == nil || s.Store == nil {
		return 0
	}
	s.mu.Lock()
	done := s.watch.Observe(ex, symbol, now, walls, minute)
	s.mu.Unlock()
	n := 0
	for _, f := range done {
		ok, err := s.Store.InsertFlash(ctx, f)
		if err == nil && ok {
			n++
		}
	}
	return n
}

func (s *Service) attachFlashes(ctx context.Context, snap *domain.BookHistorySnapshot) {
	if s == nil || s.Store == nil || snap == nil || snap.SampledAt.IsZero() {
		return
	}
	from := snap.SampledAt.UTC()
	rows, err := s.Store.ListFlashes(ctx, string(snap.Exchange), snap.Symbol, from, from.Add(domain.DefaultBookHistoryBucket))
	if err != nil || len(rows) == 0 {
		return
	}
	domain.AttachBookFlashes(snap, rows)
}

func (s *Service) keepMoves(ctx context.Context, ex domain.Exchange, symbol string, now time.Time, levels []domain.BookMoveSample, minute bool) int {
	if s == nil || s.Store == nil {
		return 0
	}
	s.mu.Lock()
	done := s.moves.Observe(ex, symbol, now, levels, minute)
	s.mu.Unlock()
	n := 0
	for _, mv := range done {
		ok, err := s.Store.InsertMove(ctx, mv)
		if err == nil && ok {
			n++
		}
	}
	return n
}

func (s *Service) attachMoves(ctx context.Context, snap *domain.BookHistorySnapshot) {
	if s == nil || s.Store == nil || snap == nil || snap.SampledAt.IsZero() {
		return
	}
	from := snap.SampledAt.UTC()
	rows, err := s.Store.ListMoves(ctx, string(snap.Exchange), snap.Symbol, from, from.Add(domain.DefaultBookHistoryBucket))
	if err != nil || len(rows) == 0 {
		return
	}
	domain.AttachBookMoves(snap, rows)
}

func parseBookExchange(raw string) (domain.Exchange, error) {
	ex := domain.ParseExchange(raw)
	if ex == "" {
		return "", fmt.Errorf("%w: exchange must be binance, coinbase, or bybit", domain.ErrInvalidArgument)
	}
	return ex, nil
}
