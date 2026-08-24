package domain

import (
	"testing"
	"time"
)

func TestDiffAroundPhases_LaterMoveIsLarger(t *testing.T) {
	from := AroundPhase{
		Phase: AroundPhaseDuring, Complete: true,
		Price: AroundPrice{Open: 100, Close: 101, Change: 1, ChangePct: 1, RangePct: 1.2, Direction: CVDDirUp},
		Flow:  AroundFlow{Volume: 2_000, VolumeRatio: 1.1, TypicalKnown: true, BuySellKnown: true, Delta: 200, Dominant: TakerSideBuy},
	}
	to := AroundPhase{
		Phase: AroundPhaseDuring, Complete: true,
		Price: AroundPrice{Open: 110, Close: 118, Change: 8, ChangePct: 7.27, RangePct: 8, Direction: CVDDirUp},
		Flow:  AroundFlow{Volume: 9_000, VolumeRatio: 4.5, TypicalKnown: true, BuySellKnown: true, Delta: 5_000, Dominant: TakerSideBuy},
	}
	got := DiffAroundPhases(AroundPhaseDuring, from, to)
	move, ok := findCompareDelta(got.Deltas, AroundCompareMetricMove)
	if !ok || move.Direction != CVDDirUp || move.To < 7 {
		t.Fatalf("move %+v", move)
	}
	vol, ok := findCompareDelta(got.Deltas, AroundCompareMetricVolume)
	if !ok || vol.To != 9_000 || vol.Direction != CVDDirUp {
		t.Fatalf("vol %+v", vol)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestCompareAroundVenues_StatePriceAndBook(t *testing.T) {
	mk := func(open, vol float64, mid float64, oi float64) AroundVenue {
		return AroundVenue{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT",
			Phases: []AroundPhase{{
				Phase: AroundPhaseDuring, Complete: true,
				Price:   AroundPrice{Open: open, Close: open * 1.01, ChangePct: 1, Direction: CVDDirUp},
				Flow:    AroundFlow{Volume: vol},
				Book:    &AroundBook{FromMid: mid, ToMid: mid * 1.01, MidDelta: mid * 0.01, Complete: true},
				Futures: &AroundFutures{OIFrom: oi, OITo: oi * 1.05, OIChangePct: 5, Complete: true},
			}},
		}
	}
	got := CompareAroundVenues(mk(100, 1_000, 100, 1_000), mk(120, 3_000, 120, 1_400))
	px, ok := findCompareDelta(got.State, AroundCompareMetricPrice)
	if !ok || px.From != 100 || px.To != 120 || px.Direction != CVDDirUp {
		t.Fatalf("price state %+v", px)
	}
	mid, ok := findCompareDelta(got.State, AroundCompareMetricBookMid)
	if !ok || mid.From != 100 || mid.To != 120 {
		t.Fatalf("book %+v", mid)
	}
	oi, ok := findCompareDelta(got.State, AroundCompareMetricOI)
	if !ok || oi.From != 1_000 || oi.To != 1_400 {
		t.Fatalf("oi %+v", oi)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestCompareAroundReports_PrefersCombined(t *testing.T) {
	at1 := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	at2 := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)
	mk := func(at time.Time, change, vol float64) AroundReport {
		ph := AroundPhase{
			Phase: AroundPhaseDuring, Complete: true,
			Price: AroundPrice{Open: 100, Close: 100 + change, Change: change, ChangePct: change, Direction: CVDDirUp},
			Flow:  AroundFlow{Volume: vol, VolumeRatio: vol / 1000, TypicalKnown: true},
		}
		ven := AroundVenue{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Phases: []AroundPhase{ph}}
		comb := ven
		comb.Exchange = "all"
		return AroundReport{
			Symbol: "BTCUSDT", Exchange: "all", At: at, Window: "1h", During: "15m",
			Venues: []AroundVenue{ven}, Combined: &comb,
		}
	}
	got := CompareAroundReports(mk(at1, 1, 2_000), mk(at2, 6, 8_000))
	if got.FromAt != at1 || got.ToAt != at2 || got.Combined == nil {
		t.Fatalf("%+v", got)
	}
	if got.FromMove == nil || got.ToMove == nil {
		t.Fatal("embedded moves")
	}
	if got.Summary == "" || got.Combined.Summary == "" {
		t.Fatalf("summary %q %q", got.Summary, got.Combined.Summary)
	}
}
