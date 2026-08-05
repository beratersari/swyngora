package portfolio

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// OrderFiller watches last prices and fills open pending paper orders.
type OrderFiller struct {
	Portfolio *Service
	Market    PriceFetcher
	Interval  time.Duration
	Logger    *slog.Logger
	now       func() time.Time
	sleep     func(context.Context, time.Duration) error
}

// Start runs the fill loop until ctx is cancelled. Blocking — call in a goroutine.
func (f *OrderFiller) Start(ctx context.Context) {
	if f.Logger == nil {
		f.Logger = slog.Default()
	}
	if f.now == nil {
		f.now = time.Now
	}
	if f.sleep == nil {
		f.sleep = sleepCtx
	}
	if f.Interval <= 0 {
		f.Interval = 15 * time.Second
	}
	// Immediate first pass so restarts re-evaluate open orders quickly.
	f.RunOnce(ctx)
	for {
		if err := f.sleep(ctx, f.Interval); err != nil {
			f.Logger.Info("portfolio order filler stopped", "err", err)
			return
		}
		f.RunOnce(ctx)
	}
}

// RunOnce loads open pending orders, applies expiry/TIF, tries fills, and runs margin maintenance.
func (f *OrderFiller) RunOnce(ctx context.Context) {
	if f.Portfolio == nil || f.Market == nil {
		return
	}
	if f.Logger == nil {
		f.Logger = slog.Default()
	}
	if f.now == nil {
		f.now = time.Now
	}
	// Margin: limit fills, liquidation, stop-loss / take-profit (durable across restarts).
	if filled, liq, stopped, err := f.Portfolio.ProcessMarginMaintenance(ctx, f.now().UTC()); err != nil {
		f.Logger.Error("margin maintenance", "err", err)
	} else if filled > 0 || liq > 0 || stopped > 0 {
		f.Logger.Info("margin maintenance", "filled", filled, "liquidated", liq, "sl_tp", stopped)
	}

	open, err := f.Portfolio.ListAllOpenPendingOrders(ctx)
	if err != nil {
		f.Logger.Error("list open pending orders", "err", err)
		return
	}
	if len(open) == 0 {
		return
	}

	now := f.now().UTC()

	// Expire GTC orders first (no ticker required).
	remaining := make([]domain.PendingOrder, 0, len(open))
	for _, o := range open {
		tif := o.TimeInForce
		if tif == "" {
			tif = domain.TimeInForceGTC
		}
		if tif == domain.TimeInForceGTC && domain.PendingOrderExpired(o, now) {
			updated, ok, err := f.Portfolio.ProcessOpenOrder(ctx, o, 0, now, 0)
			if err != nil {
				f.Logger.Debug("expire pending order failed", "id", o.ID, "err", err)
				continue
			}
			if ok && updated != nil {
				f.Logger.Info("paper pending order canceled",
					"id", updated.ID, "reason", updated.CancelReason, "status", updated.Status)
			}
			continue
		}
		remaining = append(remaining, o)
	}
	if len(remaining) == 0 {
		return
	}

	type key struct {
		ex  string
		sym string
	}
	groups := map[key][]domain.PendingOrder{}
	for _, o := range remaining {
		k := key{ex: string(o.Exchange), sym: o.Symbol}
		groups[k] = append(groups[k], o)
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for k, orders := range groups {
		k, orders := k, orders
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			tkr, err := f.Market.GetTicker24h(ctx, k.ex, k.sym)
			if err != nil || tkr == nil {
				f.Logger.Debug("ticker for pending order failed", "exchange", k.ex, "symbol", k.sym, "err", err)
				return
			}
			last, err := strconv.ParseFloat(tkr.LastPrice, 64)
			if err != nil || last <= 0 {
				f.Logger.Debug("bad last price for pending order", "symbol", k.sym, "last", tkr.LastPrice)
				return
			}
			for _, o := range orders {
				updated, ok, err := f.Portfolio.ProcessOpenOrder(ctx, o, last, now, 0)
				if err != nil {
					f.Logger.Debug("process pending order failed", "id", o.ID, "err", err)
					continue
				}
				if !ok || updated == nil {
					continue
				}
				f.Logger.Info("paper pending order processed",
					"id", updated.ID,
					"clientId", updated.ClientID,
					"type", updated.Type,
					"tif", updated.TimeInForce,
					"status", updated.Status,
					"cancelReason", updated.CancelReason,
					"filledQuantity", updated.FilledQuantity,
					"remainingQuantity", updated.RemainingQuantity,
					"fillPrice", updated.FillPrice,
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
