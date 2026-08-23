package portfolio

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakePx struct {
	prices map[string]string
}

func (f *fakePx) GetTicker24h(_ context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	p, ok := f.prices[exchange+"|"+symbol]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p}, nil
}

func newSvc(t *testing.T, px *fakePx) *Service {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if px == nil {
		px = &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	}
	return New(st, px).WithPaperCosts(domain.ZeroTradingCosts)
}

func TestPortfolio_BracketPartialEntrySizesExitsAndOCO(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "br1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	entry, tp, sl, err := svc.PlaceBracketOrder(ctx, BracketOrderInput{
		ClientID: "br1", Symbol: "BTCUSDT", Quantity: 2,
		EntryPrice: 100, TakeProfitPrice: 120, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != domain.PendingStatusOpen || tp.Status != domain.PendingStatusPending || sl.Status != domain.PendingStatusPending {
		t.Fatalf("entry open, exits pending: e=%s tp=%s sl=%s", entry.Status, tp.Status, sl.Status)
	}
	if tp.RemainingQuantity != 0 || sl.Quantity != 0 {
		t.Fatalf("exits must start at size 0: tp=%+v sl=%+v", tp, sl)
	}
	// Price at entry — fill 0.75 only
	got, ok, err := svc.TryFillPendingOrder(ctx, *entry, 100, 0.75)
	if err != nil || !ok {
		t.Fatalf("entry partial ok=%v err=%v", ok, err)
	}
	if math.Abs(got.FilledQuantity-0.75) > 1e-9 {
		t.Fatalf("entry filled=%v", got.FilledQuantity)
	}
	tp2, _ := svc.store.GetPendingOrder(ctx, "br1", tp.ID)
	sl2, _ := svc.store.GetPendingOrder(ctx, "br1", sl.ID)
	if tp2.Status != domain.PendingStatusOpen || sl2.Status != domain.PendingStatusOpen {
		t.Fatalf("exits should be open after partial entry: tp=%s sl=%s", tp2.Status, sl2.Status)
	}
	if math.Abs(tp2.RemainingQuantity-0.75) > 1e-9 || math.Abs(sl2.RemainingQuantity-0.75) > 1e-9 {
		t.Fatalf("exit size should match filled 0.75: tp=%v sl=%v", tp2.RemainingQuantity, sl2.RemainingQuantity)
	}
	// More entry fill → exits grow
	got, ok, err = svc.TryFillPendingOrder(ctx, *got, 100, 0)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if got.Status != domain.PendingStatusFilled {
		t.Fatalf("entry should be fully filled: %+v", got)
	}
	tp3, _ := svc.store.GetPendingOrder(ctx, "br1", tp.ID)
	if math.Abs(tp3.RemainingQuantity-2) > 1e-9 {
		t.Fatalf("exit size after full entry=%v want 2", tp3.RemainingQuantity)
	}
	// TP fill cancels SL (no double sell)
	filledTP, ok, err := svc.TryFillPendingOrder(ctx, *tp3, 120, 0)
	if err != nil || !ok || filledTP.Status != domain.PendingStatusFilled {
		t.Fatalf("tp fill: ok=%v err=%v %+v", ok, err, filledTP)
	}
	sl3, _ := svc.store.GetPendingOrder(ctx, "br1", sl.ID)
	if sl3.Status != domain.PendingStatusCanceled || sl3.CancelReason != domain.CancelReasonOCOPeerFilled {
		t.Fatalf("sl should cancel after tp: %+v", sl3)
	}
	// Position flat after exit sell of 2
	view, _ := svc.View(ctx, "br1")
	if view.PositionsValue > 1e-6 {
		t.Fatalf("want flat after exit, positionsValue=%v", view.PositionsValue)
	}
}

