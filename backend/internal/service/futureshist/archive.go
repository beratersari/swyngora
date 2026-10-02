package futureshist

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const archiveSlack = 48 * time.Hour

// Archive pages public 1-hour open interest, funding, and long/short history
// into the futures store on startup. The symbol list is the liquid recorder
// universe (seeds first, up to LiquidRecorderLimit). Each series stops at the
// lookback that endpoint documents (see domain.PublicFuturesHistoryLookback),
// and never past FuturesHistoryMaxLookback. A later start skips a series that
// already reaches the back of that window. Liquidation prints are filled only
// for the public force-order window (about 7 days); older prints are the ones
// the websocket already stored.
//
// A Binance rate-limit pause stops the current pass. MaxPasses above 1 waits
// RetryPause and continues, skipping series that are already covered.
type Archive struct {
	Hist       *Service
	Liq        map[domain.Exchange]domain.LiquidationHistoryPort
	Lookback   time.Duration
	Logger     *slog.Logger
	MaxPasses  int
	RetryPause time.Duration
	now        func() time.Time
}

type snapshotClock interface {
	EarliestSnapshot(ctx context.Context, metric string, exchange domain.Exchange, symbol string) (time.Time, bool, error)
}

// Start runs the backfill. MaxPasses defaults to one. A rate-limited pass
// waits RetryPause and tries again until a pass finishes or the cap is hit.
func (a *Archive) Start(ctx context.Context) {
	if a == nil || a.Hist == nil || a.Hist.Store == nil {
		return
	}
	if a.Logger == nil {
		a.Logger = slog.Default()
	}
	if a.now == nil {
		a.now = time.Now
	}
	passes := a.MaxPasses
	if passes < 1 {
		passes = 1
	}
	pause := a.RetryPause
	if pause <= 0 {
		pause = 20 * time.Second
	}
	for i := 0; i < passes; i++ {
		if ctx.Err() != nil {
			return
		}
		if !a.pass(ctx) {
			return
		}
		if i+1 >= passes {
			return
		}
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// pass runs one backfill. It reports whether a venue rate limit stopped it.
func (a *Archive) pass(ctx context.Context) bool {
	now := a.now().UTC()
	from := a.retentionFrom(now)
	symbols := a.Hist.ArchiveSymbols()
	var oiN, fundN, lsN, liqN int
	for _, sym := range symbols {
		if ctx.Err() != nil {
			a.Logger.Info("futures archive stopped", "err", ctx.Err())
			return false
		}
		for _, ex := range domain.FuturesVenues() {
			if ctx.Err() != nil {
				return false
			}
			n, stop := a.fillOI(ctx, ex, sym, now)
			oiN += n
			if stop {
				a.finish(oiN, fundN, lsN, liqN, true)
				return true
			}
			n, stop = a.fillFunding(ctx, ex, sym, now)
			fundN += n
			if stop {
				a.finish(oiN, fundN, lsN, liqN, true)
				return true
			}
			n, stop = a.fillLongShort(ctx, ex, sym, now)
			lsN += n
			if stop {
				a.finish(oiN, fundN, lsN, liqN, true)
				return true
			}
		}
	}
	liqN = a.fillLiquidations(ctx, symbols, from, now)
	a.finish(oiN, fundN, lsN, liqN, false)
	return false
}

func (a *Archive) finish(oiN, fundN, lsN, liqN int, limited bool) {
	a.Logger.Info("futures archive backfill",
		"open_interest", oiN, "funding", fundN, "long_short", lsN, "liquidations", liqN, "rate_limited", limited)
}

func (a *Archive) retentionFrom(now time.Time) time.Time {
	look := a.Lookback
	if look <= 0 || look > domain.FuturesHistoryMaxLookback {
		look = domain.FuturesHistoryMaxLookback
	}
	return now.UTC().Add(-look)
}

// historyFrom is the start of the public window for this series.
// The second result is false when this archive does not request it.
func (a *Archive) historyFrom(ex domain.Exchange, metric string, now time.Time) (time.Time, bool) {
	public := domain.PublicFuturesHistoryLookback(ex, metric, now)
	if public <= 0 {
		return time.Time{}, false
	}
	from := a.retentionFrom(now)
	earliest := now.Add(-public)
	if from.Before(earliest) {
		from = earliest
	}
	return from, true
}

func (a *Archive) covered(ctx context.Context, metric string, ex domain.Exchange, symbol string, from time.Time) bool {
	clock, ok := a.Hist.Store.(snapshotClock)
	if !ok {
		return false
	}
	at, found, err := clock.EarliestSnapshot(ctx, metric, ex, symbol)
	if err != nil || !found {
		return false
	}
	return !at.After(from.Add(archiveSlack))
}

func (a *Archive) fillOI(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) (int, bool) {
	port, ok := a.Hist.OI[ex].(domain.OpenInterestHistoryPort)
	if !ok || port == nil {
		return 0, false
	}
	from, ok := a.historyFrom(ex, domain.FuturesMetricOpenInterest, now)
	if !ok {
		return 0, false
	}
	if a.covered(ctx, domain.FuturesMetricOpenInterest, ex, symbol, from) {
		return 0, false
	}
	pts, err := port.ListOpenInterestHistory(ctx, symbol, from, now)
	if stop := a.warn(ex, symbol, "open interest", err); stop {
		return 0, true
	}
	n := 0
	for _, p := range pts {
		if p.Time.Before(from) || p.Time.After(now) || p.Value <= 0 {
			continue
		}
		ok, err := a.Hist.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: symbol,
			SampledAt: domain.TruncateToBucket(p.Time, time.Hour),
			Contracts: p.Contracts, Value: p.Value,
		})
		if err != nil {
			continue
		}
		if ok {
			n++
		}
	}
	return n, false
}

