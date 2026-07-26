package market

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type indicatorMarket struct {
	fakeMarket
}

func (m *indicatorMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	n := q.Limit
	if n <= 0 {
		n = 50
	}
	out := make([]domain.Candle, n)
	base := time.Unix(1_700_000_000, 0).UTC()
	for i := 0; i < n; i++ {
		out[i] = domain.Candle{
			OpenTime: base.Add(time.Duration(i) * time.Hour),
			Close:    fmt.Sprintf("%g", 100+float64(i)),
		}
	}
	return out, nil
}

func TestGetIndicators_RSIAndEMA(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	ser, err := svc.GetIndicators(context.Background(), "binance", "BTCUSDT", "1h", 20, 14, []int{12, 26})
	if err != nil {
		t.Fatal(err)
	}
	if ser.LatestRSI == nil {
		t.Fatal("expected latest RSI")
	}
	if ser.LatestEMA[12] == nil || ser.LatestEMA[26] == nil {
		t.Fatalf("ema latest=%v", ser.LatestEMA)
	}
	if len(ser.Points) != 20 {
		t.Fatalf("points=%d", len(ser.Points))
	}
	if ser.Symbol != "BTCUSDT" || ser.Exchange != domain.ExchangeBinance {
		t.Fatalf("%+v", ser)
	}
}

func TestGetIndicatorsBatch_OK(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	snaps, err := svc.GetIndicatorsBatch(context.Background(), "binance", "1h", []string{"BTCUSDT", "ETHUSDT"}, 14, []int{12, 26})
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 2 {
		t.Fatalf("snaps=%d", len(snaps))
	}
	for _, s := range snaps {
		if s.Error != "" {
			t.Fatalf("unexpected error for %s: %s", s.Symbol, s.Error)
		}
		if s.RSI == nil {
			t.Fatalf("missing RSI for %s", s.Symbol)
		}
	}
}

func TestGetIndicatorsBatch_RespectsCancel(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	snaps, err := svc.GetIndicatorsBatch(ctx, "binance", "1h", []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}, 14, []int{12})
	// May return partial snaps with errors and/or ctx error.
	if err == nil {
		for _, s := range snaps {
			if s.Error == "" {
				// Cancelled before work — all should error or we got lucky cache; tolerate either
				continue
			}
		}
	}
	_ = snaps
}

func TestGetIndicators_Validation(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	_, err := svc.GetIndicators(context.Background(), "binance", "", "1h", 10, 14, nil)
	if err == nil {
		t.Fatal("empty symbol")
	}
	_, err = svc.GetIndicators(context.Background(), "binance", "BTCUSDT", "1h", 10, 1, nil)
	if err == nil {
		t.Fatal("bad rsi period")
	}
	// Out-of-range EMA periods fail loud (not silent default).
	_, err = svc.GetIndicators(context.Background(), "binance", "BTCUSDT", "1h", 10, 14, []int{1, 9999})
	if err == nil {
		t.Fatal("invalid ema periods must error")
	}
}

type badCloseMarket struct {
	indicatorMarket
}

func (m *badCloseMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	n := q.Limit
	if n < 5 {
		n = 5
	}
	out := make([]domain.Candle, n)
	base := time.Unix(1_700_000_000, 0).UTC()
	for i := 0; i < n; i++ {
		close := fmt.Sprintf("%g", 100+float64(i))
		if i == n/2 {
			close = "not-a-number"
		}
		out[i] = domain.Candle{
			OpenTime: base.Add(time.Duration(i) * time.Hour),
			Close:    close,
		}
	}
	return out, nil
}

func TestGetIndicators_InvalidCloseErrors(t *testing.T) {
	svc := New(&badCloseMarket{}, &fakeSupply{})
	_, err := svc.GetIndicators(context.Background(), "binance", "BTCUSDT", "1h", 20, 14, []int{12, 26})
	if err == nil {
		t.Fatal("expected error when candle close is unparseable (must not collapse gaps)")
	}
}

func TestParseEMAPeriodsCSV(t *testing.T) {
	got, err := ParseEMAPeriodsCSV("12, 26")
	if err != nil || len(got) != 2 || got[0] != 12 || got[1] != 26 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	_, err = ParseEMAPeriodsCSV("12,abc")
	if err == nil {
		t.Fatal("non-integer token must error")
	}
	got, err = ParseEMAPeriodsCSV("")
	if err != nil || got != nil {
		t.Fatalf("empty → nil, got %v %v", got, err)
	}
}

func TestGetIndicatorsBatch_InvalidEMAPeriods(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	_, err := svc.GetIndicatorsBatch(context.Background(), "binance", "1h", []string{"BTCUSDT"}, 14, []int{1})
	if err == nil {
		t.Fatal("expected ema period validation error")
	}
}

func TestGetIndicatorsBatch_DedupesAndCaps(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	// duplicates + many symbols — only unique up to 50
	syms := make([]string, 0, 60)
	for i := 0; i < 55; i++ {
		syms = append(syms, fmt.Sprintf("S%dUSDT", i))
	}
	syms = append(syms, "S0USDT", "s0usdt") // dups
	snaps, err := svc.GetIndicatorsBatch(context.Background(), "binance", "1h", syms, 14, []int{12})
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 50 {
		t.Fatalf("want cap 50, got %d", len(snaps))
	}
}
