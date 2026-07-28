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

// Checker evaluates active price alerts on an interval.
// Each alert is marked triggered at most once (store enforces active → triggered).
type Checker struct {
	Alerts   *Service
	Market   TickerFetcher
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

// RunOnce loads active alerts, fetches unique tickers, and marks matches triggered.
func (c *Checker) RunOnce(ctx context.Context) {
	if c.Alerts == nil || c.Market == nil {
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
	// Dedupe ticker fetches across clients watching the same pair.
	groups := map[key][]domain.PriceAlert{}
	for _, a := range active {
		k := key{ex: string(a.Exchange), sym: a.Symbol}
		groups[k] = append(groups[k], a)
	}

	// Bound concurrent upstream calls.
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for k, alerts := range groups {
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

			tkr, err := c.Market.GetTicker24h(ctx, k.ex, k.sym)
			if err != nil || tkr == nil {
				c.Logger.Debug("ticker for alert failed", "exchange", k.ex, "symbol", k.sym, "err", err)
				return
			}
			last, err := strconv.ParseFloat(tkr.LastPrice, 64)
			if err != nil || last <= 0 {
				c.Logger.Debug("bad last price for alert", "symbol", k.sym, "last", tkr.LastPrice)
				return
			}
			now := c.now().UTC()
			for _, a := range alerts {
				if !domain.AlertConditionMet(a.Condition, last, a.TargetPrice) {
					continue
				}
				if _, err := c.Alerts.MarkTriggered(ctx, a.ID, last, now); err != nil {
					// Already triggered or gone — expected under races; do not re-fire.
					c.Logger.Debug("mark triggered skipped", "id", a.ID, "err", err)
					continue
				}
				c.Logger.Info("price alert triggered",
					"id", a.ID,
					"clientId", a.ClientID,
					"exchange", a.Exchange,
					"symbol", a.Symbol,
					"condition", a.Condition,
					"target", a.TargetPrice,
					"lastPrice", last,
				)
			}
		}()
	}
	wg.Wait()
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