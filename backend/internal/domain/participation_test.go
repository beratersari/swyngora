package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestBuildParticipationWindow_Example(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	oi := &OpenInterestSeries{
		Current: OpenInterestPoint{Time: now, Value: 10_300_000_000},
		History: []OpenInterestPoint{
			{Time: now.Add(-15 * time.Minute), Value: 10_000_000_000},
			{Time: now.Add(-time.Hour), Value: 9_800_000_000},
			{Time: now.Add(-4 * time.Hour), Value: 9_000_000_000},
		},
	}
	// Three 5m bars × (400M buy + 266.666...M sell) ≈ 2B over 15m.
	var buckets []TakerBucket
	for i := 0; i < 3; i++ {
		buckets = append(buckets, TakerBucket{
			Start:        now.Add(-time.Duration(i+1) * 5 * time.Minute),
			BuyNotional:  400_000_000,
			SellNotional: 266_666_666.6666667,
		})
	}
	got := BuildParticipationWindow(oi, buckets, ParticipationWindow15m, 15*time.Minute, now)
	if !got.Complete {
		t.Fatalf("incomplete %+v", got)
	}
	if math.Abs(got.OIDelta-300_000_000) > 1 {
		t.Fatalf("oiDelta=%v", got.OIDelta)
	}
	if math.Abs(got.Volume-2_000_000_000) > 2 {
		t.Fatalf("volume=%v", got.Volume)
	}
	if math.Abs(got.Ratio-0.15) > 0.001 {
		t.Fatalf("ratio=%v", got.Ratio)
	}
	if math.Abs(got.RatioPct-15) > 0.1 {
		t.Fatalf("ratioPct=%v", got.RatioPct)
	}
	if got.Lean != ParticipationEntering {
		t.Fatalf("lean=%s", got.Lean)
	}
	if got.Summary == "" {
		t.Fatal("empty summary")
	}
}

func TestClassifyParticipation(t *testing.T) {
	t.Parallel()
	if ClassifyParticipation(15) != ParticipationEntering {
		t.Fatal("entering")
	}
	if ClassifyParticipation(-12) != ParticipationExiting {
		t.Fatal("exiting")
	}
	if ClassifyParticipation(1.2) != ParticipationTransfer {
		t.Fatal("transfer")
	}
}

func TestBuildParticipationVenue_Windows(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 1, 16, 0, 0, 0, time.UTC)
	oi := &OpenInterestSeries{
		Current: OpenInterestPoint{Time: now, Value: 1100},
		History: []OpenInterestPoint{
			{Time: now.Add(-15 * time.Minute), Value: 1000},
			{Time: now.Add(-time.Hour), Value: 900},
			{Time: now.Add(-4 * time.Hour), Value: 800},
		},
	}
	var buckets []TakerBucket
	for i := 0; i < 48; i++ {
		buckets = append(buckets, TakerBucket{
			Start:        now.Add(-time.Duration(i+1) * 5 * time.Minute),
			BuyNotional:  50,
			SellNotional: 50,
		})
	}
	got := BuildParticipationVenue(ExchangeBinance, "btcusdt", oi, buckets, now)
	if got.Symbol != "BTCUSDT" || len(got.Windows) != 3 {
		t.Fatalf("%+v", got)
	}
	var h15, h1, h4 *ParticipationWindow
	for i := range got.Windows {
		switch got.Windows[i].Window {
		case ParticipationWindow15m:
			h15 = &got.Windows[i]
		case ParticipationWindow1h:
			h1 = &got.Windows[i]
		case ParticipationWindow4h:
			h4 = &got.Windows[i]
		}
	}
	if h15 == nil || !h15.Complete || h15.OIDelta != 100 {
		t.Fatalf("15m %+v", h15)
	}
	if h1 == nil || !h1.Complete || h1.OIDelta != 200 {
		t.Fatalf("1h %+v", h1)
	}
	if h4 == nil || !h4.Complete || h4.OIDelta != 300 {
		t.Fatalf("4h %+v", h4)
	}
}

func TestCombineParticipationVenues(t *testing.T) {
	t.Parallel()
	a := ParticipationVenue{
		Exchange: ExchangeBinance,
		Windows: []ParticipationWindow{
			{Window: ParticipationWindow15m, Volume: 1_500_000_000, OIDelta: 200_000_000, Complete: true},
			{Window: ParticipationWindow1h, Volume: 4_000_000_000, OIDelta: 100_000_000, Complete: true},
			{Window: ParticipationWindow4h, Volume: 10_000_000_000, OIDelta: 50_000_000, Complete: true},
		},
	}
	b := ParticipationVenue{
		Exchange: ExchangeBybit,
		Windows: []ParticipationWindow{
			{Window: ParticipationWindow15m, Volume: 500_000_000, OIDelta: 100_000_000, Complete: true},
			{Window: ParticipationWindow1h, Volume: 1_000_000_000, OIDelta: 50_000_000, Complete: true},
			{Window: ParticipationWindow4h, Volume: 2_000_000_000, OIDelta: 25_000_000, Complete: true},
		},
	}
	got := CombineParticipationVenues("BTCUSDT", []ParticipationVenue{a, b})
	if got == nil || got.Exchange != "all" {
		t.Fatalf("%+v", got)
	}
	w := got.Windows[0]
	if w.Window != ParticipationWindow15m || w.Volume != 2_000_000_000 || w.OIDelta != 300_000_000 {
		t.Fatalf("15m %+v", w)
	}
	if math.Abs(w.Ratio-0.15) > 1e-9 || w.Lean != ParticipationEntering {
		t.Fatalf("ratio %+v", w)
	}
}

func TestCombineParticipationVenues_IncompleteWindowIsBlank(t *testing.T) {
	t.Parallel()
	a := ParticipationVenue{
		Exchange: ExchangeBinance,
		Windows: []ParticipationWindow{
			{Window: ParticipationWindow15m, Volume: 1_500_000_000, OIDelta: 200_000_000, Complete: true},
		},
	}
	b := ParticipationVenue{
		Exchange: ExchangeBybit,
		Error:    "tape down",
		Windows: []ParticipationWindow{
			{Window: ParticipationWindow15m, Volume: 500_000_000, OIDelta: 100_000_000, Complete: true},
		},
	}
	got := CombineParticipationVenues("BTCUSDT", []ParticipationVenue{a, b})
	if got == nil || len(got.Windows) == 0 {
		t.Fatal("nil")
	}
	w := got.Windows[0]
	if w.Complete || w.Volume != 0 || w.Ratio != 0 || w.Lean != "" {
		t.Fatalf("%+v", w)
	}
	if len(w.Missing) != 1 || w.Missing[0] != "bybit" || !strings.Contains(w.Summary, "bybit") {
		t.Fatalf("summary %+v", w)
	}
	if !strings.Contains(got.Summary, "bybit") {
		t.Fatalf("venue summary %s", got.Summary)
	}
}

func TestTakerVolumeInWindow_NeedsHistory(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	buckets := []TakerBucket{{Start: now.Add(-10 * time.Minute), BuyNotional: 1, SellNotional: 1}}
	vol, ok := TakerVolumeInWindow(buckets, now, time.Hour, 5*time.Minute)
	if vol != 2 || ok {
		t.Fatalf("vol=%v complete=%v", vol, ok)
	}
}
