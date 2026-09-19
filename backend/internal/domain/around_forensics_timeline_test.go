package domain

import (
	"strings"
	"testing"
	"time"
)

func TestFindAroundLiqStart_FirstPrint(t *testing.T) {
	from := time.Date(2026, 8, 20, 13, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)
	got, side := FindAroundLiqStart([]LiquidationEvent{
		{Side: LiquidationSideLong, Notional: 1e6, Time: from.Add(50 * time.Minute)},
		{Side: LiquidationSideShort, Notional: 9e6, Time: from.Add(70 * time.Minute)},
		{Side: LiquidationSideShort, Notional: 2e6, Time: from.Add(-time.Minute)},
	}, from, to)
	if side != LiquidationSideLong {
		t.Fatalf("side %q", side)
	}
	if !got.Equal(from.Add(50 * time.Minute)) {
		t.Fatalf("got %s", got)
	}
}

func TestFindAroundLiqStart_SameTimePrefersLarger(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 5, 0, 0, time.UTC)
	got, side := FindAroundLiqStart([]LiquidationEvent{
		{Side: LiquidationSideLong, Notional: 1e5, Time: at},
		{Side: LiquidationSideShort, Notional: 8e6, Time: at},
	}, at.Add(-time.Hour), at.Add(time.Hour))
	if side != LiquidationSideShort || got.IsZero() {
		t.Fatalf("got %s side %q", got, side)
	}
}

func TestBuildForensicsTimeline_Order(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange:  ExchangeBinance,
		Symbol:    "BTCUSDT",
		OIStart:   at.Add(-8 * time.Minute),
		OIDir:     CVDDirUp,
		OIUntil:   AroundVenueUntilContinued,
		MoveStart: at,
		LiqStart:  at.Add(3 * time.Minute),
		LiqSide:   LiquidationSideShort,
		Phases:    []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 1.2}}},
	}
	by := AroundVenue{
		Exchange:  ExchangeBybit,
		Symbol:    "BTCUSDT",
		MoveStart: at.Add(2 * time.Minute),
		Phases:    []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 0.9}}},
	}
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m, AsOf: at.Add(2 * time.Hour),
		Tape: &AroundReport{
			Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m,
			Venues: []AroundVenue{bn, by},
		},
	}
	got := BuildForensicsTimeline(r)
	if len(got.Events) != 4 {
		t.Fatalf("events=%d %+v", len(got.Events), got.Events)
	}
	wantKind := []string{ForensicsTimelineOI, ForensicsTimelinePrice, ForensicsTimelinePrice, ForensicsTimelineLiq}
	wantVenue := []string{"binance", "binance", "bybit", "binance"}
	for i, ev := range got.Events {
		if ev.Kind != wantKind[i] || ev.Venue != wantVenue[i] {
			t.Fatalf("i=%d kind=%s venue=%s title=%q", i, ev.Kind, ev.Venue, ev.Title)
		}
	}
	if !strings.Contains(got.Events[0].Title, "open interest started rising") {
		t.Fatalf("oi title %q", got.Events[0].Title)
	}
	if !strings.Contains(got.Events[0].OffsetLabel, "before") {
		t.Fatalf("oi offset %q", got.Events[0].OffsetLabel)
	}
	if got.Events[1].OffsetLabel != "at the event" {
		t.Fatalf("price offset %q", got.Events[1].OffsetLabel)
	}
	if !strings.Contains(got.Events[2].Title, "Bybit") || !strings.Contains(got.Events[3].Title, "short liquidations") {
		t.Fatalf("follow/liq %+v", got.Events[2:])
	}
	if !strings.Contains(got.Summary, "open interest") || !strings.Contains(got.Summary, "Bybit") || !strings.Contains(got.Summary, "short liquidations") {
		t.Fatalf("summary %q", got.Summary)
	}
	if !got.KnownUntil.Equal(at.Add(2 * time.Hour)) {
		t.Fatalf("knownUntil %s", got.KnownUntil)
	}
}

func TestBuildForensicsTimeline_GammaBeforeMove(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m, AsOf: at.Add(time.Hour),
		Tape: &AroundReport{
			Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m,
			Combined: &AroundVenue{
				Exchange: "all",
				Phases: []AroundPhase{{
					Phase: AroundPhaseBefore,
					Options: &AroundOptions{
						HasGamma: true, GammaRegime: GammaRegimeShort, GammaNetUSD: -2e8,
						HasGammaFlip: true, GammaFlip: 97200, GammaFlipSide: GammaFlipAbove,
						GammaAt: at.Add(-10 * time.Minute),
					},
				}},
			},
			Venues: []AroundVenue{{
				Exchange: ExchangeBinance, Symbol: "BTCUSDT", MoveStart: at,
				Phases: []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp}}},
			}},
		},
	}
	got := BuildForensicsTimeline(r)
	var gamma ForensicsTimelineEvent
	for _, ev := range got.Events {
		if ev.Kind == ForensicsTimelineGamma {
			gamma = ev
		}
	}
	if gamma.Title != "Dealers were short gamma" {
		t.Fatalf("gamma %+v events=%+v", gamma, got.Events)
	}
	if !strings.Contains(gamma.Detail, "above the flip") || gamma.Phase != AroundPhaseBefore {
		t.Fatalf("detail %+v", gamma)
	}
}

func TestBuildForensicsTimeline_EmptyTape(t *testing.T) {
	if got := BuildForensicsTimeline(ForensicsReport{}); len(got.Events) != 0 || got.Summary != "" {
		t.Fatalf("%+v", got)
	}
}

