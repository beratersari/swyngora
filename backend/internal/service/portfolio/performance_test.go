package portfolio

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetPerformance_PeriodPnLAndSeries(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "perf1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "perf1", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	}); err != nil {
		t.Fatal(err)
	}
	// Mark BTC up 10% → equity 10000 + 20 unrealized.
	px.prices["binance|BTCUSDT"] = "110"

	got, err := svc.GetPerformance(ctx, "perf1", "1w")
	if err != nil {
		t.Fatal(err)
	}
	if got.Period != domain.PerformancePeriod1W {
		t.Fatalf("period=%s", got.Period)
	}
	if math.Abs(got.StartEquity-10000) > 1e-6 {
		t.Fatalf("startEquity=%v", got.StartEquity)
	}
	if math.Abs(got.EndEquity-10020) > 0.5 {
		t.Fatalf("endEquity=%v", got.EndEquity)
	}
	if math.Abs(got.ChangeAmount-20) > 0.5 {
		t.Fatalf("change=%v", got.ChangeAmount)
	}
	if got.ChangePct == nil || math.Abs(*got.ChangePct-0.2) > 0.05 {
		t.Fatalf("pct=%v", got.ChangePct)
	}
	if got.PointCount < 1 || len(got.Points) == 0 {
		t.Fatal("want series points")
	}
	last := got.Points[len(got.Points)-1]
	if math.Abs(last.Equity-got.EndEquity) > 1e-9 {
		t.Fatalf("last point %v vs end %v", last.Equity, got.EndEquity)
	}
}

func TestGetPerformance_CashMovesAreNotProfit(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "perf-cash", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Deposit(ctx, CashMoveInput{ClientID: "perf-cash", Amount: 2500}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Withdraw(ctx, CashMoveInput{ClientID: "perf-cash", Amount: 500}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetPerformance(ctx, "perf-cash", "1w")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.EndEquity-12000) > 1e-6 {
		t.Fatalf("end=%v", got.EndEquity)
	}
	if math.Abs(got.ChangeAmount) > 1e-6 {
		t.Fatalf("cash moves counted as P&L: %v", got.ChangeAmount)
	}
	if math.Abs(got.NetFlow-2000) > 1e-6 {
		t.Fatalf("netFlow=%v", got.NetFlow)
	}
	if got.ChangePct == nil || math.Abs(*got.ChangePct) > 1e-6 {
		t.Fatalf("pct=%v", got.ChangePct)
	}
	if got.MaxDrawdown > 1e-6 {
		t.Fatalf("withdrawal counted as drawdown: %v", got.MaxDrawdown)
	}
	last := got.Points[len(got.Points)-1]
	if math.Abs((last.Equity-last.ExternalFlow)-got.StartEquity) > 1e-6 {
		t.Fatalf("performance line %v flow %v start %v", last.Equity, last.ExternalFlow, got.StartEquity)
	}

	// A real mark-to-market gain still counts after a later deposit.
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "perf-cash", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "110"
	if _, _, err := svc.Deposit(ctx, CashMoveInput{ClientID: "perf-cash", Amount: 1000}); err != nil {
		t.Fatal(err)
	}
	got, err = svc.GetPerformance(ctx, "perf-cash", "1w")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.ChangeAmount-20) > 0.5 {
		t.Fatalf("trading pnl=%v end=%v net=%v", got.ChangeAmount, got.EndEquity, got.NetFlow)
	}
	if math.Abs(got.NetFlow-3000) > 1e-6 {
		t.Fatalf("netFlow after second deposit=%v", got.NetFlow)
	}
}

func TestGetPerformance_TransferIsNotProfit(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	main, err := svc.Create(ctx, CreateInput{ClientID: "perf-xfer", StartingBalance: 10000, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := svc.Create(ctx, CreateInput{ClientID: "perf-xfer", StartingBalance: 1000, Name: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transfer(ctx, TransferInput{
		ClientID: "perf-xfer", FromPortfolioID: main.ID, ToPortfolioID: other.ID, Amount: 2500,
	}); err != nil {
		t.Fatal(err)
	}
	from, err := svc.GetPerformance(ctx, "perf-xfer", "1d", main.ID)
	if err != nil {
		t.Fatal(err)
	}
	to, err := svc.GetPerformance(ctx, "perf-xfer", "1d", other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(from.ChangeAmount) > 1e-6 || math.Abs(to.ChangeAmount) > 1e-6 {
		t.Fatalf("transfer pnl from=%v to=%v", from.ChangeAmount, to.ChangeAmount)
	}
	if math.Abs(from.NetFlow-(-2500)) > 1e-6 || math.Abs(to.NetFlow-2500) > 1e-6 {
		t.Fatalf("flows from=%v to=%v", from.NetFlow, to.NetFlow)
	}
	if from.MaxDrawdown > 1e-6 || to.MaxDrawdown > 1e-6 {
		t.Fatalf("drawdown from=%v to=%v", from.MaxDrawdown, to.MaxDrawdown)
	}
}

func TestGetPerformance_InvalidPeriod(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "perf2", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPerformance(ctx, "perf2", "1y"); err == nil {
		t.Fatal("want invalid period")
	}
}

func TestSnapshotWorker_RunOnce(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "snapw", StartingBalance: 5000}); err != nil {
		t.Fatal(err)
	}
	w := &SnapshotWorker{Portfolio: svc, Interval: time.Minute, Retention: 24 * time.Hour}
	w.RunOnce(ctx)
	n, err := svc.SnapshotAll(ctx, time.Now().UTC())
	if err != nil || n != 1 {
		t.Fatalf("snapshot n=%d err=%v", n, err)
	}
	deleted, err := svc.PruneSnapshots(ctx, time.Now().UTC().Add(time.Hour))
	if err != nil || deleted < 1 {
		t.Fatalf("prune deleted=%d err=%v", deleted, err)
	}
}
