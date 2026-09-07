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
	if got.Leader != string(ExchangeBinance) {
		t.Fatalf("leader %q", got.Leader)
	}
}

func TestCompareAroundVenueTiming_FirstStartIsLeaderNotLargerMove(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange: ExchangeBinance, MoveStart: at,
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true,
			Price: AroundPrice{ChangePct: 0.45, Direction: CVDDirUp},
		}},
	}
	by := AroundVenue{
		Exchange: ExchangeBybit, MoveStart: at.Add(10 * time.Minute),
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true,
			Price: AroundPrice{ChangePct: 2.8, Direction: CVDDirUp},
		}},
	}
	got := CompareAroundVenueTiming([]AroundVenue{bn, by})
	if got == nil || got.Agreement != AroundVenueBinanceEarlier || got.Follow != AroundVenueFollowed {
		t.Fatalf("%+v", got)
	}
	if got.Leader != string(ExchangeBinance) {
		t.Fatalf("leader should be first start, got %q", got.Leader)
	}
}

func TestCompareAroundVenueTiming_OIAndTakersBeforeLaggard(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange: ExchangeBinance, MoveStart: at,
		Phases: []AroundPhase{
			{
				Phase: AroundPhaseDuring, Complete: true,
				Price:   AroundPrice{ChangePct: 1.6, Direction: CVDDirUp},
				Flow:    AroundFlow{BuySellKnown: true, Dominant: TakerSideBuy, BuyShare: 0.62},
				Futures: &AroundFutures{Complete: true, OIDirection: CVDDirUp, OIChangePct: 1.8},
			},
		},
	}
	by := AroundVenue{
		Exchange: ExchangeBybit,
		Phases: []AroundPhase{
			{
				Phase: AroundPhaseDuring, Complete: true,
				Price:   AroundPrice{ChangePct: 0.04, Direction: CVDDirFlat},
				Flow:    AroundFlow{BuySellKnown: true, Dominant: TakerSideBuy, BuyShare: 0.58},
				Futures: &AroundFutures{Complete: true, OIDirection: CVDDirUp, OIChangePct: 1.2},
			},
		},
	}
	got := CompareAroundVenueTiming([]AroundVenue{bn, by})
	if got == nil || got.Agreement != AroundVenueBinanceOnly {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "open interest") || !strings.Contains(got.Summary, "takers") {
		t.Fatalf("summary should mention OI and takers: %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "Bybit open interest") {
		t.Fatalf("Bybit OI should show even when Bybit price did not follow: %q", got.Summary)
	}
	if len(got.Before) < 3 {
		t.Fatalf("before bits %+v", got.Before)
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

func TestCompareAroundVenueTiming_DoesNotUseWindowEndAsBeforeCutoff(t *testing.T) {
	at := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	byStart := at.Add(5 * time.Minute)
	bn := AroundVenue{
		Exchange: ExchangeBinance, MoveStart: at,
		Phases: []AroundPhase{
			{
				Phase: AroundPhaseBefore, Complete: true,
				From: at.Add(-time.Hour), To: at,
				Futures: &AroundFutures{Complete: true, OIDirection: CVDDirUp, OIChangePct: 1.1},
			},
			{
				Phase: AroundPhaseDuring, Complete: true,
				From: at, To: at.Add(15 * time.Minute),
				Price:   AroundPrice{ChangePct: 1.6, Direction: CVDDirUp},
				Flow:    AroundFlow{BuySellKnown: true, Dominant: TakerSideBuy, BuyShare: 0.64},
				Futures: &AroundFutures{Complete: true, OIDirection: CVDDirUp, OIChangePct: 2.4},
			},
		},
	}
	by := AroundVenue{
		Exchange: ExchangeBybit, MoveStart: byStart,
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true,
			From: at, To: at.Add(15 * time.Minute),
			Price:   AroundPrice{ChangePct: 1.2, Direction: CVDDirUp},
			Flow:    AroundFlow{BuySellKnown: true, Dominant: TakerSideBuy, BuyShare: 0.6},
			Futures: &AroundFutures{Complete: true, OIDirection: CVDDirUp, OIChangePct: 1.5},
		}},
	}
	got := CompareAroundVenueTiming([]AroundVenue{bn, by})
	if got == nil || got.Agreement != AroundVenueBinanceEarlier {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "12:05") {
		t.Fatalf("cutoff should be Bybit start 12:05, not 12:15: %q", got.Summary)
	}
	if strings.Contains(got.Summary, "takers") {
		t.Fatalf("during takers run to 12:15 and must not count as before 12:05: %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "open interest") {
		t.Fatalf("before-window OI ends at 12:00 and should count: %q", got.Summary)
	}
}

func TestCompareAroundVenueTiming_EarlierBybitOIIsNotPriceOnlyLeader(t *testing.T) {
	at := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange: ExchangeBinance, MoveStart: at,
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true, From: at, To: at.Add(15 * time.Minute),
			Price: AroundPrice{ChangePct: 1.4, Direction: CVDDirUp},
		}},
	}
	by := AroundVenue{
		Exchange: ExchangeBybit, MoveStart: at.Add(8 * time.Minute),
		OIStart: at.Add(-6 * time.Minute), OIDir: CVDDirUp,
		Phases: []AroundPhase{{
			Phase: AroundPhaseDuring, Complete: true, From: at, To: at.Add(15 * time.Minute),
			Price: AroundPrice{ChangePct: 1.1, Direction: CVDDirUp},
		}},
	}
	got := CompareAroundVenueTiming([]AroundVenue{bn, by})
	if got == nil || got.Agreement != AroundVenueBinanceEarlier {
		t.Fatalf("%+v", got)
	}
	if got.Leader != string(ExchangeBybit) || got.LeaderMetric != AroundVenueMetricOI {
		t.Fatalf("first signal should be Bybit OI, got leader=%q metric=%q", got.Leader, got.LeaderMetric)
	}
	if !strings.Contains(got.Summary, "Bybit open interest") || !strings.Contains(got.Summary, "before Binance price") {
		t.Fatalf("summary %q", got.Summary)
	}
}

