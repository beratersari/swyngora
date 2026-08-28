package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestBacktest_StartRunCancelDedupe(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	t0 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]domain.Candle, 50)
	for i := 0; i < 50; i++ {
		vol := "100"
		if i == 25 {
			vol = "500"
		}
		close := 10.0 + float64(i)*0.1
		candles[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * 24 * time.Hour),
			Close:    fmt.Sprintf("%.8f", close),
			Volume:   vol,
		}
	}
	market := &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|ETHUSDT|1d": candles,
	}}
	svc := New(st, market, &fakeWatch{})
	ctx := context.Background()

	rule, err := svc.Create(ctx, CreateInput{
		ClientID: "bt-user", Type: "volume_increase", Interval: "1d",
		VolumeLookback: 10, VolumeMinRatio: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	start := t0.Add(15 * 24 * time.Hour)
	end := t0.Add(40 * 24 * time.Hour)
	job, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-user", RuleID: rule.ID, Exchange: "binance", Symbol: "ETHUSDT",
		RangeStart: start, RangeEnd: end,
	})
	if err != nil || job.Status != domain.BacktestPending {
		t.Fatalf("%+v %v", job, err)
	}
	job2, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-user", RuleID: rule.ID, Exchange: "binance", Symbol: "ETHUSDT",
		RangeStart: start, RangeEnd: end,
	})
	if err != nil || job2.ID != job.ID {
		t.Fatalf("dedupe want same id: %v vs %v err=%v", job.ID, job2.ID, err)
	}

	w := &BacktestWorker{Scanner: svc, Interval: time.Hour}
	w.RunOnce(ctx)

	got, err := svc.GetBacktest(ctx, "bt-user", job.ID)
	if err != nil || got.Status != domain.BacktestCompleted {
		t.Fatalf("status %+v err=%v", got, err)
	}
	if got.SignalCount < 1 {
		t.Fatalf("expected signals: %+v", got)
	}
	sigs, total, err := svc.ListBacktestSignals(ctx, "bt-user", job.ID, 100, 0)
	if err != nil || total != got.SignalCount || len(sigs) == 0 {
		t.Fatalf("sigs total=%d len=%d err=%v", total, len(sigs), err)
	}
	if sigs[0].SignalAt.IsZero() {
		t.Fatal("missing signalAt")
	}

	job3, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-user", RuleID: rule.ID, Exchange: "binance", Symbol: "ETHUSDT",
		RangeStart: t0, RangeEnd: t0.Add(10 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	canceled, err := svc.CancelBacktest(ctx, "bt-user", job3.ID)
	if err != nil || canceled.Status != domain.BacktestCanceled {
		t.Fatalf("%+v %v", canceled, err)
	}
}

func TestBacktest_UsesQueuedRuleNotLaterEdit(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	t0 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]domain.Candle, 40)
	for i := 0; i < 40; i++ {
		vol := "100"
		if i == 25 {
			vol = "500"
		}
		candles[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * 24 * time.Hour),
			Close:    "10",
			Volume:   vol,
		}
	}
	svc := New(st, &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|ETHUSDT|1d": candles,
	}}, &fakeWatch{})
	ctx := context.Background()
	rule, err := svc.Create(ctx, CreateInput{
		ClientID: "bt-snap", Type: "volume_increase", Interval: "1d",
		VolumeLookback: 10, VolumeMinRatio: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-snap", RuleID: rule.ID, Exchange: "binance", Symbol: "ETHUSDT",
		RangeStart: t0.Add(15 * 24 * time.Hour), RangeEnd: t0.Add(35 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.RuleJSON == "" {
		t.Fatal("queued job must store a rule snapshot")
	}
	high := 99.0
	if _, err := svc.Update(ctx, UpdateInput{ClientID: "bt-snap", ID: rule.ID, VolumeMinRatio: &high}); err != nil {
		t.Fatal(err)
	}
	(&BacktestWorker{Scanner: svc, Interval: time.Hour}).RunOnce(ctx)
	got, err := svc.GetBacktest(ctx, "bt-snap", job.ID)
	if err != nil || got.Status != domain.BacktestCompleted {
		t.Fatalf("status %+v err=%v", got, err)
	}
	if got.SignalCount < 1 {
		t.Fatalf("edit after queue must not erase the original 2x spike: %+v", got)
	}
}
