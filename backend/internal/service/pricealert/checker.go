package pricealert

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
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

// LiquidationAlertSource evaluates feed health and cascade grades.
type LiquidationAlertSource interface {
	GetLiquidationFeed(exchange string) domain.LiquidationFeed
	GetLiquidationCascade(ctx context.Context, exchange, symbol string) (*domain.CascadeReport, error)
	ScanLiquidationCascades(ctx context.Context, exchange string) (*domain.CascadeScan, error)
	ListLiquidationEvents(exchange, symbol string, since time.Time) []domain.LiquidationEvent
}

// AccountChecker reports whether a tenant is closed so workers can skip them.
type AccountChecker interface {
	IsClosed(ctx context.Context, clientID string) (bool, *domain.Account, error)
}

func tenantClosed(ctx context.Context, accounts AccountChecker, clientID string) bool {
	if accounts == nil || clientID == "" {
		return false
	}
	closed, _, err := accounts.IsClosed(ctx, clientID)
	return err == nil && closed
}

// Checker evaluates active price alerts on an interval.
// One-time alerts fire once then leave the active set.
// Repeating alerts fire on each edge into the condition zone and re-arm on the safe side.
// Closed accounts are skipped (docs/features/account-close.md — workers skip the tenant).
type Checker struct {
	Alerts       *Service
	Market       TickerFetcher
	Books        BookFetcher
	Liquidations LiquidationAlertSource
	Accounts     AccountChecker
	Interval     time.Duration
	Logger       *slog.Logger
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
	var feedAlerts []domain.PriceAlert
	notionalGroups := map[key][]domain.PriceAlert{}
	cascadeCoin := map[key][]domain.PriceAlert{}
	cascadeScan := map[string][]domain.PriceAlert{}
	for _, a := range active {
		if tenantClosed(ctx, c.Accounts, a.ClientID) {
			continue
		}
		switch {
		case domain.IsLiqFeedAlert(a.Kind):
			feedAlerts = append(feedAlerts, a)
		case domain.IsLiqNotionalAlert(a.Kind):
			k := key{ex: string(a.Exchange), sym: a.Symbol}
			notionalGroups[k] = append(notionalGroups[k], a)
		case domain.IsLiqCascadeAlert(a.Kind):
			if strings.EqualFold(a.Symbol, domain.LiqAlertSymbolAll) || strings.EqualFold(a.Symbol, "all") {
				ex := string(a.Exchange)
				if ex == "" {
					ex = "all"
				}
				cascadeScan[ex] = append(cascadeScan[ex], a)
			} else {
				k := key{ex: string(a.Exchange), sym: a.Symbol}
				cascadeCoin[k] = append(cascadeCoin[k], a)
			}
		case domain.IsBookAlert(a.Kind):
			rng := a.RangePct
			if rng <= 0 {
				rng = domain.DefaultOrderBookRangePct
			}
			k := bookKey{ex: string(a.Exchange), sym: a.Symbol, rng: rng}
			bookGroups[k] = append(bookGroups[k], a)
		default:
			k := key{ex: string(a.Exchange), sym: a.Symbol}
			priceGroups[k] = append(priceGroups[k], a)
		}
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
	c.evalFeedAlerts(ctx, feedAlerts)
	for k, alerts := range notionalGroups {
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
			c.evalNotionalGroup(ctx, k.ex, k.sym, alerts)
		}()
	}
	for k, alerts := range cascadeCoin {
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
			c.evalCascadeCoin(ctx, k.ex, k.sym, alerts)
		}()
	}
	for ex, alerts := range cascadeScan {
		ex, alerts := ex, alerts
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			c.evalCascadeScan(ctx, ex, alerts)
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
	if tkr.Halted {
		c.Logger.Debug("skip halted last print for alert", "exchange", exchange, "symbol", symbol)
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

func (c *Checker) evalFeedAlerts(ctx context.Context, alerts []domain.PriceAlert) {
	if c.Liquidations == nil || len(alerts) == 0 {
		return
	}
	now := c.now().UTC()
	feed := c.Liquidations.GetLiquidationFeed("all")
	for _, a := range alerts {
		met, metric, detail := domain.FeedAlertObservation(a, feed, now)
		ev := domain.EvaluateAlertState(a, met)
		extra := map[string]any{
			"exchange":         detail.Exchange,
			"unhealthySeconds": detail.UnhealthySeconds,
			"thresholdSeconds": detail.ThresholdSeconds,
			"missing":          detail.Missing,
			"live":             detail.Live,
		}
		if !detail.LastSeenAt.IsZero() {
			extra["lastSeenAt"] = detail.LastSeenAt.UTC().Format(time.RFC3339Nano)
		}
		if !detail.LastEventAt.IsZero() {
			extra["lastEventAt"] = detail.LastEventAt.UTC().Format(time.RFC3339Nano)
		}
		updated, fired, err := c.Alerts.ProcessObservationExtra(ctx, a, ev, metric, now, extra)
		if err != nil {
			c.Logger.Debug("process liquidation feed alert failed", "id", a.ID, "err", err)
			continue
		}
		if !fired || updated == nil {
			continue
		}
		c.Logger.Info("liquidation feed alert triggered",
			"id", updated.ID, "clientId", updated.ClientID,
			"exchange", detail.Exchange, "missing", detail.Missing,
			"unhealthySeconds", detail.UnhealthySeconds,
		)
	}
}

func (c *Checker) evalNotionalGroup(ctx context.Context, exchange, symbol string, alerts []domain.PriceAlert) {
	if c.Liquidations == nil || len(alerts) == 0 {
		return
	}
	now := c.now().UTC()
	since := now.Add(-time.Hour)
	for _, a := range alerts {
		if w := domain.LiqNotionalWindow(a); w > 0 && now.Add(-w).Before(since) {
			since = now.Add(-w)
		}
	}
	ev := c.Liquidations.ListLiquidationEvents(exchange, symbol, since)
	for _, a := range alerts {
		met, metric, detail := domain.NotionalAlertObservation(a, ev, now)
		st := domain.EvaluateAlertState(a, met)
		extra := map[string]any{
			"exchange":      detail.Exchange,
			"symbol":        detail.Symbol,
			"side":          detail.Side,
			"window":        detail.Window,
			"notional":      detail.Notional,
			"longNotional":  detail.LongNotional,
			"shortNotional": detail.ShortNotional,
			"threshold":     detail.Threshold,
			"count":         detail.Count,
		}
		updated, fired, err := c.Alerts.ProcessObservationExtra(ctx, a, st, metric, now, extra)
		if err != nil {
			c.Logger.Debug("process liquidation notional alert failed", "id", a.ID, "err", err)
			continue
		}
		if !fired || updated == nil {
			continue
		}
		c.Logger.Info("liquidation notional alert triggered",
			"id", updated.ID, "clientId", updated.ClientID,
			"exchange", detail.Exchange, "symbol", detail.Symbol,
			"side", detail.Side, "window", detail.Window, "notional", detail.Notional,
		)
	}
}

func (c *Checker) evalCascadeCoin(ctx context.Context, exchange, symbol string, alerts []domain.PriceAlert) {
	if c.Liquidations == nil {
		return
	}
	rep, err := c.Liquidations.GetLiquidationCascade(ctx, exchange, symbol)
	if err != nil {
		c.Logger.Debug("cascade for alert failed", "exchange", exchange, "symbol", symbol, "err", err)
		return
	}
	now := c.now().UTC()
	for _, a := range alerts {
		met, metric, detail := domain.CascadeAlertObservation(a, rep)
		c.finishCascade(ctx, a, met, metric, detail, now)
	}
}

func (c *Checker) evalCascadeScan(ctx context.Context, exchange string, alerts []domain.PriceAlert) {
	if c.Liquidations == nil {
		return
	}
	scan, err := c.Liquidations.ScanLiquidationCascades(ctx, exchange)
	if err != nil {
		c.Logger.Debug("cascade scan for alert failed", "exchange", exchange, "err", err)
		return
	}
	now := c.now().UTC()
	for _, a := range alerts {
		met, metric, detail := domain.CascadeScanAlertObservation(a, scan)
		c.finishCascade(ctx, a, met, metric, detail, now)
	}
}

func (c *Checker) finishCascade(ctx context.Context, a domain.PriceAlert, met bool, metric float64, detail domain.LiqCascadeAlertDetail, now time.Time) {
	ev := domain.EvaluateAlertState(a, met)
	extra := map[string]any{
		"exchange": detail.Exchange,
		"symbol":   detail.Symbol,
		"grade":    detail.Grade,
		"side":     detail.Side,
		"score":    detail.Score,
		"hottest":  detail.Hottest,
		"summary":  detail.Summary,
		"both":     detail.Both,
	}
	updated, fired, err := c.Alerts.ProcessObservationExtra(ctx, a, ev, metric, now, extra)
	if err != nil {
		c.Logger.Debug("process liquidation cascade alert failed", "id", a.ID, "err", err)
		return
	}
	if !fired || updated == nil {
		return
	}
	c.Logger.Info("liquidation cascade alert triggered",
		"id", updated.ID, "clientId", updated.ClientID,
		"exchange", detail.Exchange, "symbol", detail.Symbol,
		"grade", detail.Grade, "side", detail.Side, "score", detail.Score,
	)
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