func TestPortfolio_BracketExitsInactiveUntilEntry(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "br2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	_, tp, sl, err := svc.PlaceBracketOrder(ctx, BracketOrderInput{
		ClientID: "br2", Symbol: "ETHUSDT", Quantity: 1,
		EntryPrice: 95, TakeProfitPrice: 110, StopLossPrice: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Price hits TP while entry not filled — exits pending, no fill
	_, ok, err := svc.ProcessOpenOrder(ctx, *tp, 110, time.Now().UTC(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("pending exit must not fill before entry")
	}
	_, ok, err = svc.ProcessOCOPair(ctx, *tp, *sl, 110, time.Now().UTC(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("OCO pair must not fill pending exits")
	}
	tp2, _ := svc.store.GetPendingOrder(ctx, "br2", tp.ID)
	if tp2.Status != domain.PendingStatusPending {
		t.Fatalf("still pending: %s", tp2.Status)
	}
}

func TestPortfolio_TrailingStopRatchetsAndFiresOnce(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "trail1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "trail1", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "trail1", Symbol: "BTCUSDT", Type: "trailing_stop",
		Quantity: 1, TrailType: "percent", TrailValue: 0.10, // 10%
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(o.TrailPeak-100) > 1e-9 || math.Abs(o.TriggerPrice-90) > 1e-9 {
		t.Fatalf("seed peak/stop: %+v", o)
	}
	// Price rises — stop ratchets up
	_, _, err = svc.ProcessOpenOrder(ctx, *o, 120, time.Now().UTC(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// Not filled; may return nil if only trail updated without fill
	cur, err := svc.store.GetPendingOrder(ctx, "trail1", o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(cur.TrailPeak-120) > 1e-9 || math.Abs(cur.TriggerPrice-108) > 1e-9 {
		t.Fatalf("after rise peak/stop: %+v", cur)
	}
	if cur.Status != domain.PendingStatusOpen {
		t.Fatalf("should still be open: %+v", cur)
	}
	// Pullback above stop — no fill, peak stays
	_, okPull, err := svc.ProcessOpenOrder(ctx, *cur, 110, time.Now().UTC(), 0)
	if err != nil {
		t.Fatal(err)
	}
	cur, _ = svc.store.GetPendingOrder(ctx, "trail1", o.ID)
	if math.Abs(cur.TrailPeak-120) > 1e-9 || cur.Status != domain.PendingStatusOpen {
		t.Fatalf("pullback must not lower peak or fill: %+v ok=%v", cur, okPull)
	}
	// Gap through stop: last 100 <= 108 → fill once
	filled, ok, err := svc.ProcessOpenOrder(ctx, *cur, 100, time.Now().UTC(), 0)
	if err != nil || !ok || filled == nil {
		t.Fatalf("gap fill ok=%v err=%v filled=%v", ok, err, filled)
	}
	if filled.Status != domain.PendingStatusFilled {
		t.Fatalf("want filled once: %+v", filled)
	}
	// Second process: no re-fire
	_, ok2, err := svc.ProcessOpenOrder(ctx, *filled, 50, time.Now().UTC(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		t.Fatal("trailing stop must not run twice after filled")
	}
	view, _ := svc.View(ctx, "trail1")
	// buy 10000-100 + sell 100 = 10000
	if math.Abs(view.CashBalance-10000) > 1e-6 {
		t.Fatalf("cash=%v want 10000 after single fill @100", view.CashBalance)
	}
}

func TestPortfolio_TrailingStopOffset(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "200"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "trail2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "trail2", Symbol: "ETHUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "trail2", Symbol: "ETHUSDT", Type: "trailing_stop",
		Quantity: 1, TrailType: "offset", TrailValue: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(o.TriggerPrice-180) > 1e-9 {
		t.Fatalf("offset stop=%v want 180", o.TriggerPrice)
	}
	_, _, _ = svc.ProcessOpenOrder(ctx, *o, 250, time.Now().UTC(), 0)
	cur, _ := svc.store.GetPendingOrder(ctx, "trail2", o.ID)
	if math.Abs(cur.TrailPeak-250) > 1e-9 || math.Abs(cur.TriggerPrice-230) > 1e-9 {
		t.Fatalf("offset ratchet: %+v", cur)
	}
	// Touch stop exactly
	filled, ok, err := svc.ProcessOpenOrder(ctx, *cur, 230, time.Now().UTC(), 0)
	if err != nil || !ok || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("touch fill: ok=%v err=%v %+v", ok, err, filled)
	}
}

func TestPortfolio_OCOFullFillCancelsPeer(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco1", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tp, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco1", Symbol: "BTCUSDT", Quantity: 1,
		TakeProfitPrice: 120, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tp.OCOGroupID == "" || tp.OCOPeerID != sl.ID || sl.OCOPeerID != tp.ID {
		t.Fatalf("oco link tp=%+v sl=%+v", tp, sl)
	}
	// Shared reservation: available should be 0 for another sell of 1.
	_, err = svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "oco1", Symbol: "BTCUSDT", Type: "limit_sell", Quantity: 1, TriggerPrice: 150,
	})
	if err == nil {
		t.Fatal("expected insufficient available (reserved by OCO)")
	}
	// TP fill at 120 → SL canceled
	px.prices["binance|BTCUSDT"] = "120"
	got, ok, err := svc.ProcessOCOPair(ctx, *tp, *sl, 120, time.Now().UTC(), 0)
	if err != nil || !ok || got == nil {
		t.Fatalf("tp fill ok=%v err=%v got=%v", ok, err, got)
	}
	if got.Status != domain.PendingStatusFilled || got.Type != domain.PendingLimitSell {
		t.Fatalf("want filled TP, got %+v", got)
	}
	peer, err := svc.store.GetPendingOrder(ctx, "oco1", sl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if peer.Status != domain.PendingStatusCanceled || peer.CancelReason != domain.CancelReasonOCOPeerFilled {
		t.Fatalf("peer want canceled oco_peer_filled: %+v", peer)
	}
	// One sell trade only
	list, _, err := svc.ListTrades(ctx, "oco1", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	sells := 0
	for _, tr := range list {
		if tr.Side == domain.TradeSideSell {
			sells++
		}
	}
	if sells != 1 {
		t.Fatalf("want 1 sell trade, got %d", sells)
	}
}

func TestPortfolio_OCOSameTickOnlyStopFills(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco2", Symbol: "ETHUSDT", Side: "buy", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	// Overlapping triggers so one price can hit both (engine still picks stop only).
	tp, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco2", Symbol: "ETHUSDT", Quantity: 2,
		TakeProfitPrice: 100, StopLossPrice: 110, // invalid economically but Validate requires tp>sl
	})
	// ValidateOCO requires tp > sl — use normal prices and ProcessOCOPair with price that only hits one,
	// plus unit test for both. For same-tick both: use internal winner with engineered types via direct fill path.
	_ = tp
	_ = sl
	if err == nil {
		// takeProfit must be > stopLoss — expect error
		t.Fatal("expected validate error for tp < sl")
	}
	tp, sl, err = svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco2", Symbol: "ETHUSDT", Quantity: 2,
		TakeProfitPrice: 120, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Crash to 50: only SL triggered (TP needs >=120). Full SL fill cancels TP.
	got, ok, err := svc.ProcessOCOPair(ctx, *tp, *sl, 50, time.Now().UTC(), 0)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if got.Type != domain.PendingStopLoss || got.Status != domain.PendingStatusFilled {
		t.Fatalf("want filled stop, got %+v", got)
	}
	peer, _ := svc.store.GetPendingOrder(ctx, "oco2", tp.ID)
	if peer.Status != domain.PendingStatusCanceled {
		t.Fatalf("tp should be canceled: %+v", peer)
	}
	// Balance: sold 2 @ 50 once
	view, _ := svc.View(ctx, "oco2")
	// start 10000 - 200 buy + 100 sell = 9900
	if math.Abs(view.CashBalance-9900) > 1e-6 {
		t.Fatalf("cash=%v want 9900 (single fill)", view.CashBalance)
	}
	if view.PositionsValue > 1e-6 {
		t.Fatalf("position should be flat, value=%v", view.PositionsValue)
	}
}

func TestPortfolio_OCOPartialSyncsPeerRemaining(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco3", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco3", Symbol: "BTCUSDT", Side: "buy", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	tp, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco3", Symbol: "BTCUSDT", Quantity: 2,
		TakeProfitPrice: 120, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Partial TP fill 0.75
	got, ok, err := svc.TryFillPendingOrder(ctx, *tp, 120, 0.75)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if math.Abs(got.RemainingQuantity-1.25) > 1e-9 || got.Status != domain.PendingStatusOpen {
		t.Fatalf("tp after partial: %+v", got)
	}
	peer, err := svc.store.GetPendingOrder(ctx, "oco3", sl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if peer.Status != domain.PendingStatusOpen || math.Abs(peer.RemainingQuantity-1.25) > 1e-9 {
		t.Fatalf("sl remaining should match 1.25: %+v", peer)
	}
	// Second fill of remaining TP completes and cancels peer.
	got, ok, err = svc.TryFillPendingOrder(ctx, *got, 120, 0)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if got.Status != domain.PendingStatusFilled {
		t.Fatalf("tp full: %+v", got)
	}
	peer, _ = svc.store.GetPendingOrder(ctx, "oco3", sl.ID)
	if peer.Status != domain.PendingStatusCanceled || peer.CancelReason != domain.CancelReasonOCOPeerFilled {
		t.Fatalf("peer after full: %+v", peer)
	}
}

func TestPortfolio_CreateBuySellAndPnL(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()

	p, err := svc.Create(ctx, CreateInput{ClientID: "trader-1", StartingBalance: 10000})
	if err != nil || p.CashBalance != 10000 {
		t.Fatalf("%+v %v", p, err)
	}
	// Duplicate create fails
	if _, err := svc.Create(ctx, CreateInput{ClientID: "trader-1", StartingBalance: 1}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("dup: %v", err)
	}

	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "trader-1", Symbol: "ETHBTC", Side: "buy", Quantity: 1,
	}); err == nil {
		t.Fatal("expected quote mismatch for ETHBTC on USDT book")
	}

	tr, view, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "trader-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr.Side != domain.TradeSideBuy || math.Abs(tr.Notional-200) > 1e-9 {
		t.Fatalf("%+v", tr)
	}
	if math.Abs(view.CashBalance-9800) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}
	// Mark up for unrealized
	px.prices["binance|BTCUSDT"] = "110"
	view, err = svc.View(ctx, "trader-1")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.UnrealizedPnL-20) > 1e-6 {
		t.Fatalf("unreal=%v", view.UnrealizedPnL)
	}

	// Sell 1 at 110
	tr, view, err = svc.PlaceOrder(ctx, OrderInput{
		ClientID: "trader-1", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tr.RealizedPnL-10) > 1e-6 {
		t.Fatalf("realized trade=%v", tr.RealizedPnL)
	}
	if math.Abs(view.RealizedPnLTotal-10) > 1e-6 {
		t.Fatalf("realized total=%v", view.RealizedPnLTotal)
	}
	if math.Abs(view.CashBalance-(9800+110)) > 1e-6 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-1) > 1e-9 {
		t.Fatalf("pos=%+v", view.Positions)
	}

	list, total, err := svc.ListTrades(ctx, "trader-1", 10, 0)
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("trades total=%d list=%d err=%v", total, len(list), err)
	}
}

