package market

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// AggTradeWindowPort reads one short public aggregate-trade window.
// Binance USD-M is the only implementation. The window is [from, to).
type AggTradeWindowPort interface {
	ListAggTradeWindow(ctx context.Context, symbol string, from, to time.Time) ([]domain.TakerPrint, error)
}

// TapeBackfillResult is one symbol's step on the Binance walk.
// Done means this symbol already reaches the lookback.
// Stop means Binance asked the process to pause. Nothing from that window was stored.
type TapeBackfillResult struct {
	Advanced bool
	Done     bool
	Stop     bool
}

type tapeFlowStore interface {
	domain.FlowAtPricesStore
	EarliestFlow(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, bool, error)
	FlowScan(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, bool, error)
	SetFlowScan(ctx context.Context, exchange domain.Exchange, symbol string, until time.Time) error
	ReplaceFlowWindow(ctx context.Context, exchange domain.Exchange, symbol string, from, to time.Time, cells []domain.FlowAtPricesCell) error
}

// BackfillBinanceTape stores one hour of Binance USD-M prints that are older
// than the live edge and older than minutes already stored. Buy and sell land
// in taker minutes and in the price ladder. Bybit and OKX are not requested.
func (s *Service) BackfillBinanceTape(ctx context.Context, prints AggTradeWindowPort, symbol string, now time.Time) (TapeBackfillResult, error) {
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if s == nil || prints == nil || symbol == "" || s.takerStore == nil {
		return TapeBackfillResult{Done: true}, nil
	}
	flow, ok := s.flowHist.(tapeFlowStore)
	if !ok || flow == nil {
		return TapeBackfillResult{Done: true}, nil
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	target := now.Add(-domain.BinanceTapeBackfillLookback).Truncate(time.Minute)
	earliest, hasEarliest, err := flow.EarliestFlow(ctx, domain.ExchangeBinance, symbol)
	if err != nil {
		return TapeBackfillResult{}, err
	}
	scan, hasScan, err := flow.FlowScan(ctx, domain.ExchangeBinance, symbol)
	if err != nil {
		return TapeBackfillResult{}, err
	}
	end := domain.TapeBackfillEnd(now, earliest, scan, hasEarliest, hasScan)
	if !end.After(target) {
		return TapeBackfillResult{Done: true}, nil
	}
	window := domain.BinanceTapeWindow
	var span time.Duration
	for {
		if ctx.Err() != nil {
			return TapeBackfillResult{}, ctx.Err()
		}
		start := end.Add(-window)
		if start.Before(target) {
			start = target
		}
		if !end.After(start) {
			return TapeBackfillResult{Done: true}, nil
		}
		width := end.Sub(start)
		if span > 0 && width >= span {
			return TapeBackfillResult{}, domain.ErrTapeWindowFull
		}
		span = width
		batch, err := prints.ListAggTradeWindow(ctx, symbol, start, end)
		if errors.Is(err, domain.ErrTapeWindowFull) && window > time.Minute {
			window /= 4
			if window < time.Minute {
				window = time.Minute
			}
			continue
		}
		if errors.Is(err, domain.ErrRateLimited) {
			return TapeBackfillResult{Stop: true}, err
		}
		if err != nil {
			return TapeBackfillResult{}, err
		}
		kept := make([]domain.TakerPrint, 0, len(batch))
		for _, p := range batch {
			t := p.Time.UTC()
			if t.Before(start) || !t.Before(end) {
				continue
			}
			p.Time = t
			p.Exchange = domain.ExchangeBinance
			p.Symbol = symbol
			kept = append(kept, p)
		}
		buckets := domain.TakerBucketsFromPrints(kept)
		if len(buckets) > 0 {
			if _, err := s.takerStore.UpsertTakerBuckets(ctx, buckets); err != nil {
				return TapeBackfillResult{}, err
			}
		}
		cells, _ := domain.AccumulateFlowPrints(kept, time.Time{})
		if err := flow.ReplaceFlowWindow(ctx, domain.ExchangeBinance, symbol, start, end, cells); err != nil {
			return TapeBackfillResult{}, err
		}
		if err := flow.SetFlowScan(ctx, domain.ExchangeBinance, symbol, start); err != nil {
			return TapeBackfillResult{}, err
		}
		return TapeBackfillResult{Advanced: true, Done: !start.After(target)}, nil
	}
}

// BinanceTapeBackfill walks public Binance USD-M trades into stored minutes
// for the liquid list. Seeds come first. One pass stores one window per name,
// then waits. A Binance pause stops the pass and retries the unfinished window.
type BinanceTapeBackfill struct {
	Svc      *Service
	Prints   AggTradeWindowPort
	Symbols  func() []string
	Lookback time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (w *BinanceTapeBackfill) Start(ctx context.Context) {
	if w == nil || w.Svc == nil || w.Prints == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.sleep == nil {
		w.sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	for {
		if ctx.Err() != nil {
			return
		}
		paused, idle := w.Pass(ctx)
		wait := 2 * time.Second
		if paused {
			wait = 20 * time.Second
		} else if idle {
			wait = 15 * time.Minute
		}
		if err := w.sleep(ctx, wait); err != nil {
			w.Logger.Info("binance tape backfill stopped", "err", err)
			return
		}
	}
}

// Pass stores one window for each liquid name that still has history to fetch.
// The first result is true when Binance rate-limited the pass.
// The second is true when every name already reaches the lookback.
func (w *BinanceTapeBackfill) Pass(ctx context.Context) (paused, idle bool) {
	if w == nil || w.Svc == nil || w.Prints == nil {
		return false, true
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	symbols := domain.DefaultFuturesHistorySymbols
	if w.Symbols != nil {
		if got := w.Symbols(); len(got) > 0 {
			symbols = got
		}
	}
	symbols = domain.MergeLiquidSymbols(nil, symbols, domain.LiquidRecorderLimit)
	now := w.now().UTC()
	left := false
	var stored int
	for _, sym := range symbols {
		if ctx.Err() != nil {
			return false, false
		}
		res, err := w.Svc.BackfillBinanceTape(ctx, w.Prints, sym, now)
		if res.Stop || errors.Is(err, domain.ErrRateLimited) {
			w.Logger.Info("binance tape backfill paused", "symbol", sym, "stored", stored)
			return true, false
		}
		if err != nil {
			left = true
			w.Logger.Debug("binance tape backfill", "symbol", sym, "err", err)
			continue
		}
		if res.Advanced {
			stored++
		}
		if !res.Done {
			left = true
		}
	}
	if stored > 0 {
		w.Logger.Info("binance tape backfill", "symbols", stored)
	}
	return false, !left
}