func (a *Archive) fillFunding(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) (int, bool) {
	port := a.Hist.Funding[ex]
	if port == nil {
		return 0, false
	}
	from, ok := a.historyFrom(ex, domain.FuturesMetricFunding, now)
	if !ok {
		return 0, false
	}
	if a.covered(ctx, domain.FuturesMetricFunding, ex, symbol, from) {
		return 0, false
	}
	pts, err := port.ListFundingHistory(ctx, symbol, from, now)
	if stop := a.warn(ex, symbol, "funding", err); stop {
		return 0, true
	}
	n := 0
	for _, p := range pts {
		if p.Time.IsZero() || p.Time.Before(from) || p.Time.After(now) {
			continue
		}
		ok, err := a.Hist.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricFunding, Exchange: ex, Symbol: symbol,
			SampledAt: p.Time.UTC(), FundingRate: p.Rate, IntervalHours: 8,
		})
		if err != nil {
			continue
		}
		if ok {
			n++
		}
	}
	return n, false
}

func (a *Archive) fillLongShort(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) (int, bool) {
	port, ok := a.Hist.LS[ex].(domain.LongShortHistoryPort)
	if !ok || port == nil {
		return 0, false
	}
	from, ok := a.historyFrom(ex, domain.FuturesMetricLongShort, now)
	if !ok {
		return 0, false
	}
	if a.covered(ctx, domain.FuturesMetricLongShort, ex, symbol, from) {
		return 0, false
	}
	pts, err := port.ListLongShortHistory(ctx, symbol, from, now)
	if stop := a.warn(ex, symbol, "long short", err); stop {
		return 0, true
	}
	n := 0
	for _, p := range pts {
		if p.Time.Before(from) || p.Time.After(now) || p.LongShare <= 0 {
			continue
		}
		ok, err := a.Hist.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricLongShort, Exchange: ex, Symbol: symbol,
			SampledAt:       domain.TruncateToBucket(p.Time, time.Hour),
			LongShare:       p.LongShare,
			ShortShare:      p.ShortShare,
			Ratio:           p.Ratio,
			TopAccLongShare: p.TopAccLongShare,
			SizeLongShare:   p.SizeLongShare,
		})
		if err != nil {
			continue
		}
		if ok {
			n++
		}
	}
	return n, false
}

func (a *Archive) fillLiquidations(ctx context.Context, symbols []string, from, now time.Time) int {
	n := 0
	liqFrom := from
	if max := now.Add(-domain.LiquidationHistoryMaxLookback); liqFrom.Before(max) {
		liqFrom = max
	}
	for ex, src := range a.Liq {
		if src == nil || ctx.Err() != nil {
			continue
		}
		res, err := src.ListLiquidationHistory(ctx, domain.LiquidationHistoryQuery{
			Exchange:  ex,
			Symbols:   symbols,
			From:      liqFrom,
			To:        now,
			MaxWindow: domain.LiquidationHistoryMaxLookback,
		})
		if errors.Is(err, domain.ErrRateLimited) {
			a.Logger.Info("futures archive liquidation fill paused", "exchange", ex)
			return n
		}
		if err != nil {
			a.Logger.Debug("futures archive liquidation skip", "exchange", ex, "err", err)
			continue
		}
		for _, ev := range res.Events {
			ok, err := a.Hist.Store.InsertLiquidation(ctx, ev)
			if err == nil && ok {
				n++
			}
		}
	}
	return n
}

func (a *Archive) warn(ex domain.Exchange, symbol, what string, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrRateLimited) {
		a.Logger.Info("futures archive paused", "exchange", ex, "symbol", symbol, "series", what)
		return true
	}
	a.Logger.Debug("futures archive skip", "exchange", ex, "symbol", symbol, "series", what, "err", err)
	return false
}