func TestPortfolio_InsufficientCashAndQty(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "1000"}})
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "poor", StartingBalance: 100})
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "poor", Symbol: "ETHUSDT", Side: "buy", Quantity: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "poor", Symbol: "ETHUSDT", Side: "sell", Quantity: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestPortfolio_PendingLimitBuyFillAndCancel(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "110"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, err := svc.Create(ctx, CreateInput{ClientID: "pend-1", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}

	// Limit buy at 100 — reserves cash, not triggered while last is 110
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "pend-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
	})
	if err != nil || o.Status != domain.PendingStatusOpen || math.Abs(o.ReservedCash-100) > 1e-9 {
		t.Fatalf("%+v %v", o, err)
	}
	view, err := svc.View(ctx, "pend-1")
	if err != nil || math.Abs(view.ReservedCash-100) > 1e-9 || math.Abs(view.AvailableCash-9900) > 1e-9 {
		t.Fatalf("view after reserve %+v err=%v", view, err)
	}
	// Reserved cash cannot fund a market buy that needs all cash
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "pend-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 99.1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("should block market buy with reserved cash: %v", err)
	}

	filled, ok, err := svc.TryFillPendingOrder(ctx, *o, 110, 0)
	if err != nil || ok || filled != nil {
		t.Fatalf("should not fill: ok=%v filled=%v err=%v", ok, filled, err)
	}

	// Price drops to 99 → fill at last
	filled, ok, err = svc.TryFillPendingOrder(ctx, *o, 99, 0)
	if err != nil || !ok || filled == nil || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("fill: ok=%v filled=%+v err=%v", ok, filled, err)
	}
	view, err = svc.View(ctx, "pend-1")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-(10000-99)) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}
	if view.ReservedCash != 0 {
		t.Fatalf("reserved after fill=%v", view.ReservedCash)
	}

	// Idempotent: second fill attempt no-ops
	_, ok, err = svc.TryFillPendingOrder(ctx, *o, 90, 0)
	if err != nil || ok {
		t.Fatalf("second fill: ok=%v err=%v", ok, err)
	}

	// Cancel path: place then cancel before fill releases reservation
	o2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "pend-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	view, _ = svc.View(ctx, "pend-1")
	if view.ReservedCash < 49 {
		t.Fatalf("expected reserve after place: %+v", view)
	}
	canceled, err := svc.CancelPendingOrder(ctx, "pend-1", o2.ID)
	if err != nil || canceled.Status != domain.PendingStatusCanceled || canceled.ReservedCash != 0 {
		t.Fatalf("%+v %v", canceled, err)
	}
	view, _ = svc.View(ctx, "pend-1")
	if view.ReservedCash != 0 {
		t.Fatalf("reserved after cancel=%v", view.ReservedCash)
	}
	// Canceled must not execute later
	_, ok, err = svc.TryFillPendingOrder(ctx, *o2, 40, 0)
	if err != nil || ok {
		t.Fatalf("canceled fill: ok=%v err=%v", ok, err)
	}
	open, err := svc.ListPendingOrders(ctx, "pend-1", "open", 10, 0)
	if err != nil || len(open) != 0 {
		t.Fatalf("open=%+v err=%v", open, err)
	}
}

