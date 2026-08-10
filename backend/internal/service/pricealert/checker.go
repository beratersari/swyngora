package pricealert

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// TickerFetcher loads last prices for alert evaluation (usually *market.Service).
type TickerFetcher interface {
	GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error)
}

// BookFetcher loads a live grouped book + analysis for imbalance/wall alerts.
type BookFetcher interface {
	GetSpotOrderBook(ctx context.Context, exchange, symbol, group string, levels int, rangePct float64) (*domain.OrderBook, error)
}

// Checker evaluates active price alerts on an interval.
// One-time alerts fire once then leave the active set.
// Repeating alerts fire on each edge into the condition zone and re-arm on the safe side.
type Checker struct {
	Alerts   *Service
	Market   TickerFetcher
	Books    BookFetcher
	Interval time.Duration
	Logger   *slog.Logger
	// now and sleep are injectable for tests.
	now   func() time.Time
	sleep func(context.Context, time.Duration) error
}

// Start runs the check loop until ctx is cancelled. Blocking — call in a goroutine.
func (c *Checker) Start(ctx context.Context) {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.now == nil {
		c.now = time.Now
	}
	if c.sleep == nil {
		c.sleep = sleepCtx
	}
	if c.Interval <= 0 {
		c.Interval = 30 * time.Second
	}
	// Immediate first pass so restarts re-evaluate quickly.
	c.RunOnce(ctx)
	for {
		if err := c.sleep(ctx, c.Interval); err != nil {
			c.Logger.Info("price alert checker stopped", "err", err)
			return
		}
		c.RunOnce(ctx)
	}
}

// RunOnce loads active alerts, fetches unique tickers, and applies price evaluation.
func (c *Checker) RunOnce(ctx context.Context) {
	if c.Alerts == nil {
		return
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.now == nil {
		c.now = time.Now
	}
	active, err := c.Alerts.ListActive(ctx)
	if err != nil {
		c.Logger.Error("list active alerts", "err", err)
		return
	}
	if len(active) == 0 {
		return
	}

	type key struct {
		ex  string
		sym string
	}
	type bookKey struct {
		ex  string
		sym string
		rng float64
	}
	// Dedupe ticker / book fetches across clients watching the same pair.
	priceGroups := map[key][]domain.PriceAlert{}
	bookGroups := map[bookKey][]domain.PriceAlert{}
	for _, a := range active {
		if domain.IsBookAlert(a.Kind) {
			rng := a.RangePct
			if rng <= 0 {
				rng = domain.DefaultOrderBookRangePct
			}
			k := bookKey{ex: string(a.Exchange), sym: a.Symbol, rng: rng}
			bookGroups[k] = append(bookGroups[k], a)
			continue
		}
		k := key{ex: string(a.Exchange), sym: a.Symbol}
		priceGroups[k] = append(priceGroups[k], a)
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for k, alerts := range priceGroups {
		k, alerts := k, alerts
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			c.evalPriceGroup(ctx, k.ex, k.sym, alerts)
		}()
	}
	for k, alerts := range bookGroups {
		k, alerts := k, alerts
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			c.evalBookGroup(ctx, k.ex, k.sym, k.rng, alerts)
		}()
	}
	wg.Wait()
}

func (c *Checker) evalPriceGroup(ctx context.Context, exchange, symbol string, alerts []domain.PriceAlert) {
	if c.Market == nil {
		return
	}
	tkr, err := c.Market.GetTicker24h(ctx, exchange, symbol)
	if err != nil || tkr == nil {
		c.Logger.Debug("ticker for alert failed", "exchange", exchange, "symbol", symbol, "err", err)
		return
	}
	last, err := strconv.ParseFloat(tkr.LastPrice, 64)
	if err != nil || last <= 0 {
		c.Logger.Debug("bad last price for alert", "symbol", symbol, "last", tkr.LastPrice)
		return
	}
	now := c.now().UTC()
	for _, a := range alerts {
		updated, fired, err := c.Alerts.ProcessPrice(ctx, a, last, now)
		if err != nil {
			c.Logger.Debug("process price alert failed", "id", a.ID, "err", err)
			continue
		}
		if !fired || updated == nil {
			continue
		}
		c.Logger.Info("price alert triggered",
			"id", updated.ID,
			"clientId", updated.ClientID,
			"exchange", updated.Exchange,
			"symbol", updated.Symbol,
			"condition", updated.Condition,
			"mode", updated.Mode,
			"target", updated.TargetPrice,
			"lastPrice", last,
		)
	}
}

func (c *Checker) evalBookGroup(ctx context.Context, exchange, symbol string, rangePct float64, alerts []domain.PriceAlert) {
	if c.Books == nil {
		c.Logger.Debug("order book alerts skipped (no book fetcher)", "exchange", exchange, "symbol", symbol)
		return
	}
	book, err := c.Books.GetSpotOrderBook(ctx, exchange, symbol, "", 5, rangePct)
	if err != nil || book == nil {
		c.Logger.Debug("order book for alert failed", "exchange", exchange, "symbol", symbol, "err", err)
		return
	}
	now := c.now().UTC()
	for _, a := range alerts {
		updated, fired, err := c.Alerts.ProcessBook(ctx, a, book.Analysis, now)
		if err != nil {
			c.Logger.Debug("process book alert failed", "id", a.ID, "err", err)
			continue
		}
		if !fired || updated == nil {
			continue
		}
		c.Logger.Info("order book alert triggered",
			"id", updated.ID,
			"clientId", updated.ClientID,
			"exchange", updated.Exchange,
			"symbol", updated.Symbol,
			"kind", updated.Kind,
			"condition", updated.Condition,
			"mode", updated.Mode,
			"target", updated.TargetPrice,
			"metric", updated.TriggeredPrice,
		)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
