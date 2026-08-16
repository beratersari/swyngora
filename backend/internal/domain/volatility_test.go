package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func ohlc(start time.Time, n int, step time.Duration, closeFn func(i int) float64, wick float64) []OHLCBar {
	out := make([]OHLCBar, n)
	for i := 0; i < n; i++ {
		c := closeFn(i)
		out[i] = OHLCBar{
			Time: start.Add(time.Duration(i) * step),
			Open: c, High: c + wick, Low: c - wick, Close: c,
		}
	}
	return out
}

func TestMeasureVolatility_RangeAndNet(t *testing.T) {
	start := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	bars := []OHLCBar{
		{Time: start, Open: 100, High: 102, Low: 99, Close: 100},
		{Time: start.Add(time.Minute), Open: 100, High: 101, Low: 98, Close: 101},
	}
	got := MeasureVolatility(bars)
	if !got.Complete {
		t.Fatal("complete")
	}
	if math.Abs(got.NetPct-1) > 1e-9 {
		t.Fatalf("net %v", got.NetPct)
	}
	// high 102, low 98, ref 100 → 4%
	if math.Abs(got.RangePct-4) > 1e-9 {
		t.Fatalf("range %v", got.RangePct)
	}
}

func TestClassifyVolVsNormalAndTrend(t *testing.T) {
	if ClassifyVolVsNormal(2, 1, 4) != VolVsHigher {
		t.Fatal("higher")
	}
	if ClassifyVolVsNormal(0.5, 1, 4) != VolVsLower {
		t.Fatal("lower")
	}
	if ClassifyVolTrend(2, 1) != VolTrendExpanding {
		t.Fatal("expand")
	}
	if ClassifyVolTrend(0.7, 1) != VolTrendShrinking {
		t.Fatal("shrink")
	}
}

func TestClassifyVsMarket(t *testing.T) {
	if ClassifyVsMarket(2, 1) != VolVsMore {
		t.Fatal("more")
	}
	if ClassifyVsMarket(0.5, 1) != VolVsCalmer {
		t.Fatal("calmer")
	}
}

func TestBuildVolWindow_MoreVolatileThanBTC(t *testing.T) {
	start := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	n := 200
	// Quiet history, then a wide last hour; BTC stays quiet.
	asset := ohlc(start, n, time.Minute, func(i int) float64 {
		if i >= n-60 {
			return 20 + float64(i-(n-60))*0.08
		}
		return 20 + math.Sin(float64(i)/8)*0.02
	}, 0.01)
	for i := n - 60; i < n; i++ {
		asset[i].High = asset[i].Close + 0.4
		asset[i].Low = asset[i].Close - 0.4
	}
	btc := ohlc(start, n, time.Minute, func(i int) float64 { return 100 + math.Sin(float64(i)/10)*0.05 }, 0.02)
	eth := ohlc(start, n, time.Minute, func(i int) float64 { return 50 + math.Sin(float64(i)/10)*0.03 }, 0.01)
	got := BuildVolWindow("SOLUSDT", VolatilityWindows[0], asset, btc, eth)
	if !got.Coin.Complete || got.VsMarket != VolVsMore {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(strings.ToLower(got.Summary), "volatile") && !strings.Contains(strings.ToLower(got.Summary), "range") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestSplitVolWindows(t *testing.T) {
	start := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	bars := ohlc(start, 40, time.Minute, func(i int) float64 { return 10 + float64(i) }, 0.1)
	cur, prev, priors := SplitVolWindows(bars, 10)
	if len(cur) != 10 || len(prev) != 10 || len(priors) != 2 {
		t.Fatalf("cur=%d prev=%d priors=%d", len(cur), len(prev), len(priors))
	}
}

func TestExplainVolatilityReport(t *testing.T) {
	w := BuildVolWindow("SOLUSDT", VolatilityWindows[0],
		ohlc(time.Now().UTC().Add(-3*time.Hour), 180, time.Minute, func(i int) float64 { return 20 + float64(i%7)*0.2 }, 0.3),
		ohlc(time.Now().UTC().Add(-3*time.Hour), 180, time.Minute, func(i int) float64 { return 100 }, 0.01),
		ohlc(time.Now().UTC().Add(-3*time.Hour), 180, time.Minute, func(i int) float64 { return 50 }, 0.01),
	)
	got := ExplainVolatilityReport("SOLUSDT", []VolWindow{w})
	if !strings.Contains(got, "1h") {
		t.Fatalf("%s", got)
	}
}