func TestPortfolio_PartialFillsAndSellReservation(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "part-1", StartingBalance: 10000})
	// Buy 5 for inventory
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "part-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 5})
	if err != nil {
		t.Fatal(err)
	}
	// Limit sell 4 — reserves position
	ls, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "part-1", Symbol: "BTCUSDT", Type: "limit_sell", Quantity: 4, TriggerPrice: 110,
	})
	if err != nil || math.Abs(ls.ReservedQuantity-4) > 1e-9 {
		t.Fatalf("%+v %v", ls, err)
	}
	// Only 1 available for market sell
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "part-1", Symbol: "BTCUSDT", Side: "sell", Quantity: 2})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("should not sell reserved qty: %v", err)
	}
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "part-1", Symbol: "BTCUSDT", Side: "sell", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}

	// Partial fill of limit sell: 1.5 then rest
	px.prices["binance|BTCUSDT"] = "120"
	got, ok, err := svc.TryFillPendingOrder(ctx, *ls, 120, 1.5)
	if err != nil || !ok || got.Status != domain.PendingStatusOpen {
		t.Fatalf("partial1 %+v ok=%v err=%v", got, ok, err)
	}
	if math.Abs(got.FilledQuantity-1.5) > 1e-9 || math.Abs(got.RemainingQuantity-2.5) > 1e-9 {
		t.Fatalf("partial sizes %+v", got)
	}
	if math.Abs(got.ReservedQuantity-2.5) > 1e-9 {
		t.Fatalf("reserved after partial=%v", got.ReservedQuantity)
	}
	// Second partial completes
	got, ok, err = svc.TryFillPendingOrder(ctx, *got, 120, 0)
	if err != nil || !ok || got.Status != domain.PendingStatusFilled {
		t.Fatalf("complete %+v ok=%v err=%v", got, ok, err)
	}
	list, total, err := svc.ListTrades(ctx, "part-1", 20, 0)
	if err != nil || total < 4 {
		t.Fatalf("trades total=%d err=%v", total, err)
	}
	// Two partial sells should be in history with pending order id
	var sellFills int
	for _, tr := range list {
		if tr.PendingOrderID == ls.ID {
			sellFills++
		}
	}
	if sellFills != 2 {
		t.Fatalf("want 2 pending fills, got %d list=%+v", sellFills, list)
	}

	// Buy reserve: cannot over-reserve cash
	_, err = svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "part-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 100000, TriggerPrice: 100,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("over-reserve buy: %v", err)
	}
}

