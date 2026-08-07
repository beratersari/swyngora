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
