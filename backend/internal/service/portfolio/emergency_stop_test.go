package portfolio

import (
	"context"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestEmergencyStop_FlattensAndReportsEachStep(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "stop", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "stop", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 50,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "stop", Exchange: "binance", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 0.1, TriggerPrice: 10,
	}); err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(24 * time.Hour)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "stop", Symbol: "BTCUSDT", Amount: 25, Frequency: "daily", StartAt: &start,
	})
	if err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "stop", Symbol: "BTCUSDT", Side: "long", Type: "market", Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetEmergencyStop(ctx, EmergencyStopInput{ClientID: "stop", Enabled: true, LossPct: 10}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "40"
	if _, _, err := svc.Deposit(ctx, CashMoveInput{ClientID: "stop", Amount: 50000}); err != nil {
		t.Fatal(err)
	}
	n, err := svc.ProcessEmergencyStops(ctx)
	if err != nil || n != 1 {
		t.Fatalf("tripped n=%d err=%v", n, err)
	}
	view, err := svc.GetEmergencyStop(ctx, "stop")
	if err != nil || view.Run == nil || !view.Tripped || !view.Hit {
		t.Fatalf("view %+v err %v", view, err)
	}
	var failed []string
	seen := map[string]bool{}
	for _, a := range view.Run.Actions {
		seen[a.Kind] = true
		if a.Status != domain.EmergencyOK {
			failed = append(failed, a.Kind+":"+a.Detail)
		}
	}
	if len(failed) > 0 {
		t.Fatalf("failed %v", failed)
	}
	for _, kind := range []string{domain.EmergencyCancelOrder, domain.EmergencyPausePlan, domain.EmergencyCloseSpot, domain.EmergencyCloseMargin} {
		if !seen[kind] {
			t.Fatalf("missing %s in %+v", kind, view.Run.Actions)
		}
	}
	positions, err := svc.store.ListPositions(ctx, "stop")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range positions {
		if p.Quantity > domain.PositionEpsilon {
			t.Fatalf("spot still open %+v", p)
		}
	}
	margin, err := svc.ListMarginPositions(ctx, "stop")
	if err != nil || len(margin) != 0 {
		t.Fatalf("margin %+v err %v", margin, err)
	}
	orders, err := svc.ListPendingOrders(ctx, "stop", "open", 20, 0)
	if err != nil || len(orders) != 0 {
		t.Fatalf("orders %+v err %v", orders, err)
	}
	paused, err := svc.GetRecurringBuyPlan(ctx, "stop", plan.ID)
	if err != nil || paused.Status != domain.RecurringBuyPaused {
		t.Fatalf("plan %+v err %v", paused, err)
	}
	_ = pos
	tradesBefore, _, err := svc.ListTrades(ctx, "stop", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := svc.ProcessEmergencyStops(ctx); err != nil || n != 0 {
		t.Fatalf("second pass n=%d err=%v", n, err)
	}
	tradesAfter, _, err := svc.ListTrades(ctx, "stop", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradesAfter) != len(tradesBefore) {
		t.Fatalf("retried a close that already worked: %d -> %d", len(tradesBefore), len(tradesAfter))
	}
}

func TestEmergencyStop_NamesAFailedCloseAndRetriesIt(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	books := map[string]*domain.RawOrderBook{
		"binance": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
	}
	svc.SetBookSource(booksByVenue{by: books})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "stop-fail", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "stop-fail", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 4,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetEmergencyStop(ctx, EmergencyStopInput{ClientID: "stop-fail", Enabled: true, LossPct: 5}); err != nil {
		t.Fatal(err)
	}
	books["binance"] = &domain.RawOrderBook{Asks: []domain.PriceLevel{{Price: 40, Quantity: 5}}}
	px.prices["binance|BTCUSDT"] = "40"
	if _, err := svc.ProcessEmergencyStops(ctx); err != nil {
		t.Fatal(err)
	}
	view, err := svc.GetEmergencyStop(ctx, "stop-fail")
	if err != nil || view.Run == nil {
		t.Fatalf("%+v %v", view, err)
	}
	var closeFailed bool
	for _, a := range view.Run.Actions {
		if a.Kind == domain.EmergencyCloseSpot && a.Status == domain.EmergencyFailed && a.Detail != "" {
			closeFailed = true
		}
	}
	if !closeFailed {
		t.Fatalf("expected a named spot failure: %+v", view.Run.Actions)
	}
	runID := view.Run.ID
	books["binance"] = &domain.RawOrderBook{
		Asks: []domain.PriceLevel{{Price: 40, Quantity: 5}},
		Bids: []domain.PriceLevel{{Price: 40, Quantity: 5}},
	}
	if _, err := svc.ProcessEmergencyStops(ctx); err != nil {
		t.Fatal(err)
	}
	view, err = svc.GetEmergencyStop(ctx, "stop-fail")
	if err != nil || view.Run == nil || view.Run.ID != runID {
		t.Fatalf("run changed %+v err %v", view, err)
	}
	for _, a := range view.Run.Actions {
		if a.Kind == domain.EmergencyCloseSpot && a.Status != domain.EmergencyOK {
			t.Fatalf("retry still failed %+v", a)
		}
	}
	positions, _ := svc.store.ListPositions(ctx, "stop-fail")
	for _, p := range positions {
		if p.Symbol == "BTCUSDT" && p.Quantity > domain.PositionEpsilon {
			t.Fatalf("still open %+v", p)
		}
	}
}

func TestEmergencyStop_OffDoesNotClose(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "stop-off", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "stop-off", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 50,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetEmergencyStop(ctx, EmergencyStopInput{ClientID: "stop-off", Enabled: false, LossPct: 1}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "10"
	if n, err := svc.ProcessEmergencyStops(ctx); err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	positions, err := svc.store.ListPositions(ctx, "stop-off")
	if err != nil || len(positions) != 1 || positions[0].Quantity < 1 {
		t.Fatalf("%+v %v", positions, err)
	}
	view, err := svc.GetEmergencyStop(ctx, "stop-off")
	if err != nil || view.Enabled || view.Run != nil {
		t.Fatalf("%+v %v", view, err)
	}
}

func TestEmergencyStop_EnabledWithoutPercentIsRejected(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "stop-bad", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.SetEmergencyStop(ctx, EmergencyStopInput{ClientID: "stop-bad", Enabled: true})
	if err == nil || !strings.Contains(err.Error(), "lossPct") {
		t.Fatalf("%v", err)
	}
}
