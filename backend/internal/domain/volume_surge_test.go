package domain

import (
	"strings"
	"testing"
	"time"
)

func surgeBar(t0 time.Time, i int, vol, buy float64) VolumeBar {
	return VolumeBar{
		Time:   t0.Add(time.Duration(i) * 5 * time.Minute),
		Volume: vol, BuyVolume: buy, SellVolume: vol - buy, BuySellKnown: true,
	}
}

func TestMeasureVolumeSurge_FiveMinuteSpike(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	var bars []VolumeBar
	for i := 0; i < 24; i++ {
		bars = append(bars, surgeBar(t0, i, 2_000_000, 1_000_000))
	}
	bars = append(bars, surgeBar(t0, 24, 10_000_000, 8_000_000))
	got := MeasureVolumeSurge(bars)
	var w5 VolumeSurgeWindow
	for _, w := range got {
		if w.Window == VolumeSurgeWindow5m {
			w5 = w
		}
	}
	if !w5.Complete || w5.Typical != 2_000_000 || w5.Current != 10_000_000 {
		t.Fatalf("5m %+v", w5)
	}
	if w5.Ratio < 4.9 || w5.Ratio > 5.1 {
		t.Fatalf("ratio %v", w5.Ratio)
	}
	if w5.Grade != VolumeSurgeHigh {
		t.Fatalf("grade %s", w5.Grade)
	}
	if w5.BuyRatio < 7.9 || w5.Dominant != TakerSideBuy {
		t.Fatalf("buy side %+v", w5)
	}
}

func TestMeasureVolumeSurge_SellSideOnly(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	var bars []VolumeBar
	for i := 0; i < 20; i++ {
		bars = append(bars, surgeBar(t0, i, 1_000_000, 500_000))
	}
	bars = append(bars, surgeBar(t0, 20, 4_000_000, 500_000)) // sell 3.5M vs typical 0.5M
	got := MeasureVolumeSurge(bars)
	var w5 VolumeSurgeWindow
	for _, w := range got {
		if w.Window == VolumeSurgeWindow5m {
			w5 = w
		}
	}
	if w5.Dominant != TakerSideSell || w5.SellRatio < 6 {
		t.Fatalf("%+v", w5)
	}
}

func TestMeasureVolumeSurge_FifteenMinuteBucket(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	var bars []VolumeBar
	// 8 full 15m buckets of 3×1M = 3M, then one 15m of 3×4M = 12M.
	for i := 0; i < 27; i++ {
		vol := 1_000_000.0
		if i >= 24 {
			vol = 4_000_000
		}
		bars = append(bars, surgeBar(t0, i, vol, vol/2))
	}
	got := MeasureVolumeSurge(bars)
	var w15 VolumeSurgeWindow
	for _, w := range got {
		if w.Window == VolumeSurgeWindow15m {
			w15 = w
		}
	}
	if !w15.Complete || w15.Typical != 3_000_000 || w15.Current != 12_000_000 {
		t.Fatalf("15m %+v", w15)
	}
	if w15.Ratio < 3.9 || w15.Ratio > 4.1 {
		t.Fatalf("15m ratio %v", w15.Ratio)
	}
}

func TestVolumeSurgeGrade(t *testing.T) {
	if VolumeSurgeGrade(1.2) != VolumeSurgeTypical {
		t.Fatal("1.2")
	}
	if VolumeSurgeGrade(2) != VolumeSurgeElevated {
		t.Fatal("2")
	}
	if VolumeSurgeGrade(4) != VolumeSurgeHigh {
		t.Fatal("4")
	}
	if VolumeSurgeGrade(8) != VolumeSurgeExtreme {
		t.Fatal("8")
	}
}

func TestBuildVolumeSurgeVenue_Summary(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	var bars []VolumeBar
	for i := 0; i < 20; i++ {
		bars = append(bars, surgeBar(t0, i, 2_000_000, 1_000_000))
	}
	bars = append(bars, surgeBar(t0, 20, 10_000_000, 8_000_000))
	got := BuildVolumeSurgeVenue(ExchangeBinance, "BTCUSDT", bars)
	if got.MaxRatio < 4.9 || got.Hottest != VolumeSurgeWindow5m {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "5m") || !strings.Contains(got.Summary, "buy") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestVolumeBarsFromCandles(t *testing.T) {
	bars := VolumeBarsFromCandles([]Candle{
		{OpenTime: time.Unix(1, 0).UTC(), QuoteVolume: "100", TakerBuyQuote: "70"},
		{OpenTime: time.Unix(2, 0).UTC(), QuoteVolume: "50"},
	})
	if len(bars) != 2 || bars[0].BuyVolume != 70 || !bars[0].BuySellKnown {
		t.Fatalf("%+v", bars)
	}
	if bars[1].BuySellKnown {
		t.Fatalf("unknown %+v", bars[1])
	}
}

func TestExplainVolumeSurgeScan(t *testing.T) {
	empty := ExplainVolumeSurgeScan(VolumeSurgeScan{MinRatio: 2, SymbolLimit: 30})
	if !strings.Contains(empty, "No coins") {
		t.Fatalf("%s", empty)
	}
	got := ExplainVolumeSurgeScan(VolumeSurgeScan{
		MinRatio: 2, SymbolLimit: 30,
		Hits: []VolumeSurgeHit{{Symbol: "ETHUSDT", MaxRatio: 5, Hottest: "5m", Grade: VolumeSurgeHigh}},
	})
	if !strings.Contains(got, "ETH") || !strings.Contains(got, "5.0x") && !strings.Contains(got, "5x") {
		t.Fatalf("%s", got)
	}
}
