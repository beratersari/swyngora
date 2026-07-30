package scanner

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeCandles struct {
	byKey map[string][]domain.Candle
}

func (f *fakeCandles) GetCandles(_ context.Context, exchange, symbol, interval string, limit int, _, _ *time.Time) ([]domain.Candle, error) {
	k := exchange + "|" + symbol + "|" + interval
	c := f.byKey[k]
	if len(c) > limit {
		return c[len(c)-limit:], nil
	}
	return c, nil
}

type fakeWatch struct {
	wl *domain.Watchlist
}

func (f *fakeWatch) Get(_ context.Context, clientID string) (*domain.Watchlist, error) {
	if f.wl == nil || f.wl.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	return f.wl, nil
}

func TestScanner_CreateRunDedupe(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Volume spike candles
	candles := make([]domain.Candle, 25)
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		vol := "100"
		if i == 24 {
			vol = "400"
		}
		candles[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    "10",
			Volume:   vol,
		}
	}
	market := &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|BTCUSDT|1h": candles,
	}}
	watch := &fakeWatch{wl: &domain.Watchlist{
		ClientID: "scan-user",
		Items: []domain.WatchlistItem{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"},
		},
	}}
	svc := New(st, market, watch)
	ctx := context.Background()

	rule, err := svc.Create(ctx, CreateInput{
		ClientID: "scan-user", Type: "volume_increase", Interval: "1h",
		VolumeLookback: 20, VolumeMinRatio: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.RunOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("first run n=%d err=%v", n, err)
	}
	// Same market data — no new result
	n, err = svc.RunOnce(ctx)
	if err != nil || n != 0 {
		t.Fatalf("dedupe n=%d err=%v", n, err)
	}
	list, total, err := svc.ListResults(ctx, "scan-user", 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("results total=%d list=%d err=%v", total, len(list), err)
	}
	if list[0].RuleID != rule.ID || list[0].Symbol != "BTCUSDT" {
		t.Fatalf("%+v", list[0])
	}

	// New bar → new result allowed
	candles2 := append([]domain.Candle{}, candles...)
	next := candles[24]
	next.OpenTime = candles[24].OpenTime.Add(time.Hour)
	next.Volume = "500"
	candles2 = append(candles2, next)
	market.byKey["binance|BTCUSDT|1h"] = candles2
	n, err = svc.RunOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("new bar n=%d err=%v", n, err)
	}
}

func TestScanner_CreateValidation(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st, &fakeCandles{}, &fakeWatch{})
	ctx := context.Background()
	_, err = svc.Create(ctx, CreateInput{ClientID: "u", Type: "nope"})
	if err == nil {
		t.Fatal("want invalid type")
	}
	_, err = svc.Create(ctx, CreateInput{
		ClientID: "u", Type: "rsi", RSICondition: "below", RSIThreshold: 30, RSIPeriod: 14,
	})
	if err != nil {
		t.Fatal(err)
	}
}
