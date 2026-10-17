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

func TestPortfolio_MoveStopToEntryAfterNearestTarget(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-move", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-move", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	targets := []domain.TakeProfitTarget{
		{Percent: 25, Price: 105},
		{Percent: 25, Price: 110},
		{Percent: 50, Price: 115},
	}
	tps, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-move", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90,
		TakeProfitTargets: targets, MoveStopToEntry: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sl.MoveStopToEntry || math.Abs(sl.EntryPrice-100) > 1e-9 {
		t.Fatalf("stop should remember the average cost: %+v", sl)
	}
	now := time.Now().UTC()
	if _, ok, err := svc.ProcessOCOGroup(ctx, ordersOf(tps, sl), 106, now, 0); err != nil || !ok {
		t.Fatalf("nearest fill ok=%v err=%v", ok, err)
	}
	stop, err := svc.store.GetPendingOrder(ctx, "oco-move", sl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(stop.TriggerPrice-100) > 1e-9 || math.Abs(stop.RemainingQuantity-0.75) > 1e-6 {
		t.Fatalf("stop should rise to the buy price and cover 0.75: %+v", stop)
	}
	second, _ := svc.store.GetPendingOrder(ctx, "oco-move", tps[1].ID)
	if second.Status != domain.PendingStatusOpen {
		t.Fatalf("the next target stays open: %+v", second)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-still", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-still", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	off, offSL, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-still", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90,
		TakeProfitTargets: targets,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := svc.ProcessOCOGroup(ctx, ordersOf(off, offSL), 106, now, 0); err != nil || !ok {
		t.Fatalf("plain fill ok=%v err=%v", ok, err)
	}
	plain, _ := svc.store.GetPendingOrder(ctx, "oco-still", offSL.ID)
	if math.Abs(plain.TriggerPrice-90) > 1e-9 || math.Abs(plain.RemainingQuantity-0.75) > 1e-6 {
		t.Fatalf("without the option the stop stays: %+v", plain)
	}
}

func TestPortfolio_PartialTargetDoesNotMoveStop(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-part-move", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-part-move", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tps, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-part-move", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90, MoveStopToEntry: true,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, Price: 105},
			{Percent: 75, Price: 115},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *tps[0], 106, 0.1); err != nil || !ok {
		t.Fatalf("partial ok=%v err=%v", ok, err)
	}
	stop, _ := svc.store.GetPendingOrder(ctx, "oco-part-move", sl.ID)
	if math.Abs(stop.TriggerPrice-90) > 1e-9 || math.Abs(stop.RemainingQuantity-0.9) > 1e-6 {
		t.Fatalf("a partial nearest target leaves the stop: %+v", stop)
	}
	first, _ := svc.store.GetPendingOrder(ctx, "oco-part-move", tps[0].ID)
	if _, ok, err := svc.TryFillPendingOrder(ctx, *first, 106, 0); err != nil || !ok {
		t.Fatalf("rest of target ok=%v err=%v", ok, err)
	}
	stop, _ = svc.store.GetPendingOrder(ctx, "oco-part-move", sl.ID)
	if math.Abs(stop.TriggerPrice-100) > 1e-9 || math.Abs(stop.RemainingQuantity-0.75) > 1e-6 {
		t.Fatalf("the stop moves once the nearest target is complete: %+v", stop)
	}
}

func TestPortfolio_CanceledNearestTargetDoesNotMoveStop(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-skip", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-skip", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	tps, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-skip", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90, MoveStopToEntry: true,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, Price: 105},
			{Percent: 25, Price: 110},
			{Percent: 50, Price: 115},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelPendingOrder(ctx, "oco-skip", tps[0].ID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, ok, err := svc.ProcessOCOGroup(ctx, ordersOf(tps[1:], sl), 120, now, 0); err != nil || !ok {
		t.Fatalf("later targets ok=%v err=%v", ok, err)
	}
	stop, _ := svc.store.GetPendingOrder(ctx, "oco-skip", sl.ID)
	if stop.Status != domain.PendingStatusOpen || math.Abs(stop.TriggerPrice-90) > 1e-9 || math.Abs(stop.RemainingQuantity-0.25) > 1e-6 {
		t.Fatalf("a later target does not move the stop: %+v", stop)
	}
}

func TestPortfolio_MoveStopRejectsBuyPriceOutOfRange(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-bad", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "oco-bad", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-bad", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 90, MoveStopToEntry: true,
		TakeProfitTargets: []domain.TakeProfitTarget{{Percent: 100, Price: 99}},
	})
	if err == nil {
		t.Fatal("expected the buy price above the first target to be rejected")
	}
	_, _, err = svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-bad", Symbol: "BTCUSDT", Quantity: 1, StopLossPrice: 110, MoveStopToEntry: true,
		TakeProfitTargets: []domain.TakeProfitTarget{{Percent: 100, Price: 120}},
	})
	if err == nil {
		t.Fatal("expected a buy price at or below the stop to be rejected")
	}
}

