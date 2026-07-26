package domain

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestEMA_SimpleSeed(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5}
	ema := EMA(closes, 3)
	if ema[0] != nil || ema[1] != nil {
		t.Fatalf("warm-up: %v %v", ema[0], ema[1])
	}
	if ema[2] == nil || math.Abs(*ema[2]-2) > 1e-9 {
		t.Fatalf("seed sma want 2 got %v", ema[2])
	}
	if ema[3] == nil || math.Abs(*ema[3]-3) > 1e-6 {
		t.Fatalf("ema3=%v", ema[3])
	}
}

func TestRSIWilder_ConstantPrice(t *testing.T) {
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = 100
	}
	rsi := RSIWilder(closes, 14)
	if rsi[14] == nil {
		t.Fatal("expected rsi")
	}
	if math.Abs(*rsi[14]-50) > 1e-6 {
		t.Fatalf("flat rsi=%v", *rsi[14])
	}
}

func TestRSIWilder_AllUp(t *testing.T) {
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	rsi := RSIWilder(closes, 14)
	if rsi[14] == nil || *rsi[14] < 99 {
		t.Fatalf("all-up rsi should be ~100, got %v", rsi[14])
	}
}

func TestBuildIndicatorSeries(t *testing.T) {
	candles := make([]Candle, 30)
	base := time.Unix(0, 0).UTC()
	for i := range candles {
		candles[i] = Candle{
			OpenTime: base.Add(time.Duration(i) * time.Hour),
			Close:    fmt.Sprintf("%g", 100+float64(i)*0.5),
		}
	}
	pts, err := BuildIndicatorSeries(candles, 14, []int{12, 26})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 30 {
		t.Fatalf("len=%d", len(pts))
	}
	if pts[14].RSI == nil {
		t.Fatal("want rsi at index 14")
	}
	if pts[11].EMA[12] == nil {
		t.Fatal("ema12 first at index 11")
	}
	if pts[25].EMA[26] == nil {
		t.Fatal("ema26 first at index 25")
	}
}

func TestNormalizeEMAPeriods(t *testing.T) {
	got := NormalizeEMAPeriods([]int{26, 12, 12, 1, 9999})
	if len(got) != 2 || got[0] != 12 || got[1] != 26 {
		t.Fatalf("%v", got)
	}
}

func TestValidateAndNormalizeEMAPeriods_RejectsInvalid(t *testing.T) {
	_, err := ValidateAndNormalizeEMAPeriods([]int{12, 1})
	if err == nil {
		t.Fatal("expected error for period 1")
	}
	got, err := ValidateAndNormalizeEMAPeriods([]int{26, 12, 12})
	if err != nil || len(got) != 2 || got[0] != 12 || got[1] != 26 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestBuildIndicatorSeries_InvalidCloseErrors(t *testing.T) {
	candles := []Candle{
		{OpenTime: time.Unix(1, 0).UTC(), Close: "10"},
		{OpenTime: time.Unix(2, 0).UTC(), Close: "bad"},
		{OpenTime: time.Unix(3, 0).UTC(), Close: "12"},
	}
	_, err := BuildIndicatorSeries(candles, 2, []int{2})
	if err == nil {
		t.Fatal("expected error for invalid close (must not collapse gaps)")
	}
}

func TestNormalizeSymbol_Coinbase(t *testing.T) {
	if got := NormalizeSymbol(ExchangeCoinbase, "btcusd"); got != "BTC-USD" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeSymbol(ExchangeBinance, "btc-usdt"); got != "BTCUSDT" {
		t.Fatalf("got %q", got)
	}
}
