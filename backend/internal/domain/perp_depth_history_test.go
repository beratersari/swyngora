package domain

import (
	"strings"
	"testing"
	"time"
)

func TestAttachPerpDepthWindows_BidShareRisesAndNamesDrivers(t *testing.T) {
	now := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	live := BuildPerpDepthOverview("all", 1, []PerpDepthCoinInput{
		{Symbol: "ETHUSDT", Base: "ETH", Bid: 120, Ask: 20, Complete: true},
		{Symbol: "BTCUSDT", Base: "BTC", Bid: 140, Ask: 120, Complete: true},
	}, now)
	// Then: ETH 80/20 + BTC 140/160 = bid 220 / total 400 = 55%.
	// Now: ETH 120/20 + BTC 140/120 = bid 260 / total 400 = 65%.
	past := now.Add(-time.Hour)
	samples := []PerpDepthSample{
		{Time: past, Exchange: "all", RangePct: 1, Symbol: "ETHUSDT", Base: "ETH", Bid: 80, Ask: 20},
		{Time: past, Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Base: "BTC", Bid: 140, Ask: 160},
	}
	AttachPerpDepthWindows(live, samples)
	var hour PerpDepthShareWindow
	for _, w := range live.Windows {
		if w.Window == ParticipationWindow1h {
			hour = w
		}
	}
	if !hour.Complete {
		t.Fatalf("incomplete %+v", live.Windows)
	}
	if hour.PastBidPct < 54.5 || hour.PastBidPct > 55.5 {
		t.Fatalf("past bid %v", hour.PastBidPct)
	}
	if hour.BidPct < 64.5 || hour.BidPct > 65.5 {
		t.Fatalf("now bid %v", hour.BidPct)
	}
	if hour.ChangePct < 9.5 || hour.ChangePct > 10.5 {
		t.Fatalf("change %v", hour.ChangePct)
	}
	if hour.Kind != PerpDepthShareBidGained {
		t.Fatalf("kind %s", hour.Kind)
	}
	if hour.PastTotal != 400 || hour.Total != 400 || hour.TotalChange != 0 || hour.DepthKind != PerpDepthSizeSteady {
		t.Fatalf("size %+v", hour)
	}
	if !strings.Contains(hour.Summary, "about the same") {
		t.Fatalf("summary %s", hour.Summary)
	}
	if len(hour.Drivers) == 0 || hour.Drivers[0].Base != "ETH" || hour.Drivers[0].ChangePct < 9 {
		t.Fatalf("drivers %+v", hour.Drivers)
	}
	if !strings.Contains(hour.Summary, "55") || !strings.Contains(hour.Summary, "65") || !strings.Contains(hour.Summary, "ETH") {
		t.Fatalf("summary %s", hour.Summary)
	}
	if !strings.Contains(live.Summary, "55") {
		t.Fatalf("live summary %s", live.Summary)
	}
}

func TestAttachPerpDepthWindows_IncompleteWithoutPast(t *testing.T) {
	live := BuildPerpDepthOverview("all", 1, []PerpDepthCoinInput{
		{Symbol: "BTCUSDT", Base: "BTC", Bid: 80, Ask: 20, Complete: true},
	}, time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC))
	AttachPerpDepthWindows(live, nil)
	if len(live.Windows) != 3 {
		t.Fatalf("%+v", live.Windows)
	}
	for _, w := range live.Windows {
		if w.Complete {
			t.Fatalf("expected incomplete %+v", w)
		}
	}
}

func TestAttachPerpDepthWindows_TotalBookGetsThicker(t *testing.T) {
	now := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	live := BuildPerpDepthOverview("all", 1, []PerpDepthCoinInput{
		{Symbol: "BTCUSDT", Base: "BTC", Bid: 300, Ask: 200, Complete: true},
	}, now)
	past := now.Add(-15 * time.Minute)
	AttachPerpDepthWindows(live, []PerpDepthSample{
		{Time: past, Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Base: "BTC", Bid: 200, Ask: 200},
	})
	var win PerpDepthShareWindow
	for _, w := range live.Windows {
		if w.Window == ParticipationWindow15m {
			win = w
		}
	}
	if !win.Complete || win.PastTotal != 400 || win.Total != 500 {
		t.Fatalf("%+v", win)
	}
	if win.TotalChange != 100 || win.TotalChangePct != 25 || win.DepthKind != PerpDepthSizeThicker {
		t.Fatalf("change %+v", win)
	}
	if !strings.Contains(win.Summary, "thicker") || !strings.Contains(win.Summary, "+25") {
		t.Fatalf("summary %s", win.Summary)
	}
}

func TestClassifyPerpDepthSize(t *testing.T) {
	if ClassifyPerpDepthSize(5) != PerpDepthSizeThicker || ClassifyPerpDepthSize(-5) != PerpDepthSizeThinner {
		t.Fatal("sides")
	}
	if ClassifyPerpDepthSize(4.9) != PerpDepthSizeSteady {
		t.Fatal("steady")
	}
}

func TestClassifyPerpDepthShare(t *testing.T) {
	if ClassifyPerpDepthShare(10) != PerpDepthShareBidGained || ClassifyPerpDepthShare(-10) != PerpDepthShareAskGained {
		t.Fatal("sides")
	}
	if ClassifyPerpDepthShare(0.4) != PerpDepthShareStable {
		t.Fatal("stable")
	}
}
