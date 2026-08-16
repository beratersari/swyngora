package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestPriceVolumeWindows(t *testing.T) {
	now := time.Date(2026, 8, 16, 16, 0, 0, 0, time.UTC)
	bars := []OHLCBar{
		{Time: now.Add(-2 * time.Hour), Close: 100, QuoteVol: 10},
		{Time: now.Add(-90 * time.Minute), Close: 100, QuoteVol: 20},
		{Time: now.Add(-30 * time.Minute), Close: 102, QuoteVol: 40},
		{Time: now.Add(-5 * time.Minute), Close: 103, QuoteVol: 30},
	}
	got := PriceVolumeWindows(bars, now)
	var h1 SnapshotWindow
	for _, w := range got {
		if w.Window == SnapshotWindow1h {
			h1 = w
		}
	}
	if !h1.Price.Complete || math.Abs(h1.Price.Current-103) > 1e-9 {
		t.Fatalf("price %+v", h1.Price)
	}
	if h1.Volume.Current < 60 {
		t.Fatalf("vol %+v", h1.Volume)
	}
}

func TestApplyMarketCap(t *testing.T) {
	wins := []SnapshotWindow{{
		Window: SnapshotWindow1h,
		Price:  SnapshotChange{Window: SnapshotWindow1h, ChangePct: 10, Direction: "up", Complete: true},
	}}
	ApplyMarketCap(wins, 100, 2) // mcap 200
	if !wins[0].MarketCap.Complete || math.Abs(wins[0].MarketCap.Current-200) > 1e-9 {
		t.Fatalf("%+v", wins[0].MarketCap)
	}
	if math.Abs(wins[0].MarketCap.ChangePct-10) > 1e-9 {
		t.Fatalf("mcap pct %+v", wins[0].MarketCap)
	}
}

func TestBuildSnapshotVenue_LeadBeforePrice(t *testing.T) {
	now := time.Date(2026, 8, 16, 16, 0, 0, 0, time.UTC)
	spot := []SnapshotWindow{{
		Window: SnapshotWindow1h,
		Price:  SnapshotChange{Window: SnapshotWindow1h, ChangePct: 0.01, Direction: "flat", Complete: true},
		Volume: SnapshotChange{Window: SnapshotWindow1h, Direction: "up", Complete: true},
	}}
	oi := &OpenInterestSeries{
		Current: OpenInterestPoint{Time: now, Contracts: 120, Value: 12000},
		History: []OpenInterestPoint{{Time: now.Add(-time.Hour), Contracts: 100, Value: 10000}},
	}
	taker := &TakerVenueFlow{Windows: []TakerWindowFlow{
		SummarizeTakerWindow(80, 20, TakerWindow1h, true),
	}}
	got := BuildSnapshotVenue(ExchangeBinance, spot, oi, nil, nil, taker, now)
	if !strings.Contains(strings.ToLower(got.Summary), "before") && !strings.Contains(strings.ToLower(got.Summary), "rising") {
		t.Fatalf("summary %s", got.Summary)
	}
	if got.Windows[0].OI.Direction != "up" {
		t.Fatalf("oi %+v", got.Windows[0].OI)
	}
}

func TestChangeFromValues(t *testing.T) {
	got := ChangeFromValues("1h", 110, 100, true)
	if got.Direction != "up" || math.Abs(got.ChangePct-10) > 1e-9 || !got.Complete {
		t.Fatalf("%+v", got)
	}
}
