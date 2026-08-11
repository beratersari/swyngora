package domain

import (
	"math"
	"testing"
	"time"
)

func synthTrend(n int, start, step float64) []OHLC {
	out := make([]OHLC, n)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	px := start
	for i := 0; i < n; i++ {
		o := px
		px += step * (1 + 0.05*math.Sin(float64(i)/5))
		h := math.Max(o, px) + step*0.4
		l := math.Min(o, px) - step*0.3
		out[i] = OHLC{
			OpenTime: t0.Add(time.Duration(i) * 4 * time.Hour),
			CloseTime: t0.Add(time.Duration(i+1) * 4 * time.Hour),
			Open: o, High: h, Low: l, Close: px, Volume: 1000 + float64(i),
		}
	}
	return out
}

func synthChop(n int, mid float64) []OHLC {
	out := make([]OHLC, n)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		px := mid + 2*math.Sin(float64(i)/2)
		out[i] = OHLC{
			OpenTime: t0.Add(time.Duration(i) * 4 * time.Hour),
			CloseTime: t0.Add(time.Duration(i+1) * 4 * time.Hour),
			Open: px, High: px + 0.8, Low: px - 0.8, Close: px + 0.1*math.Cos(float64(i)),
			Volume: 800,
		}
	}
	return out
}

func TestClosedBarsDropsForming(t *testing.T) {
	bars := synthTrend(5, 100, 1)
	now := bars[len(bars)-1].CloseTime.Add(-time.Minute)
	got := ClosedBars(bars, now)
	if len(got) != 4 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestATRPositiveOnTrend(t *testing.T) {
	bars := synthTrend(80, 100, 0.5)
	atr, ok := ATR(bars, 14)
	if !ok || atr <= 0 {
		t.Fatalf("atr=%v ok=%v", atr, ok)
	}
}

func TestADXTrendHigherThanChop(t *testing.T) {
	trend := synthTrend(120, 100, 1.2)
	chop := synthChop(120, 100)
	ta, _, _, ok1 := ADX(trend, 14)
	ca, _, _, ok2 := ADX(chop, 14)
	if !ok1 || !ok2 {
		t.Fatalf("adx ok trend=%v chop=%v", ok1, ok2)
	}
	if ta <= ca {
		t.Fatalf("expected trend ADX > chop, got %v vs %v", ta, ca)
	}
}

func TestEvaluateSwing_UptrendTriggerOrWatch(t *testing.T) {
	// Strong uptrend with a late acceleration so EMA/MACD/volume can fire fresh.
	bars := synthTrend(220, 50, 0.8)
	// boost last volume for breakout
	bars[len(bars)-1].Volume = 20000
	for i := 0; i < 8; i++ {
		bars[len(bars)-2-i].High = bars[len(bars)-2-i].Close
	}
	daily := synthTrend(220, 40, 1.5)
	dec, err := EvaluateSwing(SwingScanInput{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT",
		Primary: bars, Higher: daily,
		BTCPrimary: daily, BTCHigher: daily,
		QuoteVolume: 50_000_000,
		Now:         bars[len(bars)-1].CloseTime.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Price <= 0 {
		t.Fatal("price")
	}
	if len(dec.Patterns) == 0 {
		t.Fatal("expected patterns")
	}
	if dec.Levels == nil {
		t.Fatal("expected levels")
	}
	if dec.Levels.StopLoss >= dec.Levels.Entry {
		t.Fatalf("stop %+v", dec.Levels)
	}
	if dec.Levels.TakeProfit <= dec.Levels.Entry {
		t.Fatalf("tp %+v", dec.Levels)
	}
	if dec.Levels.RR < 1.4 {
		t.Fatalf("rr too low %+v", dec.Levels)
	}
}

func TestEvaluateSwing_ChopRejectedForTrend(t *testing.T) {
	chop := synthChop(120, 100)
	dec, err := EvaluateSwing(SwingScanInput{
		Exchange: ExchangeBinance, Symbol: "ALTUSDT",
		Primary: chop, Higher: chop,
		BTCPrimary: chop, BTCHigher: chop,
		QuoteVolume: 5_000_000,
		Now:         chop[len(chop)-1].CloseTime.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Accepted && dec.SetupType == SwingSetupTrendPullback && dec.BTCRegime == SwingRegimeChop {
		t.Fatalf("chop should not accept trend setup: %+v", dec)
	}
}

func TestEvaluateSwing_TooFewBars(t *testing.T) {
	bars := synthTrend(10, 100, 1)
	dec, err := EvaluateSwing(SwingScanInput{Primary: bars, Now: bars[len(bars)-1].CloseTime.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Accepted || len(dec.Reasons) == 0 {
		t.Fatalf("%+v", dec)
	}
}

func TestEvaluateSwing_IlliquidRejected(t *testing.T) {
	bars := synthTrend(80, 100, 0.4)
	dec, err := EvaluateSwing(SwingScanInput{
		Primary: bars, Higher: bars, QuoteVolume: 100, // tiny
		Now: bars[len(bars)-1].CloseTime.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range dec.Reasons {
		if len(r) >= 12 && r[:12] == "quote_volume" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons=%v", dec.Reasons)
	}
}

func TestClassifyBTCRegimeBull(t *testing.T) {
	bars := synthTrend(220, 20, 1)
	got := ClassifyBTCRegime(bars, bars, nil)
	if got != SwingRegimeBull {
		t.Fatalf("got %s", got)
	}
}

func TestPlanLevelsRR(t *testing.T) {
	bars := synthTrend(40, 100, 0.3)
	atr, ok := ATR(bars, 14)
	if !ok {
		t.Fatal("atr")
	}
	lv, reasons := planSwingLevels(bars, atr, bars[len(bars)-1].Close)
	if lv == nil || len(reasons) > 0 {
		t.Fatalf("%+v %v", lv, reasons)
	}
	if lv.RR+1e-9 < SwingMinRRWatch {
		t.Fatalf("rr=%v", lv.RR)
	}
}

func TestMACDCross(t *testing.T) {
	// down then up so histogram can turn
	n := 80
	closes := make([]float64, n)
	for i := 0; i < 40; i++ {
		closes[i] = 100 - float64(i)*0.4
	}
	for i := 40; i < n; i++ {
		closes[i] = closes[39] + float64(i-39)*0.8
	}
	_, _, hist := MACD(closes, 12, 26, 9)
	if lastPtr(hist) == nil {
		t.Fatal("hist")
	}
}

func TestParseOHLCAndClosed(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	cs := []Candle{
		{OpenTime: now.Add(-2 * time.Hour), CloseTime: now.Add(-time.Hour), Open: "1", High: "2", Low: "0.5", Close: "1.5", Volume: "10"},
		{OpenTime: now.Add(-time.Hour), CloseTime: now.Add(time.Hour), Open: "1.5", High: "2", Low: "1", Close: "1.6", Volume: "11"},
	}
	bars, err := ParseOHLC(cs)
	if err != nil || len(bars) != 2 {
		t.Fatalf("%v %v", bars, err)
	}
	got := ClosedBars(bars, now)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
}
