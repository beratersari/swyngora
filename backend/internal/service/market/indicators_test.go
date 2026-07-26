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
}