func TestPreview_MoveStopToEntry(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-move", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "prev-move", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-move", Symbol: "BTCUSDT", Type: "oco", Quantity: 1, StopLossPrice: 90,
		MoveStopToEntry: true,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, OffsetPercent: 5},
			{Percent: 75, OffsetPercent: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.MoveStopToEntry || math.Abs(got.MoveStopPrice-100) > 1e-6 {
		t.Fatalf("preview buy price: %+v", got.MoveStopPrice)
	}
	bracket, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-move", Symbol: "BTCUSDT", Type: "bracket", Quantity: 1,
		TriggerPrice: 100, StopLossPrice: 90, TakeProfitPrice: 120, MoveStopToEntry: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bracket.MoveStopToEntry || math.Abs(bracket.MoveStopPrice-100) > 1e-9 {
		t.Fatalf("bracket preview uses the entry limit: %+v", bracket.MoveStopPrice)
	}
}

func TestPortfolio_BracketMovesStopToAverageFill(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "br-move", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	entry, tps, sl, err := svc.PlaceBracketOrder(ctx, BracketOrderInput{
		ClientID: "br-move", Symbol: "BTCUSDT", Quantity: 1, EntryPrice: 100, StopLossPrice: 80,
		MoveStopToEntry: true,
		TakeProfitTargets: []domain.TakeProfitTarget{
			{Percent: 25, OffsetPercent: 5},
			{Percent: 25, OffsetPercent: 10},
			{Percent: 50, OffsetPercent: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *entry, 90, 0.4); err != nil || !ok {
		t.Fatalf("entry fill err=%v", err)
	}
	filledEntry, _ := svc.store.GetPendingOrder(ctx, "br-move", entry.ID)
	stop, _ := svc.store.GetPendingOrder(ctx, "br-move", sl.ID)
	if math.Abs(filledEntry.EntryPrice-90) > 1e-9 || math.Abs(stop.EntryPrice-90) > 1e-9 {
		t.Fatalf("average fill should be stored: entry=%v stop=%v", filledEntry.EntryPrice, stop.EntryPrice)
	}
	if math.Abs(stop.TriggerPrice-80) > 1e-9 {
		t.Fatalf("the stop waits for the first target: %+v", stop)
	}
	first, _ := svc.store.GetPendingOrder(ctx, "br-move", tps[0].ID)
	if _, ok, err := svc.TryFillPendingOrder(ctx, *first, 105, 0); err != nil || !ok {
		t.Fatalf("target fill err=%v", err)
	}
	stop, _ = svc.store.GetPendingOrder(ctx, "br-move", sl.ID)
	if math.Abs(stop.TriggerPrice-90) > 1e-9 || math.Abs(stop.RemainingQuantity-0.3) > 1e-6 {
		t.Fatalf("stop rises to the fill and shrinks: %+v", stop)
	}
	rest, _ := svc.store.GetPendingOrder(ctx, "br-move", entry.ID)
	if _, ok, err := svc.TryFillPendingOrder(ctx, *rest, 100, 0); err != nil || !ok {
		t.Fatalf("rest of entry err=%v", err)
	}
	stop, _ = svc.store.GetPendingOrder(ctx, "br-move", sl.ID)
	if math.Abs(stop.EntryPrice-96) > 1e-6 {
		t.Fatalf("later fills update the stored average: %+v", stop)
	}
	if math.Abs(stop.TriggerPrice-90) > 1e-9 {
		t.Fatalf("the stop does not move a second time: %+v", stop)
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
