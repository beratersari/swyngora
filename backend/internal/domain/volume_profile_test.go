package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestIntervalDuration(t *testing.T) {
	if IntervalDuration(Interval5m) != 5*time.Minute {
		t.Fatalf("5m=%s", IntervalDuration(Interval5m))
	}
	if IntervalDuration(Interval1h) != time.Hour {
		t.Fatalf("1h=%s", IntervalDuration(Interval1h))
	}
}

func TestParseVolumeProfileWindow(t *testing.T) {
	id, dur, err := ParseVolumeProfileWindow("")
	if err != nil || id != VolumeProfileWindow24h || dur != 24*time.Hour {
		t.Fatalf("default %s %s %v", id, dur, err)
	}
	id, dur, err = ParseVolumeProfileWindow("7d")
	if err != nil || id != VolumeProfileWindow7d || dur != 7*24*time.Hour {
		t.Fatalf("7d %s %s %v", id, dur, err)
	}
	if _, _, err := ParseVolumeProfileWindow("2h"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("bad window %v", err)
	}
}

func TestResolveVolumeProfileRange_CustomAndCap(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	from, to, id, err := ResolveVolumeProfileRange("4h", nil, nil, now)
	if err != nil || id != VolumeProfileWindow4h || to != now || to.Sub(from) != 4*time.Hour {
		t.Fatalf("%s %s %s %v", id, from, to, err)
	}
	start := now.Add(-2 * time.Hour)
	end := now.Add(-time.Hour)
	from, to, id, err = ResolveVolumeProfileRange("24h", &start, &end, now)
	if err != nil || id != "custom" || !from.Equal(start) || !to.Equal(end) {
		t.Fatalf("custom %s %s %s %v", id, from, to, err)
	}
	tooLong := now.Add(-31 * 24 * time.Hour)
	if _, _, _, err := ResolveVolumeProfileRange("", &tooLong, &now, now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("cap %v", err)
	}
	after := now.Add(time.Hour)
	if _, _, _, err := ResolveVolumeProfileRange("", &after, &now, now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("order %v", err)
	}
}

func TestNiceTickSizeAndAuto(t *testing.T) {
	if got := NiceTickSize(62.5); got != 100 {
		t.Fatalf("nice %v", got)
	}
	if got := NiceTickSize(3.1); got != 5 {
		t.Fatalf("nice3 %v", got)
	}
	tick := AutoTickSize(64000, 68000)
	if tick != 100 {
		t.Fatalf("auto %v", tick)
	}
	wide := ClampVolumeProfileTick(100, 100+201*0.1, 0.1)
	if int((201*0.1)/wide) > MaxVolumeProfileBins {
		t.Fatalf("clamp still too fine %v", wide)
	}
}

func TestVolumeProfileBarsFromCandles_BuySell(t *testing.T) {
	bars := VolumeProfileBarsFromCandles([]Candle{
		{High: "102", Low: "100", Close: "101", QuoteVolume: "100", TakerBuyQuote: "70"},
		{High: "103", Low: "101", Close: "102", QuoteVolume: "50"},
	})
	if len(bars) != 2 {
		t.Fatalf("%+v", bars)
	}
	if !bars[0].BuySellKnown || bars[0].BuyVolume != 70 || bars[0].SellVolume != 30 {
		t.Fatalf("binance-like %+v", bars[0])
	}
	if bars[1].BuySellKnown {
		t.Fatalf("bybit-like should not invent buy/sell %+v", bars[1])
	}
}

