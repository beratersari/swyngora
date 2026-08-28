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
}

func TestRSIHeatmapCacheKey(t *testing.T) {
	a := RSIHeatmapCacheKey(ExchangeBinance, "usdt", "marketCapCirculating", "1h", 100, 14)
	b := RSIHeatmapCacheKey(ExchangeBinance, "USDT", "marketCapCirculating", "1h", 100, 14)
	c := RSIHeatmapCacheKey(ExchangeBinance, "USDT", "marketCapCirculating", "4h", 100, 14)
	if a != b {
		t.Fatalf("keys should match: %s vs %s", a, b)
	}
	if a == c {
		t.Fatal("different intervals must not share a key")
	}
}