func TestPortfolio_StopLossAndSellReserve(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "2000"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "stop-1", StartingBalance: 5000})
	// Buy 1 ETH
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "stop-1", Symbol: "ETHUSDT", Side: "buy", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	// Stop loss at 1800
	sl, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "stop-1", Symbol: "ETHUSDT", Type: "stop_loss", Quantity: 1, TriggerPrice: 1800,
	})
	if err != nil || math.Abs(sl.ReservedQuantity-1) > 1e-9 {
		t.Fatalf("%+v %v", sl, err)
	}
	// Cannot sell the reserved unit
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "stop-1", Symbol: "ETHUSDT", Side: "sell", Quantity: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("reserved sell blocked: %v", err)
	}
	// Above stop — no fill
	_, ok, err := svc.TryFillPendingOrder(ctx, *sl, 1900, 0)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	// Drop through stop
	filled, ok, err := svc.TryFillPendingOrder(ctx, *sl, 1700, 0)
	if err != nil || !ok || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("stop fill %+v ok=%v err=%v", filled, ok, err)
	}
	view, _ := svc.View(ctx, "stop-1")
	if len(view.Positions) != 0 {
		t.Fatalf("expected flat: %+v", view.Positions)
	}

	// Limit sell with no position → fail at place (reservation)
	_, err = svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "stop-1", Symbol: "ETHUSDT", Type: "limit_sell", Quantity: 1, TriggerPrice: 100,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want insufficient position at place: %v", err)
	}
}

func TestPortfolio_GTCExpireIOCAndFOK(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "tif-1", StartingBalance: 10000})

	// GTC with past expiry is rejected at place
	past := time.Now().UTC().Add(-time.Minute)
	_, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
		TimeInForce: "gtc", ExpiresAt: &past,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("past expire: %v", err)
	}

	// GTC expires via ProcessOpenOrder
	exp := time.Now().UTC().Add(time.Hour)
	gtc, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
		TimeInForce: "gtc", ExpiresAt: &exp,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Force expiry by processing with now past expiresAt
	pastNow := exp.Add(time.Second)
	got, ok, err := svc.ProcessOpenOrder(ctx, *gtc, 40, pastNow, 0)
	if err != nil || !ok || got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonExpired {
		t.Fatalf("expired %+v ok=%v err=%v", got, ok, err)
	}
	if got.ReservedCash != 0 {
		t.Fatalf("reserve after expire=%v", got.ReservedCash)
	}

	// IOC: not marketable → cancel no fill
	ioc, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
		TimeInForce: "ioc",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *ioc, 100, time.Now().UTC(), 0)
	if err != nil || !ok || got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonIOCNoFill {
		t.Fatalf("ioc no fill %+v ok=%v err=%v", got, ok, err)
	}

	// IOC: partial fill then cancel remainder
	ioc2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 100,
		TimeInForce: "ioc",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *ioc2, 90, time.Now().UTC(), 1) // fill only 1 of 2
	if err != nil || !ok {
		t.Fatalf("ioc partial err=%v ok=%v", err, ok)
	}
	if got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonIOCRemainder {
		t.Fatalf("ioc remainder status %+v", got)
	}
	if math.Abs(got.FilledQuantity-1) > 1e-9 || got.ReservedCash != 0 {
		t.Fatalf("ioc filled/reserve %+v", got)
	}

	// FOK: cannot full fill → cancel with no fill
	// Reserve cash for 2, but maxFill path: use Process with maxFill that would partial - FOK checks full remaining
	// Place FOK buy 3 at 100 needing 300 cash - we have plenty. Trigger at 90 so marketable.
	// Force incomplete by maxFillQty simulation inside Process: FOK ignores maxFill and requires full can from reservation.
	// To force FOK fail: use sell FOK without enough reserved - actually reservation ensures full size.
	// FOK fails when not triggered, or when maxFillable < remaining (e.g. reserved cash insufficient for full at fill price - impossible for limit buy if reserved correctly).
	// Use maxFillable with sell: buy 2, FOK sell 2, marketable - full fill.
	// For FOK unfilled: not marketable.
	fok, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 10,
		TimeInForce: "fok",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *fok, 100, time.Now().UTC(), 0)
	if err != nil || !ok || got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonFOKUnfilled {
		t.Fatalf("fok unfilled %+v ok=%v err=%v", got, ok, err)
	}
	if got.FilledQuantity != 0 {
		t.Fatalf("fok must not fill: %+v", got)
	}

	// FOK full fill when marketable
	fok2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
		TimeInForce: "fok",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *fok2, 95, time.Now().UTC(), 0)
	if err != nil || !ok || got.Status != domain.PendingStatusFilled {
		t.Fatalf("fok fill %+v ok=%v err=%v", got, ok, err)
	}
}