func TestForensicsTimelineKnownUntil_UsesAsOf(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	asOf := at.Add(10 * time.Minute)
	if got := ForensicsTimelineKnownUntil(asOf); !got.Equal(asOf) {
		t.Fatalf("got %s", got)
	}
	later := at.Add(26 * time.Hour)
	if got := ForensicsTimelineKnownUntil(later); !got.Equal(later) {
		t.Fatalf("old event %s", got)
	}
}

func TestBuildForensicsTimeline_OldEventKeepsLaterPrints(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	bn := AroundVenue{
		Exchange:  ExchangeBinance,
		Symbol:    "BTCUSDT",
		MoveStart: at,
		LiqStart:  at.Add(30 * time.Minute),
		LiqSide:   LiquidationSideShort,
		Phases:    []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 1.2}}},
	}
	by := AroundVenue{
		Exchange:  ExchangeBybit,
		Symbol:    "BTCUSDT",
		MoveStart: at.Add(30 * time.Minute),
		Phases:    []AroundPhase{{Phase: AroundPhaseAfter, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 0.9}}},
	}
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m, AsOf: at.Add(26 * time.Hour),
		Tape: &AroundReport{
			Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m,
			Venues: []AroundVenue{bn, by},
		},
	}
	got := BuildForensicsTimeline(r)
	if !got.KnownUntil.Equal(at.Add(26 * time.Hour)) {
		t.Fatalf("knownUntil %s", got.KnownUntil)
	}
	if len(got.Events) != 3 {
		t.Fatalf("events=%d %+v", len(got.Events), got.Events)
	}
}

func TestBuildForensicsTimeline_AddsLaterPrintsWhenStillOpen(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	asOf := at.Add(40 * time.Minute)
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m, AsOf: asOf,
		Tape: &AroundReport{
			Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m,
			Venues: []AroundVenue{{
				Exchange: ExchangeBinance, Symbol: "BTCUSDT",
				MoveStart: at, LiqStart: at.Add(30 * time.Minute), LiqSide: LiquidationSideShort,
				Phases: []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 1}}},
			}},
		},
	}
	got := BuildForensicsTimeline(r)
	if len(got.Events) != 2 {
		t.Fatalf("events=%d %+v", len(got.Events), got.Events)
	}
	if got.Events[1].Kind != ForensicsTimelineLiq {
		t.Fatalf("%+v", got.Events[1])
	}
}

func TestBuildForensicsTimeline_AsOfTenMinutesHidesLaterVenue(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m,
		AsOf: at.Add(10 * time.Minute), AsOfExplicit: true,
		Tape: &AroundReport{
			Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m,
			Venues: []AroundVenue{
				{Exchange: ExchangeBinance, Symbol: "BTCUSDT", MoveStart: at, LiqStart: at.Add(3 * time.Minute), LiqSide: LiquidationSideShort},
				{Exchange: ExchangeBybit, Symbol: "BTCUSDT", MoveStart: at.Add(30 * time.Minute)},
			},
		},
	}
	got := BuildForensicsTimeline(r)
	if len(got.Events) != 2 {
		t.Fatalf("events=%d %+v", len(got.Events), got.Events)
	}
	if got.Events[0].Venue != "binance" || got.Events[1].Kind != ForensicsTimelineLiq {
		t.Fatalf("%+v", got.Events)
	}
}

func TestBuildForensicsTimeline_OISampleOverlapsEarlierPriceBar(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m, AsOf: at.Add(2 * time.Hour),
		Tape: &AroundReport{
			Venues: []AroundVenue{{
				Exchange: ExchangeBinance, Interval: string(Interval1m),
				MoveStart: at.Add(3 * time.Minute),
				OIStart:   at.Add(5 * time.Minute), OIDir: CVDDirUp,
				Phases: []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 1}}},
			}},
		},
	}
	got := BuildForensicsTimeline(r)
	if got.OrderKnown {
		t.Fatalf("OI 14:05 vs 1m price 14:03 must be unknown %+v", got)
	}
	if !strings.Contains(got.Summary, "Order is not certain") {
		t.Fatalf("summary %q", got.Summary)
	}
}

func TestBuildForensicsTimeline_OverlappingStampsDoNotClaimOrder(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	r := ForensicsReport{
		Symbol: "BTCUSDT", At: at, Window: AroundWindow1h, During: AroundDuring15m, AsOf: at.Add(2 * time.Hour),
		Tape: &AroundReport{
			Venues: []AroundVenue{
				{Exchange: ExchangeBinance, Interval: string(Interval5m), MoveStart: at,
					Phases: []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 1}}}},
				{Exchange: ExchangeBybit, Interval: string(Interval5m), MoveStart: at.Add(2 * time.Minute),
					Phases: []AroundPhase{{Phase: AroundPhaseDuring, Complete: true, Price: AroundPrice{Direction: CVDDirUp, ChangePct: 0.9}}}},
			},
		},
	}
	got := BuildForensicsTimeline(r)
	if got.OrderKnown || len(got.Events) != 2 {
		t.Fatalf("%+v", got)
	}
	if !got.Events[0].OrderUnknown || !got.Events[1].OrderUnknown {
		t.Fatalf("events %+v", got.Events)
	}
	if !strings.Contains(got.Summary, "Order is not certain") {
		t.Fatalf("summary %q", got.Summary)
	}
	if got.Events[1].OffsetLabel != "around the event" {
		t.Fatalf("offset %q", got.Events[1].OffsetLabel)
	}
}

func TestFormatForensicsTimelineOffset(t *testing.T) {
	if got := FormatForensicsTimelineOffset(-8 * time.Minute); got != "-8m" {
		t.Fatalf("%q", got)
	}
	if got := FormatForensicsTimelineOffset(3 * time.Minute); got != "+3m" {
		t.Fatalf("%q", got)
	}
	if got := FormatForensicsTimelineOffset(10 * time.Second); got != "0" {
		t.Fatalf("%q", got)
	}
}
