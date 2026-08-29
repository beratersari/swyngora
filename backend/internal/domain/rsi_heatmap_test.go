package domain

import "testing"

func TestRSIZoneFor(t *testing.T) {
	low := 22.0
	mid := 51.0
	high := 81.0
	if RSIZoneFor(&low, 30, 70) != RSIZoneOversold {
		t.Fatal("22 should be oversold")
	}
	if RSIZoneFor(&mid, 30, 70) != RSIZoneNeutral {
		t.Fatal("51 should be neutral")
	}
	if RSIZoneFor(&high, 30, 70) != RSIZoneOverbought {
		t.Fatal("81 should be overbought")
	}
	if RSIZoneFor(nil, 30, 70) != RSIZoneUnknown {
		t.Fatal("nil rsi is unknown")
	}
	edge := 30.0
	if RSIZoneFor(&edge, 0, 0) != RSIZoneOversold {
		t.Fatal("non-positive bands must fall back to 30/70")
	}
}

func TestParseRSIHeatmapInterval(t *testing.T) {
	got, err := ParseRSIHeatmapInterval("", ExchangeBinance)
	if err != nil || got != "1h" {
		t.Fatalf("default=%q err=%v", got, err)
	}
	got, err = ParseRSIHeatmapInterval("4h,1d", ExchangeBinance)
	if err != nil || got != "4h" {
		t.Fatalf("first of list=%q err=%v", got, err)
	}
	got, err = ParseRSIHeatmapInterval("  ,1h", ExchangeBinance)
	if err != nil || got != "1h" {
		t.Fatalf("empty first token should default: %q err=%v", got, err)
	}
	if _, err := ParseRSIHeatmapInterval("3y", ExchangeBinance); err == nil {
		t.Fatal("expected invalid interval")
	}
}

func TestClampRSIHeatmapLimit(t *testing.T) {
	if ClampRSIHeatmapLimit(0) != RSIHeatmapDefaultLimit {
		t.Fatal("default")
	}
	if ClampRSIHeatmapLimit(999) != RSIHeatmapMaxLimit {
		t.Fatal("cap")
	}
}

func TestIsRSIHeatmapStableBase(t *testing.T) {
	if !IsRSIHeatmapStableBase("USDC") || !IsRSIHeatmapStableBase("usdt") {
		t.Fatal("stables should be excluded")
	}
	if IsRSIHeatmapStableBase("BTC") || IsRSIHeatmapStableBase("ETH") {
		t.Fatal("majors stay on the map")
	}
}

func TestSummarizeRSIHeatmap_Nil(t *testing.T) {
	SummarizeRSIHeatmap(nil)
}

func TestSummarizeRSIHeatmap(t *testing.T) {
	low, mid, high := 22.0, 51.0, 81.0
	h := &RSIHeatmap{Items: []RSIHeatmapRow{
		{Symbol: "AAA", RSI: &low, Zone: RSIZoneOversold},
		{Symbol: "BBB", RSI: &mid, Zone: RSIZoneNeutral},
		{Symbol: "CCC", RSI: &high, Zone: RSIZoneOverbought},
		{Symbol: "DDD"},
	}}
	SummarizeRSIHeatmap(h)
	if h.Items[0].Rank != 1 || h.Items[3].Rank != 4 {
		t.Fatalf("ranks=%v", h.Items)
	}
	if h.OversoldCount != 1 || h.NeutralCount != 1 || h.OverboughtCount != 1 {
		t.Fatalf("counts oversold=%d neutral=%d overbought=%d", h.OversoldCount, h.NeutralCount, h.OverboughtCount)
	}
	if h.AverageRSI == nil || *h.AverageRSI < 51 || *h.AverageRSI > 52 {
		t.Fatalf("avg=%v", h.AverageRSI)
	}
}

func TestLatestRSI_AllUp(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	got := LatestRSI(closes, 14)
	if got == nil || *got < 70 {
		t.Fatalf("all-up RSI=%v", got)
	}
	if LatestRSI(nil, 0) != nil {
		t.Fatal("empty series")
	}
	if LatestRSI([]float64{1, 2}, 0) != nil {
		t.Fatal("too short even with default period")
	}
}

func TestRSIHeatmapCacheKey(t *testing.T) {
	a := RSIHeatmapCacheKey(ExchangeBinance, "usdt", "marketCapCirculating", "1h", 14)
	b := RSIHeatmapCacheKey(ExchangeBinance, "USDT", "marketCapCirculating", "1h", 14)
	c := RSIHeatmapCacheKey(ExchangeBinance, "USDT", "marketCapCirculating", "4h", 14)
	if a != b {
		t.Fatalf("keys should match: %s vs %s", a, b)
	}
	if a == c {
		t.Fatal("different intervals must not share a key")
	}
}

func TestClipRSIHeatmap_NilAndEmpty(t *testing.T) {
	if ClipRSIHeatmap(nil, 10) != nil {
		t.Fatal("nil")
	}
	got := ClipRSIHeatmap(&RSIHeatmap{}, 5)
	if got == nil || len(got.Items) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestClipRSIHeatmap_SmallerTopFromLargerMap(t *testing.T) {
	low, high := 20.0, 80.0
	full := &RSIHeatmap{Items: []RSIHeatmapRow{
		{Symbol: "AAA", RSI: &low, Zone: RSIZoneOversold},
		{Symbol: "BBB", RSI: &high, Zone: RSIZoneOverbought},
		{Symbol: "CCC", RSI: &high, Zone: RSIZoneOverbought},
	}}
	SummarizeRSIHeatmap(full)
	got := ClipRSIHeatmap(full, 1)
	if len(got.Items) != 1 || got.Items[0].Symbol != "AAA" || got.Items[0].Rank != 1 {
		t.Fatalf("%+v", got.Items)
	}
	if got.OversoldCount != 1 || got.OverboughtCount != 0 {
		t.Fatalf("counts %+v", got)
	}
	if full.Items[0].Rank != 1 || len(full.Items) != 3 {
		t.Fatal("clip must not mutate the cached map")
	}
}

func TestRSIHeatmapFetchLimit(t *testing.T) {
	if RSIHeatmapFetchLimit(14) != RSIHeatmapCandleLimit {
		t.Fatalf("default period: %d", RSIHeatmapFetchLimit(14))
	}
	if RSIHeatmapFetchLimit(100) < 150 {
		t.Fatalf("long period should grow seed: %d", RSIHeatmapFetchLimit(100))
	}
}
