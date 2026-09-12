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
}

func TestBuildForensicsTimeline_EmptyTape(t *testing.T) {
	if got := BuildForensicsTimeline(ForensicsReport{}); len(got.Events) != 0 || got.Summary != "" {
		t.Fatalf("%+v", got)
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