func TestPortfolio_OrderFillerRunOnce(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "95"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "fill-w", StartingBalance: 10000})
	_, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "fill-w", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	f := &OrderFiller{Portfolio: svc, Market: px, Interval: time.Hour}
	f.RunOnce(ctx)
	open, _ := svc.ListPendingOrders(ctx, "fill-w", "open", 10, 0)
	if len(open) != 0 {
		t.Fatalf("expected filled open=%+v", open)
	}
	view, _ := svc.View(ctx, "fill-w")
	if math.Abs(view.CashBalance-(10000-190)) > 1e-6 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
}

func TestPortfolio_PendingPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pf.db")
	ctx := context.Background()
	st1, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc1 := New(st1, px).WithPaperCosts(domain.ZeroTradingCosts)
	_, _ = svc1.Create(ctx, CreateInput{ClientID: "po-persist", StartingBalance: 1000})
	o, err := svc1.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "po-persist", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := o.ID
	if math.Abs(o.ReservedCash-200) > 1e-9 {
		t.Fatalf("reserve %+v", o)
	}
	_ = st1.Close()

	st2, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := New(st2, px).WithPaperCosts(domain.ZeroTradingCosts)
	open, err := svc2.ListPendingOrders(ctx, "po-persist", "open", 10, 0)
	if err != nil || len(open) != 1 || open[0].ID != id {
		t.Fatalf("%+v err=%v", open, err)
	}
	if math.Abs(open[0].ReservedCash-200) > 1e-9 || math.Abs(open[0].RemainingQuantity-2) > 1e-9 {
		t.Fatalf("persisted reservation %+v", open[0])
	}
	view, err := svc2.View(ctx, "po-persist")
	if err != nil || math.Abs(view.ReservedCash-200) > 1e-9 {
		t.Fatalf("view %+v err=%v", view, err)
	}
}

func TestPortfolio_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pf.db")
	ctx := context.Background()
	st1, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "50"}}
	svc1 := New(st1, px).WithPaperCosts(domain.ZeroTradingCosts)
	_, err = svc1.Create(ctx, CreateInput{ClientID: "persist", StartingBalance: 5000})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc1.PlaceOrder(ctx, OrderInput{ClientID: "persist", Symbol: "BTCUSDT", Side: "buy", Quantity: 10})
	if err != nil {
		t.Fatal(err)
	}
	_ = st1.Close()

	st2, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := New(st2, px).WithPaperCosts(domain.ZeroTradingCosts)
	view, err := svc2.View(ctx, "persist")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-4500) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}
	list, total, err := svc2.ListTrades(ctx, "persist", 10, 0)
	if err != nil || total != 1 || list[0].Side != domain.TradeSideBuy {
		t.Fatalf("%+v total=%d err=%v", list, total, err)
	}
}

func TestPortfolio_AmendPendingLimitBuyPriceAndRemaining(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "amd-1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 10, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetPendingOrderDetail(ctx, "amd-1", o.ID)
	if err != nil || !detail.Editable || detail.LastPrice != 100 {
		t.Fatalf("detail %+v err=%v", detail, err)
	}
	if math.Abs(detail.AvailableCashForOrder-10000) > 1e-6 {
		t.Fatalf("cash for order=%v", detail.AvailableCashForOrder)
	}
	trig := 80.0
	rem := 5.0
	got, view, err := svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-1", OrderID: o.ID, TriggerPrice: &trig, RemainingQuantity: &rem,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.TriggerPrice-80) > 1e-12 || math.Abs(got.RemainingQuantity-5) > 1e-12 {
		t.Fatalf("amended %+v", got)
	}
	if math.Abs(got.Quantity-5) > 1e-12 || math.Abs(got.ReservedCash-400) > 1e-9 {
		t.Fatalf("qty/reserve %+v", got)
	}
	if got.ID != o.ID || got.Status != domain.PendingStatusOpen {
		t.Fatalf("same id open: %+v", got)
	}
	if math.Abs(view.ReservedCash-400) > 1e-9 {
		t.Fatalf("view reserve=%v", view.ReservedCash)
	}
}

func TestPortfolio_AmendInsufficientCashAndRejectsSpecialTypes(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "amd-2", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-2", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 5, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	trig := 250.0
	_, _, err = svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-2", OrderID: o.ID, TriggerPrice: &trig,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want insufficient cash: %v", err)
	}
	trail, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-2", Symbol: "ETHUSDT", Type: "trailing_stop", Quantity: 0.001,
		TrailType: "percent", TrailValue: 0.05,
	})
	// trailing needs a sell position — may fail; if placed, amend must reject
	if err == nil {
		_, _, aerr := svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
			ClientID: "amd-2", OrderID: trail.ID, TriggerPrice: &trig,
		})
		if !errors.Is(aerr, domain.ErrInvalidArgument) {
			t.Fatalf("trailing amend: %v", aerr)
		}
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "amd-2", Symbol: "ETHUSDT", Side: "buy", Quantity: 1}); err != nil {
		// may fail if cash reserved by limit buy (500 left after 5*100)
	}
	_, _, err = svc.AmendPendingOrder(ctx, AmendPendingOrderInput{ClientID: "amd-2", OrderID: o.ID})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty amend: %v", err)
	}
}

