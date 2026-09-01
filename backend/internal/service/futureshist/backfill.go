package futureshist

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const backfillTimeout = 45 * time.Second

// Backfiller fills disconnect holes from each venue's own history.
// Binance and Bybit are never mixed.
type Backfiller struct {
	Book    *domain.LiquidationBook
	Hist    *Service
	Sources map[domain.Exchange]domain.LiquidationHistoryPort
	Seeds   []string
	// Watch subscribes a linear symbol (Bybit allLiquidation). Binance is all-market.
	Watch  func(symbol string)
	Logger *slog.Logger

	mu   sync.Mutex
	busy map[domain.Exchange]bool
}

// Schedule fetches history for closed gaps on one venue. Safe to call from SetLive.
func (b *Backfiller) Schedule(ex domain.Exchange) {
	if b == nil || b.Book == nil || ex == "" {
		return
	}
	src := b.Sources[ex]
	if src == nil {
		return
	}
	b.mu.Lock()
	if b.busy == nil {
		b.busy = map[domain.Exchange]bool{}
	}
	if b.busy[ex] {
		b.mu.Unlock()
		return
	}
	b.busy[ex] = true
	b.mu.Unlock()

	go func() {
		defer func() {
			b.mu.Lock()
			b.busy[ex] = false
			b.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), backfillTimeout)
		defer cancel()
		b.fillVenue(ctx, ex, src)
	}()
}

func (b *Backfiller) fillVenue(ctx context.Context, ex domain.Exchange, src domain.LiquidationHistoryPort) {
	gaps := b.Book.ClosedGaps(ex)
	if len(gaps) == 0 {
		return
	}
	syms := b.symbols(ex)
	var added int
	var last domain.LiquidationBackfillStats
	for _, g := range gaps {
		if ctx.Err() != nil {
			return
		}
		q, ok := domain.NormalizeHistoryQuery(domain.LiquidationHistoryQuery{
			Exchange: ex, Symbols: syms, From: g.From, To: g.To,
		})
		if !ok {
			continue
		}
		res, err := src.ListLiquidationHistory(ctx, q)
		if err != nil {
			b.log().Warn("liquidation history fetch failed", "exchange", ex, "err", err)
			continue
		}
		last = b.Book.ApplyHistory(ex, q.From, q.To, res.Events, res.CoveredFrom, res.CoveredTo)
		added += last.Added
		if b.Hist != nil {
			for _, e := range res.Events {
				if e.Exchange == "" {
					e.Exchange = ex
				}
				if e.Exchange != ex {
					continue
				}
				b.Hist.SaveLiquidation(ctx, e)
			}
		}
	}
	if added > 0 || last.Filled {
		b.log().Info("liquidation gap backfill",
			"exchange", ex,
			"added", added,
			"missing_seconds", last.MissingSeconds,
		)
	}
}

func (b *Backfiller) symbols(ex domain.Exchange) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = domain.NormalizeLiquidationSymbol(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if b.Book != nil {
		for _, s := range b.Book.WatchedSymbols(ex) {
			add(s)
		}
	}
	for _, s := range b.Seeds {
		add(s)
	}
	for _, s := range domain.DefaultFuturesHistorySymbols {
		add(s)
	}
	return out
}

const prepareTimeout = 12 * time.Second

// Prepare subscribes a coin (if it is not already on the live set) and fills
// [now-lookback, now] from that venue's own history. Overlapping live prints
// are ignored by identity. History failure does not unsubscribe.
func (b *Backfiller) Prepare(ctx context.Context, exchange, symbol string, lookback time.Duration) {
	if b == nil {
		return
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if symbol == "" || strings.EqualFold(symbol, "all") || symbol == domain.LiqAlertSymbolAll {
		return
	}
	if b.Watch != nil {
		b.Watch(symbol)
	}
	if b.Hist != nil {
		b.Hist.NoteSymbol(symbol)
	}
	if lookback <= 0 {
		return
	}
	if lookback > 24*time.Hour {
		lookback = 24 * time.Hour
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, prepareTimeout)
	defer cancel()
	now := time.Now().UTC()
	from := now.Add(-lookback)
	exs := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if parsed, err := domain.ParseLiquidationExchange(exchange); err == nil && parsed != "all" {
		exs = []domain.Exchange{domain.Exchange(parsed)}
	}
	var added int
	for _, ex := range exs {
		if ctx.Err() != nil {
			return
		}
		src := b.Sources[ex]
		if src == nil {
			continue
		}
		q, ok := domain.NormalizeHistoryQuery(domain.LiquidationHistoryQuery{
			Exchange: ex, Symbols: []string{symbol}, From: from, To: now,
		})
		if !ok {
			continue
		}
		res, err := src.ListLiquidationHistory(ctx, q)
		if err != nil {
			b.log().Warn("liquidation alert history fetch failed", "exchange", ex, "symbol", symbol, "err", err)
			continue
		}
		// Insert only — do not treat a one-coin fetch as venue-wide gap coverage.
		if b.Book != nil {
			st := b.Book.ApplyHistory(ex, q.From, q.To, res.Events, time.Time{}, time.Time{})
			added += st.Added
		}
		if b.Hist != nil {
			for _, e := range res.Events {
				if e.Exchange == "" {
					e.Exchange = ex
				}
				if e.Exchange != ex {
					continue
				}
				if domain.NormalizeLiquidationSymbol(e.Symbol) != symbol {
					continue
				}
				b.Hist.SaveLiquidation(ctx, e)
			}
		}
	}
	if added > 0 {
		b.log().Info("liquidation alert window filled",
			"symbol", symbol, "exchange", exchange,
			"lookback", lookback.String(), "added", added,
		)
	}
}

func (b *Backfiller) log() *slog.Logger {
	if b != nil && b.Logger != nil {
		return b.Logger
	}
	return slog.Default()
}