func TestFindAroundTakerRunStart_FindsLean(t *testing.T) {
	at := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	plan := aroundTestPlan(at)
	var bars []AroundBar
	for i := -20; i < 15; i++ {
		t0 := at.Add(time.Duration(i) * time.Minute)
		b := AroundBar{Time: t0, Open: 100, Close: 100, High: 100, Low: 100, Volume: 10, BuySellKnown: true}
		if i >= 4 {
			b.BuyVolume, b.SellVolume = 8, 2
		} else {
			b.BuyVolume, b.SellVolume = 5, 5
		}
		bars = append(bars, b)
	}
	got, dir := FindAroundTakerRunStart(bars, plan)
	if dir != TakerSideBuy {
		t.Fatalf("dir %q", dir)
	}
	if got.Before(at.Add(3*time.Minute)) || got.After(at.Add(6*time.Minute)) {
		t.Fatalf("taker start %s", got)
	}
}

func TestFindAroundOIRunStart_FindsFirstSample(t *testing.T) {
	at := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	plan := aroundTestPlan(at)
	var snaps []FuturesSnapshot
	for i := -20; i <= 15; i++ {
		oi := 1000.0
		if i >= 3 {
			oi = 1000 + float64(i-2)*4
		}
		snaps = append(snaps, FuturesSnapshot{
			Metric: FuturesMetricOpenInterest, SampledAt: at.Add(time.Duration(i) * time.Minute), Value: oi,
		})
	}
	got, dir := FindAroundOIRunStart(snaps, plan)
	if dir != CVDDirUp {
		t.Fatalf("dir %q", dir)
	}
	if got.Before(at.Add(2*time.Minute)) || got.After(at.Add(8*time.Minute)) {
		t.Fatalf("oi start %s", got)
	}
}
