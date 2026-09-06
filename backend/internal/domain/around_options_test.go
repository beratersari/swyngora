package domain

import (
	"strings"
	"testing"
	"time"
)

func TestOptionsAcrossWindow_DeltasAndUnusual(t *testing.T) {
	base := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	points := []OptionsHistoryPoint{
		{Time: base, OpenInterest: 100, HasOI: true, PutCallOI: 0.9, HasPutCallOI: true, DVOL: 40, HasDVOL: true},
		{Time: base.Add(time.Hour), OpenInterest: 101, HasOI: true, PutCallOI: 0.91, HasPutCallOI: true, DVOL: 40.2, HasDVOL: true},
		{Time: base.Add(2 * time.Hour), OpenInterest: 130, HasOI: true, PutCallOI: 1.3, HasPutCallOI: true, DVOL: 46, HasDVOL: true},
	}
	from := base.Add(time.Hour)
	to := base.Add(2 * time.Hour)
	got := OptionsAcrossWindow(OptionsAssetBTC, points, from, to)
	if got == nil || !got.Complete || !got.HasOI || !got.Unusual {
		t.Fatalf("%+v", got)
	}
	if got.OIDirection != CVDDirUp || got.OIChangePct < 20 {
		t.Fatalf("oi %+v", got)
	}
	if got.LeanTo != OptionsLeanPuts {
		t.Fatalf("lean %q", got.LeanTo)
	}
	evs := AroundOptionsToEvents(*got, AroundPhaseBefore, to)
	if len(evs) == 0 || evs[0].Kind != AroundEventOptions {
		t.Fatalf("events %+v", evs)
	}
}

func TestOptionsAcrossWindow_Missing(t *testing.T) {
	if got := OptionsAcrossWindow(OptionsAssetBTC, nil, time.Now().Add(-time.Hour), time.Now()); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestOptionsAcrossWindow_OneSidedDoesNotInventChange(t *testing.T) {
	to := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	from := to.Add(-time.Hour)
	// Only an end print — a missing start must not look like OI jumped from 0.
	points := []OptionsHistoryPoint{
		{Time: to.Add(-2 * time.Minute), OpenInterest: 120, HasOI: true, PutCallOI: 1.3, HasPutCallOI: true, DVOL: 48, HasDVOL: true},
	}
	if got := OptionsAcrossWindow(OptionsAssetBTC, points, from, to); got != nil {
		t.Fatalf("expected no change from a single sample, got %+v", got)
	}
}

func TestMarkAroundOptionsTiming_BeforeVsDuring(t *testing.T) {
	before := &AroundOptions{Complete: true, Unusual: true, HasOI: true, OIDirection: CVDDirUp, OIChangePct: 12}
	during := &AroundOptions{Complete: true, Unusual: true, HasOI: true, OIDirection: CVDDirUp, OIChangePct: 3}
	MarkAroundOptionsTiming(before, during)
	if before.Started != AroundOptionsStartedBefore || during.Started != AroundOptionsStartedBefore {
		t.Fatalf("continued print: before=%q during=%q", before.Started, during.Started)
	}

	fresh := &AroundOptions{Complete: true, Unusual: true, HasOI: true, OIDirection: CVDDirDown, OIChangePct: -10}
	quiet := &AroundOptions{Complete: true, Unusual: false, HasOI: true, OIDirection: CVDDirFlat}
	MarkAroundOptionsTiming(quiet, fresh)
	if fresh.Started != AroundOptionsStartedDuring {
		t.Fatalf("fresh during %q", fresh.Started)
	}

	pre := &AroundOptions{Complete: true, Unusual: true, HasOI: true, OIDirection: CVDDirUp, OIChangePct: 9}
	flip := &AroundOptions{Complete: true, Unusual: true, HasPutCall: true, LeanTo: OptionsLeanPuts, PutCallDelta: 0.3}
	MarkAroundOptionsTiming(pre, flip)
	if pre.Started != AroundOptionsStartedBefore || flip.Started != AroundOptionsStartedDuring {
		t.Fatalf("new print during: before=%q during=%q", pre.Started, flip.Started)
	}
}

func TestOptionsHistoryUntil(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	in := []OptionsHistoryPoint{
		{Time: t0, HasOI: true, OpenInterest: 1},
		{Time: t0.Add(time.Hour), HasOI: true, OpenInterest: 2},
		{Time: t0.Add(2 * time.Hour), HasOI: true, OpenInterest: 3},
	}
	got := OptionsHistoryUntil(in, t0.Add(time.Hour))
	if len(got) != 2 || got[1].OpenInterest != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestExplainAroundPhase_UnusualOptions(t *testing.T) {
	p := AroundPhase{
		Phase:    AroundPhaseBefore,
		Complete: true,
		Price:    AroundPrice{Open: 100, Close: 100.2, ChangePct: 0.2, Direction: CVDDirFlat},
		Flow:     AroundFlow{Volume: 1000},
		Options: &AroundOptions{
			Complete: true, Unusual: true, HasOI: true, OIChangePct: 8, OIDirection: CVDDirUp,
			Summary: "Unusual options print — listed open interest up +8.00%.",
		},
	}
	got := ExplainAroundPhase(p)
	if !strings.Contains(got, "unusual options") {
		t.Fatalf("%q", got)
	}
}