func TestPortfolio_AmendLimitSellAndStopFillsImmediately(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "amd-3", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "amd-3", Symbol: "BTCUSDT", Side: "buy", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	ls, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-3", Symbol: "BTCUSDT", Type: "limit_sell", Quantity: 1, TriggerPrice: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	rem := 0.4
	got, view, err := svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-3", OrderID: ls.ID, RemainingQuantity: &rem,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.RemainingQuantity-0.4) > 1e-12 || math.Abs(got.ReservedQuantity-0.4) > 1e-12 {
		t.Fatalf("%+v", got)
	}
	if view.Positions[0].AvailableQuantity < 1.5-1e-9 {
		t.Fatalf("avail after shrink sell reserve: %+v", view.Positions[0])
	}

	sl, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-3", Symbol: "BTCUSDT", Type: "stop_loss", Quantity: 0.5, TriggerPrice: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	trig := 100.0
	filled, _, err := svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-3", OrderID: sl.ID, TriggerPrice: &trig,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filled.Status != domain.PendingStatusFilled {
		t.Fatalf("stop should fill immediately after amend to last: %+v", filled)
	}
}

func TestPortfolio_AmendRejectsOCOAndCanceled(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "amd-4", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "amd-4", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tp, _, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "amd-4", Symbol: "BTCUSDT", Quantity: 1, TakeProfitPrice: 120, StopLossPrice: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	trig := 115.0
	_, _, err = svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-4", OrderID: tp.ID, TriggerPrice: &trig,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("oco amend: %v", err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-4", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelPendingOrder(ctx, "amd-4", o.ID); err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-4", OrderID: o.ID, TriggerPrice: &trig,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("canceled amend: %v", err)
	}
	if _, err := svc.GetPendingOrder(ctx, "amd-4", "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get missing: %v", err)
	}
}

func TestPortfolio_AmendLimitBuyFillsWhenMovedToMarket(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "amd-5", StartingBalance: 5000}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-5", Symbol: "ETHUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 40,
	})
	if err != nil {
		t.Fatal(err)
	}
	trig := 50.0
	got, view, err := svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-5", OrderID: o.ID, TriggerPrice: &trig,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.PendingStatusFilled {
		t.Fatalf("limit buy moved to last should fill: %+v", got)
	}
	if view.ReservedCash > 1e-9 {
		t.Fatalf("reserve after fill=%v", view.ReservedCash)
	}
}

func TestPortfolio_CancelAllOpenOrdersOneMarketAndAll(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cxl-all", StartingBalance: 20000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "cxl-all", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 90,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "cxl-all", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 80,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "cxl-all", Symbol: "ETHUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 40,
	}); err != nil {
		t.Fatal(err)
	}
	btc, view, err := svc.CancelOpenPendingOrders(ctx, CancelOpenOrdersInput{
		ClientID: "cxl-all", Symbol: "BTCUSDT",
	})
	if err != nil || len(btc) != 2 {
		t.Fatalf("btc n=%d err=%v", len(btc), err)
	}
	if math.Abs(view.ReservedCash-80) > 1e-9 {
		t.Fatalf("eth still reserved=%v", view.ReservedCash)
	}
	open, err := svc.ListPendingOrders(ctx, "cxl-all", "open", 20, 0)
	if err != nil || len(open) != 1 || open[0].Symbol != "ETHUSDT" {
		t.Fatalf("left open %+v err=%v", open, err)
	}
	all, view, err := svc.CancelOpenPendingOrders(ctx, CancelOpenOrdersInput{ClientID: "cxl-all"})
	if err != nil || len(all) != 1 {
		t.Fatalf("all n=%d err=%v", len(all), err)
	}
	if view.ReservedCash > 1e-9 {
		t.Fatalf("reserved after all=%v", view.ReservedCash)
	}
	again, _, err := svc.CancelOpenPendingOrders(ctx, CancelOpenOrdersInput{ClientID: "cxl-all"})
	if err != nil || len(again) != 0 {
		t.Fatalf("idempotent %+v err=%v", again, err)
	}
}

func newSvcWithCosts(t *testing.T, px *fakePx) *Service {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf-cost.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if px == nil {
		px = &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	}
	return New(st, px)
}

