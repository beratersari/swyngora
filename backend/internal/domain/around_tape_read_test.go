package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBuildAroundTapeRead_BuyVsSellPocDominanceAndDeltaFlip(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 13, 0, 0, 0, time.UTC)
	var bars []AroundBar
	// Climb 65400 → 65900. Heavy two-way flow around 65520, then sellers take over.
	prices := []float64{
		65400, 65440, 65480, 65520, 65520, 65520, 65560, 65600,
		65650, 65700, 65720, 65750, 65800, 65820, 65850, 65880, 65900, 65940, 65980,
	}
	for i, px := range prices {
		buy, sell := 200_000.0, 180_000.0
		switch {
		case px == 65520:
			buy, sell = 4_000_000, 1_000_000
		case px >= 65700 && px < 65800:
			buy, sell = 120_000, 700_000
		case px >= 65800:
			buy, sell = 80_000, 1_800_000
		}
		bars = append(bars, AroundBar{
			Time: t0.Add(time.Duration(i) * time.Minute),
			Open: px, High: px + 20, Low: px - 20, Close: px,
			Volume: buy + sell, BuyVolume: buy, SellVolume: sell, BuySellKnown: true,
		})
	}
	from, to := t0, t0.Add(time.Duration(len(prices))*time.Minute)
	got := BuildAroundTapeRead(bars, from, to)
	if got == nil || !got.Complete {
		t.Fatal(got)
	}
	if got.POC < 65400 || got.POC > 65600 {
		t.Fatalf("poc %v", got.POC)
	}
	if got.POCBuy <= got.POCSell {
		t.Fatalf("poc flow buy=%v sell=%v", got.POCBuy, got.POCSell)
	}
	if got.DominantSide != TakerSideSell || got.DominantFrom < 65600 {
		t.Fatalf("dominance %+v", got)
	}
	if got.DeltaFlipKind != "negative" || got.Divergence != AroundTapeDivPriceUpDeltaDown {
		t.Fatalf("flip %+v", got)
	}
	if !strings.Contains(got.Summary, "market buys") || !strings.Contains(got.Summary, "seller dominance") {
		t.Fatalf("summary %s", got.Summary)
	}
	if !strings.Contains(got.Summary, "still going up") {
		t.Fatalf("divergence summary %s", got.Summary)
	}
}

func TestBuildAroundTapeRead_NeedsBuySellBars(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 13, 0, 0, 0, time.UTC)
	var bars []AroundBar
	for i := 0; i < 12; i++ {
		bars = append(bars, AroundBar{
			Time: t0.Add(time.Duration(i) * time.Minute),
			Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000,
		})
	}
	if got := BuildAroundTapeRead(bars, t0, t0.Add(20*time.Minute)); got != nil {
		t.Fatalf("%+v", got)
	}
}
