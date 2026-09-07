package domain

import (
	"strings"
	"testing"
	"time"
)

func aroundTestPlan(at time.Time) AroundPlan {
	return AroundPlan{
		At: at, DuringFrom: at, DuringTo: at.Add(15 * time.Minute),
		BeforeFrom: at.Add(-time.Hour), AfterFrom: at.Add(15 * time.Minute),
		AfterTo: at.Add(75 * time.Minute), WindowDur: time.Hour,
	}
}

func aroundFlatBars(at time.Time, fromMin, toMin int, px float64) []AroundBar {
	var bars []AroundBar
	for i := fromMin; i < toMin; i++ {
		t0 := at.Add(time.Duration(i) * time.Minute)
		bars = append(bars, AroundBar{Time: t0, Open: px, Close: px, High: px, Low: px})
	}
	return bars
}

func TestFindAroundPriceRunStart_FindsFirstPrint(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	plan := aroundTestPlan(at)
	var bars []AroundBar
	px := 100.0
	for i := -20; i < 15; i++ {
		t0 := at.Add(time.Duration(i) * time.Minute)
		if i >= 3 {
			px = 101.5 + float64(i-3)*0.2
		}
		bars = append(bars, AroundBar{Time: t0, Open: px, Close: px, High: px, Low: px})
	}
	got := FindAroundPriceRunStart(bars, plan)
	if got.Before(at.Add(2*time.Minute)) || got.After(at.Add(5*time.Minute)) {
		t.Fatalf("start %s", got)
	}
}

func TestFindAroundPriceRunStart_QuietVenueHasNoStart(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	plan := aroundTestPlan(at)
	bars := aroundFlatBars(at, -20, 80, 100)
	// A 0.08% wiggle before `at` must not look like Bybit started the move.
	for i := range bars {
		if bars[i].Time.Equal(at.Add(-8 * time.Minute)) {
			bars[i].Close = 100.08
			bars[i].High = 100.08
		}
	}
	got := FindAroundPriceRunStart(bars, plan)
	if !got.IsZero() {
		t.Fatalf("quiet venue start %s", got)
	}
}

func TestFindAroundPriceRunStart_FollowsInAfter(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	plan := aroundTestPlan(at)
	var bars []AroundBar
	px := 100.0
	for i := -20; i < 80; i++ {
		t0 := at.Add(time.Duration(i) * time.Minute)
		if i >= 20 {
			px = 101.2 + float64(i-20)*0.05
		}
		bars = append(bars, AroundBar{Time: t0, Open: px, Close: px, High: px, Low: px})
	}
	got := FindAroundPriceRunStart(bars, plan)
	if got.Before(at.Add(18*time.Minute)) || got.After(at.Add(25*time.Minute)) {
		t.Fatalf("after follow start %s", got)
	}
}

func TestCompareAroundVenueTiming_BinanceEarlier(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	got := CompareAroundVenueTiming([]AroundVenue{
		{Exchange: ExchangeBinance, MoveStart: at.Add(-3 * time.Minute)},
		{Exchange: ExchangeBybit, MoveStart: at.Add(5 * time.Minute)},
	})
	if got == nil || got.Agreement != AroundVenueBinanceEarlier || got.Follow != AroundVenueFollowed {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "Binance started") || !strings.Contains(got.Summary, "Bybit followed") {
		t.Fatalf("summary %q", got.Summary)
	}
	same := CompareAroundVenueTiming([]AroundVenue{
		{Exchange: ExchangeBinance, MoveStart: at},
		{Exchange: ExchangeBybit, MoveStart: at.Add(time.Minute)},
	})
	if same == nil || same.Agreement != AroundVenueSame {
		t.Fatalf("same %+v", same)
	}
}

func TestCompareAroundVenueTiming_BinanceOnlyWhenBybitDidNotMove(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange: ExchangeBinance, MoveStart: at,
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true,
			Price: AroundPrice{ChangePct: 1.8, Direction: CVDDirUp},
		}},
	}
	by := AroundVenue{
		Exchange: ExchangeBybit, MoveStart: at.Add(-4 * time.Minute),
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true,
			Price: AroundPrice{ChangePct: 0.04, Direction: CVDDirFlat},
		}},
	}
	got := CompareAroundVenueTiming([]AroundVenue{bn, by})
	if got == nil || got.Agreement != AroundVenueBinanceOnly || got.Follow != AroundVenueFollowStayed {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "Bybit did not follow") {
		t.Fatalf("summary %q", got.Summary)
	}
}

func TestCompareAroundVenueTiming_BybitFollowsLater(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange: ExchangeBinance, MoveStart: at,
		Phases: []AroundPhase{
			{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{ChangePct: 1.6, Direction: CVDDirUp}},
			{Phase: AroundPhaseAfter, Complete: true, Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat}},
		},
	}
	by := AroundVenue{
		Exchange: ExchangeBybit, MoveStart: at.Add(20 * time.Minute),
		Phases: []AroundPhase{
			{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{ChangePct: 0.03, Direction: CVDDirFlat}},
			{Phase: AroundPhaseAfter, Complete: true, Price: AroundPrice{ChangePct: 1.4, Direction: CVDDirUp}},
		},
	}
	got := CompareAroundVenueTiming([]AroundVenue{bn, by})
	if got == nil || got.Agreement != AroundVenueBinanceEarlier || got.Follow != AroundVenueFollowed {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "Bybit followed later") {
		t.Fatalf("summary %q", got.Summary)
	}
}