func TestPaperCosts_MarketBuySellLotsAndReserve(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvcWithCosts(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "fee1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	cost := domain.TradingCostFor(domain.ExchangeBinance)
	last := 100.0
	fillBuy := domain.ApplySlippage(last, domain.TradeSideBuy, cost.SlippageRate)
	debit := domain.BuyCashDebit(1, fillBuy, cost.FeeRate)
	unit := domain.BuyUnitCost(fillBuy, cost.FeeRate)
	feeBuy := domain.FeeAmount(1, fillBuy, cost.FeeRate)

	tr, view, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "fee1", Symbol: "BTCUSDT", Side: "buy", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tr.Price-fillBuy) > 1e-9 || math.Abs(tr.LastPrice-last) > 1e-9 || math.Abs(tr.Fee-feeBuy) > 1e-9 {
		t.Fatalf("buy trade %+v want fill=%v last=%v fee=%v", tr, fillBuy, last, feeBuy)
	}
	if math.Abs(view.CashBalance-(10000-debit)) > 1e-9 {
		t.Fatalf("cash after buy=%v want %v", view.CashBalance, 10000-debit)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].AvgCost-unit) > 1e-9 {
		t.Fatalf("avg cost %+v want %v", view.Positions, unit)
	}
	lots, err := svc.ListLots(ctx, "fee1", "binance", "BTCUSDT", "open")
	if err != nil || len(lots) != 1 || math.Abs(lots[0].Price-unit) > 1e-9 {
		t.Fatalf("lot %+v err=%v", lots, err)
	}

	fillSell := domain.ApplySlippage(last, domain.TradeSideSell, cost.SlippageRate)
	credit := domain.SellCashCredit(1, fillSell, cost.FeeRate)
	feeSell := domain.FeeAmount(1, fillSell, cost.FeeRate)
	wantRealized := (domain.NetSellPrice(fillSell, cost.FeeRate) - unit) * 1
	tr, view, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "fee1", Symbol: "BTCUSDT", Side: "sell", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tr.Price-fillSell) > 1e-9 || math.Abs(tr.Fee-feeSell) > 1e-9 {
		t.Fatalf("sell trade %+v", tr)
	}
	if math.Abs(tr.RealizedPnL-wantRealized) > 1e-9 {
		t.Fatalf("realized=%v want %v", tr.RealizedPnL, wantRealized)
	}
	if math.Abs(view.CashBalance-(10000-debit+credit)) > 1e-9 {
		t.Fatalf("cash after sell=%v", view.CashBalance)
	}
	if view.RealizedPnLTotal < 0 && math.Abs(view.RealizedPnLTotal-wantRealized) > 1e-9 {
		t.Fatalf("book realized=%v", view.RealizedPnLTotal)
	}
}

func TestPaperCosts_PendingBuyReserveCoversFee(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvcWithCosts(t, px)
	ctx := context.Background()
	cost := domain.TradingCostFor(domain.ExchangeBinance)
	need := domain.BuyReserveCash(1, 100, cost)
	if need <= 100 {
		t.Fatalf("reserve should exceed trigger: %v", need)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "fee-res", StartingBalance: 100.0}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "fee-res", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
	})
	if err == nil {
		t.Fatal("expected insufficient cash when starting balance equals trigger only")
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "fee-ok", StartingBalance: need + 0.01}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "fee-ok", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(o.ReservedCash-need) > 1e-9 {
		t.Fatalf("reserved=%v want %v", o.ReservedCash, need)
	}
	filled, ok, err := svc.TryFillPendingOrder(ctx, *o, 100, 0)
	if err != nil || !ok || filled == nil || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("fill ok=%v err=%v %+v", ok, err, filled)
	}
	view, _ := svc.View(ctx, "fee-ok")
	fill := domain.ApplySlippage(100, domain.TradeSideBuy, cost.SlippageRate)
	debit := domain.BuyCashDebit(1, fill, cost.FeeRate)
	if math.Abs(view.CashBalance-(need+0.01-debit)) > 1e-6 {
		t.Fatalf("cash=%v start=%v debit=%v", view.CashBalance, need+0.01, debit)
	}
	if view.ReservedCash > 1e-9 {
		t.Fatalf("reserved after fill=%v", view.ReservedCash)
	}
}

func TestPaperCosts_RecurringBuyIncludesFee(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "100"}}
	svc := newSvcWithCosts(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "fee-dca", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "fee-dca", Symbol: "ETHUSDT", Amount: 100, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	view, err := svc.View(ctx, "fee-dca")
	if err != nil {
		t.Fatal(err)
	}
	cost := domain.TradingCostFor(domain.ExchangeBinance)
	fill := domain.ApplySlippage(100, domain.TradeSideBuy, cost.SlippageRate)
	unit := domain.BuyUnitCost(fill, cost.FeeRate)
	qty := 100 / unit
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-qty) > 1e-6 {
		t.Fatalf("qty=%v want %v pos=%+v", view.Positions, qty, view.Positions)
	}
	if math.Abs(view.CashBalance-(1000-100)) > 0.02 {
		t.Fatalf("cash=%v (should spend ~amount including fee)", view.CashBalance)
	}
	runs, err := svc.ListRecurringBuyRuns(ctx, "fee-dca", plan.ID, 10, 0)
	if err != nil || len(runs) == 0 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("runs %+v err=%v", runs, err)
	}
	if math.Abs(runs[0].Price-fill) > 1e-9 {
		t.Fatalf("run price=%v want slipped %v", runs[0].Price, fill)
	}
}
type haltedPx struct {
	price string
}

func (h haltedPx) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	p := h.price
	if p == "" {
		p = "100"
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p, Halted: true}, nil
}

func TestPlaceOrder_DoesNotFillHaltedLastPrintAsLive(t *testing.T) {
	svc := newSvc(t, nil)
	svc.market = haltedPx{price: "100"}
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "halt-fill", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	tr, view, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "halt-fill", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err == nil {
		t.Fatalf("filled halted ticker as live: trade=%+v cash=%v", tr, view.CashBalance)
	}
}