func TestBuildVolumeProfile_POCValueAreaAndSides(t *testing.T) {
	// Most volume sits at 102. Expanding from POC should take 101 next (50 > 40).
	bars := []VolumeProfileBar{
		{Low: 100, High: 100.4, Close: 100.2, Volume: 10, BuyVolume: 4, SellVolume: 6, BuySellKnown: true},
		{Low: 101, High: 101.4, Close: 101.2, Volume: 50, BuyVolume: 20, SellVolume: 30, BuySellKnown: true},
		{Low: 102, High: 102.4, Close: 102.2, Volume: 100, BuyVolume: 80, SellVolume: 20, BuySellKnown: true},
		{Low: 103, High: 103.4, Close: 103.2, Volume: 40, BuyVolume: 10, SellVolume: 30, BuySellKnown: true},
		{Low: 104, High: 104.4, Close: 104.2, Volume: 10, BuyVolume: 5, SellVolume: 5, BuySellKnown: true},
	}
	got := BuildVolumeProfile(ExchangeBinance, "BTCUSDT", bars, 1, 102.2, time.Time{}, time.Time{}, Interval5m)
	if got.Error != "" {
		t.Fatal(got.Error)
	}
	if got.POC.Price != 102 || got.POC.Volume != 100 {
		t.Fatalf("poc %+v", got.POC)
	}
	if got.ValueArea.Low != 101 || got.ValueArea.High != 103 {
		t.Fatalf("va %+v bins=%+v", got.ValueArea, got.Bins)
	}
	if got.ValueArea.Volume < 150 || got.ValueArea.VolumePct < 70 {
		t.Fatalf("va volume %+v total=%v", got.ValueArea, got.TotalVolume)
	}
	if got.LastVsArea != VolumeProfileVsInside {
		t.Fatalf("last vs %s", got.LastVsArea)
	}
	if !got.BuySellKnown || got.POC.BuyVolume != 80 {
		t.Fatalf("buy/sell %+v", got.POC)
	}
	var pocBin VolumeProfileBin
	for _, b := range got.Bins {
		if b.IsPoc {
			pocBin = b
		}
	}
	if !pocBin.InValueArea || pocBin.BuyPct < 79 || pocBin.BuyPct > 81 {
		t.Fatalf("poc bin %+v", pocBin)
	}
	if got.Summary == "" || !containsAll(got.Summary, "65200") && !containsAll(got.Summary, "102") {
		// 102 is the POC
		if got.Summary == "" {
			t.Fatal("summary")
		}
	}
}

func TestBuildVolumeProfile_DistributesAcrossRange(t *testing.T) {
	bars := []VolumeProfileBar{{Low: 100, High: 102, Close: 101, Volume: 30, BuyVolume: 15, SellVolume: 15, BuySellKnown: true}}
	got := BuildVolumeProfile(ExchangeBinance, "ETHUSDT", bars, 1, 101, time.Time{}, time.Time{}, Interval1m)
	if len(got.Bins) != 3 {
		t.Fatalf("bins %+v", got.Bins)
	}
	for _, b := range got.Bins {
		if math.Abs(b.Volume-10) > 1e-9 {
			t.Fatalf("expected 10 per row %+v", b)
		}
	}
}

func TestCombineVolumeProfiles_AddsVenues(t *testing.T) {
	from := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	a := BuildVolumeProfile(ExchangeBinance, "BTCUSDT", []VolumeProfileBar{
		{Low: 100, High: 100.2, Close: 100.1, Volume: 10, BuyVolume: 8, SellVolume: 2, BuySellKnown: true},
	}, 1, 100.1, from, to, Interval1m)
	b := BuildVolumeProfile(ExchangeBybit, "BTCUSDT", []VolumeProfileBar{
		{Low: 100, High: 100.2, Close: 100.1, Volume: 20},
	}, 1, 100.1, from, to, Interval1m)
	got := CombineVolumeProfiles("BTCUSDT", []VolumeProfileVenue{a, b}, 1, 100.1, from, to, Interval1m)
	if got == nil || got.TotalVolume != 30 || got.POC.Volume != 30 {
		t.Fatalf("%+v", got)
	}
	if !got.BuySellKnown || !got.BuySellPartial {
		t.Fatalf("sides %+v", got)
	}
	if len(got.Bins) != 1 || len(got.Bins[0].Shares) != 2 {
		t.Fatalf("shares %+v", got.Bins)
	}
	var bn, by VolumeProfileShare
	for _, s := range got.Bins[0].Shares {
		switch s.Exchange {
		case ExchangeBinance:
			bn = s
		case ExchangeBybit:
			by = s
		}
	}
	if bn.Volume != 10 || by.Volume != 20 {
		t.Fatalf("share vols %+v", got.Bins[0].Shares)
	}
}

func TestBuildVolumeProfile_Empty(t *testing.T) {
	got := BuildVolumeProfile(ExchangeBinance, "BTCUSDT", nil, 1, 0, time.Time{}, time.Time{}, Interval1m)
	if got.Error == "" {
		t.Fatal("expected error")
	}
}

func TestProfileBarInterval(t *testing.T) {
	if ProfileBarInterval(time.Hour) != Interval1m {
		t.Fatal("1h")
	}
	if ProfileBarInterval(24*time.Hour) != Interval1m {
		t.Fatal("24h")
	}
	if ProfileBarInterval(7*24*time.Hour) != Interval5m {
		t.Fatal("7d")
	}
	if ProfileBarInterval(30*24*time.Hour) != Interval15m {
		t.Fatal("30d")
	}
}

func TestExplainVolumeProfileReport(t *testing.T) {
	rep := VolumeProfileReport{
		Combined: &VolumeProfileVenue{Summary: "clustered at 100", Error: ""},
	}
	if ExplainVolumeProfileReport(rep) != "Combined: clustered at 100" {
		t.Fatalf("%s", ExplainVolumeProfileReport(rep))
	}
}

func containsAll(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
