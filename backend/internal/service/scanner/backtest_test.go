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
	again, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-snap", RuleID: rule.ID, Exchange: "binance", Symbol: "ETHUSDT",
		RangeStart: t0.Add(15 * 24 * time.Hour), RangeEnd: t0.Add(35 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID == job.ID {
		t.Fatal("changed rule must start a new backtest, not reuse the completed one")
	}
}

func ohlcToCandles(bars []domain.OHLC) []domain.Candle {
	out := make([]domain.Candle, len(bars))
	for i, b := range bars {
		out[i] = domain.Candle{
			OpenTime: b.OpenTime, CloseTime: b.CloseTime,
			Open: fmt.Sprintf("%.8f", b.Open), High: fmt.Sprintf("%.8f", b.High),
			Low: fmt.Sprintf("%.8f", b.Low), Close: fmt.Sprintf("%.8f", b.Close),
			Volume: fmt.Sprintf("%.4f", b.Volume),
		}
	}
	return out
}

func TestBacktest_CustomProfileStartAndRun(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	primOHLC := synthTrendService(220, 50, 0.8)
	dailyOHLC := synthTrendService(220, 40, 1.5)
	primary := ohlcToCandles(primOHLC)
	daily := ohlcToCandles(dailyOHLC)
	svc := New(st, &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|ETHUSDT|4h": primary,
		"binance|ETHUSDT|1d": daily,
		"binance|BTCUSDT|4h": daily,
		"binance|BTCUSDT|1d": daily,
	}}, &fakeWatch{})
	ctx := context.Background()

	start := primary[80].OpenTime
	end := primary[len(primary)-1].OpenTime
	job, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-custom", ProfileID: domain.CustomBacktestBuiltInID,
		Exchange: "binance", Symbol: "ETHUSDT", RangeStart: start, RangeEnd: end,
	})
	if err != nil || job.Kind != domain.BacktestKindCustom {
		t.Fatalf("%+v %v", job, err)
	}
	if !domain.IsCustomProfileBacktest(*job) {
		t.Fatal("queued job must be a custom snapshot")
	}
	dup, err := svc.StartBacktest(ctx, StartBacktestInput{
		ClientID: "bt-custom", ProfileID: domain.CustomBacktestBuiltInID,
		Exchange: "binance", Symbol: "ETHUSDT", RangeStart: start, RangeEnd: end,
	})
	if err != nil || dup.ID != job.ID {
		t.Fatalf("dedupe %v vs %v err=%v", job.ID, dup.ID, err)
	}

	(&BacktestWorker{Scanner: svc, Interval: time.Hour}).RunOnce(ctx)
	got, err := svc.GetBacktest(ctx, "bt-custom", job.ID)
	if err != nil || got.Status != domain.BacktestCompleted {
		t.Fatalf("status %+v err=%v", got, err)
	}
	want, err := domain.ReplayCustomSignalBacktest(domain.CustomBacktestInput{
		Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT",
		Primary: primOHLC, Higher: dailyOHLC, BTCPrimary: dailyOHLC, BTCHigher: dailyOHLC,
		RangeStart: start, RangeEnd: end, Weights: domain.DefaultCustomSignalWeights(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.SignalCount != len(want.Hits) {
		t.Fatalf("signals got %d want %d job=%+v", got.SignalCount, len(want.Hits), got)
	}
	if got.SkippedBars != want.SkippedBars {
		t.Fatalf("skipped got %d want %d", got.SkippedBars, want.SkippedBars)
	}
	sigs, total, err := svc.ListBacktestSignals(ctx, "bt-custom", job.ID, 50, 0)
	if err != nil || total != got.SignalCount || len(sigs) == 0 {
		t.Fatalf("sigs total=%d len=%d err=%v", total, len(sigs), err)
	}
	if _, ok := sigs[0].Metrics["score"]; !ok {
		t.Fatalf("metrics %+v", sigs[0].Metrics)
	}
	if len(sigs[0].Factors) == 0 {
		t.Fatal("persisted signal must keep factor scores and weights")
	}
	if sigs[0].Trade == nil || sigs[0].Trade.Entry <= 0 || sigs[0].Trade.Leverage < 1 {
		t.Fatalf("persisted trade %+v", sigs[0].Trade)
	}
	if got.Performance.Trades.SampleCount != got.SignalCount {
		t.Fatalf("trade stats %+v", got.Performance.Trades)
	}
	if got.Performance == nil || got.Performance.SignalCount != got.SignalCount {
		t.Fatalf("performance %+v", got.Performance)
	}
	if got.Performance.Return1d.Days != 1 || got.Performance.Return5d.Days != 5 || got.Performance.Return20d.Days != 20 {
		t.Fatalf("horizons %+v", got.Performance)
	}
}

func TestBacktest_CustomProfileMissing(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st, &fakeCandles{}, &fakeWatch{})
	_, err = svc.StartBacktest(context.Background(), StartBacktestInput{
		ClientID: "bt-miss", ProfileID: "no-such-profile",
		Exchange: "binance", Symbol: "ETHUSDT",
		RangeStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		RangeEnd:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected missing profile")
	}
}

func synthTrendService(n int, start, step float64) []domain.OHLC {
	out := make([]domain.OHLC, n)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	px := start
	for i := 0; i < n; i++ {
		o := px
		px += step * (1 + 0.05*float64((i%7)-3)/3)
		h, l := o, o
		if px > h {
			h = px
		}
		if px < l {
			l = px
		}
		out[i] = domain.OHLC{
			OpenTime:  t0.Add(time.Duration(i) * 4 * time.Hour),
			CloseTime: t0.Add(time.Duration(i+1) * 4 * time.Hour),
			Open:      o, High: h + step*0.4, Low: l - step*0.3, Close: px,
			Volume: 1000 + float64(i),
		}
	}
	out[n-1].Volume = 20000
	return out
}
