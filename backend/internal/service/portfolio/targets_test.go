package portfolio

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPortfolio_OCOTargetsReduceStop(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-tp", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-tp", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tps, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-tp", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, OffsetPercent: 5},
			{Percent: 25, OffsetPercent: 10},
			{Percent: 50, OffsetPercent: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tps) != 3 {
		t.Fatalf("targets: %d", len(tps))
	}
	if math.Abs(tps[0].TriggerPrice-105) > 1e-9 || math.Abs(tps[0].Quantity-0.25) > 1e-9 {
		t.Fatalf("nearest: %+v", tps[0])
	}
	if math.Abs(tps[2].TriggerPrice-115) > 1e-9 || math.Abs(tps[2].Quantity-0.5) > 1e-9 {
		t.Fatalf("furthest: %+v", tps[2])
	}
	if math.Abs(sl.Quantity-1) > 1e-9 {
		t.Fatalf("stop starts at the whole size: %+v", sl)
	}
	if _, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "oco-tp", Symbol: "BTCUSDT", Type: "limit_sell", Quantity: 0.1, TriggerPrice: 200,
	}); err == nil {
		t.Fatal("expected the whole coin to stay reserved")
	}

	now := time.Now().UTC()
	if _, ok, err := svc.ProcessOCOGroup(ctx, ordersOf(tps, sl), 110, now, 0); err != nil || !ok {
		t.Fatalf("fills ok=%v err=%v", ok, err)
	}
	slNow, err := svc.store.GetPendingOrder(ctx, "oco-tp", sl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if slNow.Status != domain.PendingStatusOpen || math.Abs(slNow.RemainingQuantity-0.5) > 1e-9 {
		t.Fatalf("stop should cover the remaining half: %+v", slNow)
	}
	far, err := svc.store.GetPendingOrder(ctx, "oco-tp", tps[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	if far.Status != domain.PendingStatusOpen || math.Abs(far.RemainingQuantity-0.5) > 1e-9 {
		t.Fatalf("the +15%% target stays: %+v", far)
	}
	view, err := svc.View(ctx, "oco-tp")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-0.5) > 1e-9 {
		t.Fatalf("half the coin remains: %+v", view.Positions)
	}

	if _, ok, err := svc.ProcessOCOGroup(ctx, ordersOf(tps, sl), 80, now, 0); err != nil || !ok {
		t.Fatalf("stop fill ok=%v err=%v", ok, err)
	}
	slDone, _ := svc.store.GetPendingOrder(ctx, "oco-tp", sl.ID)
	if slDone.Status != domain.PendingStatusFilled {
		t.Fatalf("stop filled: %+v", slDone)
	}
	far, _ = svc.store.GetPendingOrder(ctx, "oco-tp", tps[2].ID)
	if far.Status != domain.PendingStatusCanceled || far.CancelReason != domain.CancelReasonOCOPeerFilled {
		t.Fatalf("remaining target canceled: %+v", far)
	}
	view, _ = svc.View(ctx, "oco-tp")
	if len(view.Positions) != 0 {
		t.Fatalf("flat after the stop: %+v", view.Positions)
	}
}

func TestPortfolio_BracketTargetsTrackEntryAndStop(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "br-tp", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	entry, tps, sl, err := svc.PlaceBracketOrder(ctx, BracketOrderInput{
		ClientID: "br-tp", Symbol: "BTCUSDT", Quantity: 1, EntryPrice: 100, StopLossPrice: 90,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, OffsetPercent: 5},
			{Percent: 25, OffsetPercent: 10},
			{Percent: 50, OffsetPercent: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tps[0].TriggerPrice-105) > 1e-9 || tps[0].Status != domain.PendingStatusPending {
		t.Fatalf("pending target: %+v", tps[0])
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *entry, 100, 0.4); err != nil || !ok {
		t.Fatalf("partial entry err=%v", err)
	}
	first, _ := svc.store.GetPendingOrder(ctx, "br-tp", tps[0].ID)
	stop, _ := svc.store.GetPendingOrder(ctx, "br-tp", sl.ID)
	if math.Abs(first.RemainingQuantity-0.1) > 1e-9 || math.Abs(stop.RemainingQuantity-0.4) > 1e-9 {
		t.Fatalf("sized to the fill: tp=%v sl=%v", first.RemainingQuantity, stop.RemainingQuantity)
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *first, 105, 0); err != nil || !ok {
		t.Fatalf("target fill err=%v", err)
	}
	stop, _ = svc.store.GetPendingOrder(ctx, "br-tp", sl.ID)
	if math.Abs(stop.RemainingQuantity-0.3) > 1e-9 {
		t.Fatalf("stop shrinks after the target: %+v", stop)
	}
	entryNow, _ := svc.store.GetPendingOrder(ctx, "br-tp", entry.ID)
	if _, ok, err := svc.TryFillPendingOrder(ctx, *entryNow, 100, 0); err != nil || !ok {
		t.Fatalf("rest of entry err=%v", err)
	}
	first, _ = svc.store.GetPendingOrder(ctx, "br-tp", tps[0].ID)
	third, _ := svc.store.GetPendingOrder(ctx, "br-tp", tps[2].ID)
	stop, _ = svc.store.GetPendingOrder(ctx, "br-tp", sl.ID)
	if math.Abs(first.RemainingQuantity-0.15) > 1e-9 {
		t.Fatalf("first target grows with the rest of the buy: %+v", first)
	}
	if math.Abs(third.RemainingQuantity-0.5) > 1e-9 {
		t.Fatalf("half target: %+v", third)
	}
	if math.Abs(stop.RemainingQuantity-0.9) > 1e-9 {
		t.Fatalf("stop covers what is still held: %+v", stop)
	}
}

func TestPortfolio_PartialStopShrinksOpenTargets(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-part", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-part", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tps, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-part", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, Price: 105},
			{Percent: 25, Price: 110},
			{Percent: 50, Price: 115},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *sl, 90, 0.4); err != nil || !ok {
		t.Fatalf("partial stop ok=%v err=%v", ok, err)
	}
	var sum float64
	for _, tp := range tps {
		got, gerr := svc.store.GetPendingOrder(ctx, "oco-part", tp.ID)
		if gerr != nil {
			t.Fatal(gerr)
		}
		if got.Status != domain.PendingStatusOpen {
			t.Fatalf("target should stay open: %+v", got)
		}
		sum += got.RemainingQuantity
	}
	if math.Abs(sum-0.6) > 1e-6 {
		t.Fatalf("open targets should add up to the coins still held, got %v", sum)
	}
	stop, _ := svc.store.GetPendingOrder(ctx, "oco-part", sl.ID)
	if math.Abs(stop.RemainingQuantity-0.6) > 1e-6 {
		t.Fatalf("stop remaining: %+v", stop)
	}
}

func TestPortfolio_CancelOneTargetLeavesStop(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-c", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-c", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tps, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-c", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, Price: 105},
			{Percent: 25, Price: 110},
			{Percent: 50, Price: 115},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelPendingOrder(ctx, "oco-c", tps[0].ID); err != nil {
		t.Fatal(err)
	}
	stop, err := svc.store.GetPendingOrder(ctx, "oco-c", sl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stop.Status != domain.PendingStatusOpen || math.Abs(stop.RemainingQuantity-1) > 1e-9 {
		t.Fatalf("stop still covers the coin: %+v", stop)
	}
	other, _ := svc.store.GetPendingOrder(ctx, "oco-c", tps[1].ID)
	if other.Status != domain.PendingStatusOpen {
		t.Fatalf("other target stays: %+v", other)
	}
}

func TestPreview_OCOTargets(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-tp", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "prev-tp", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-tp", Symbol: "BTCUSDT", Type: "oco", Quantity: 1, StopLossPrice: 90,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, OffsetPercent: 5},
			{Percent: 25, OffsetPercent: 10},
			{Percent: 50, OffsetPercent: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Legs) != 4 {
		t.Fatalf("legs: %+v", got.Legs)
	}
	if got.Legs[0].Percent != 25 || math.Abs(got.Legs[0].Quantity-0.25) > 1e-9 || math.Abs(got.Legs[0].ReferencePrice-105) > 1e-6 {
		t.Fatalf("first leg: %+v", got.Legs[0])
	}
	if got.Legs[3].Role != "stop_loss" || math.Abs(got.Legs[3].Quantity-1) > 1e-9 {
		t.Fatalf("stop leg: %+v", got.Legs[3])
	}
}

func ordersOf(tps []*domain.PendingOrder, sl *domain.PendingOrder) []domain.PendingOrder {
	out := make([]domain.PendingOrder, 0, len(tps)+1)
	for _, tp := range tps {
		if tp != nil {
			out = append(out, *tp)
		}
	}
	if sl != nil {
		out = append(out, *sl)
	}
	return out
}
