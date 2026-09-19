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

func TestOptionsAcrossWindow_AttachesGammaState(t *testing.T) {
	base := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	points := []OptionsHistoryPoint{
		{Time: base, OpenInterest: 100, HasOI: true, HasGamma: true, GammaRegime: GammaRegimeLong, GammaNetUSD: 400},
		{
			Time: base.Add(time.Hour), OpenInterest: 102, HasOI: true,
			HasGamma: true, GammaRegime: GammaRegimeShort, GammaNetUSD: -200,
			HasGammaFlip: true, GammaFlip: 97200, GammaFlipSide: GammaFlipAbove,
			HasIndex: true, IndexPrice: 100000,
		},
	}
	got := OptionsAcrossWindow(OptionsAssetBTC, points, base, base.Add(time.Hour))
	if got == nil || !got.HasGamma || got.GammaRegime != GammaRegimeShort {
		t.Fatalf("gamma %+v", got)
	}
	if !got.HasGammaFlip || got.GammaFlipSide != GammaFlipAbove || !got.HasGammaRegimeChange {
		t.Fatalf("flip %+v", got)
	}
	if !strings.Contains(got.Summary, "short gamma") || !strings.Contains(got.Summary, "above the flip") {
		t.Fatalf("summary %q", got.Summary)
	}
}

func TestOptionsGammaAt_OneSample(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	points := []OptionsHistoryPoint{{
		Time: at.Add(-3 * time.Minute), HasGamma: true, GammaRegime: GammaRegimeLong,
		HasGammaFlip: true, GammaFlip: 101000, GammaFlipSide: GammaFlipBelow,
	}}
	got := OptionsGammaAt(OptionsAssetBTC, points, at)
	if got == nil || got.GammaRegime != GammaRegimeLong || got.GammaFlipSide != GammaFlipBelow {
		t.Fatalf("%+v", got)
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

func TestMeasureAroundOptionsLead_ReportsHowMuchEarlier(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	var points []OptionsHistoryPoint
	for i := 24; i >= 0; i-- {
		t0 := at.Add(-time.Duration(i) * 5 * time.Minute)
		oi := 100.0
		if !t0.Before(at.Add(-40 * time.Minute)) {
			oi = 120
		}
		points = append(points, OptionsHistoryPoint{Time: t0, OpenInterest: oi, HasOI: true})
	}
	o := &AroundOptions{
		Complete: true, Unusual: true, Started: AroundOptionsStartedBefore,
		HasOI: true, OIChangePct: 20, OIDirection: CVDDirUp,
	}
	MeasureAroundOptionsLead(o, points, at)
	if o.Lead < 30*time.Minute || o.Lead > 50*time.Minute {
		t.Fatalf("lead %s label %q", o.Lead, o.LeadLabel)
	}
	if !strings.Contains(o.LeadLabel, "40 minutes") && !strings.Contains(o.LeadLabel, "35 minutes") && !strings.Contains(o.LeadLabel, "45 minutes") {
		t.Fatalf("lead label %q", o.LeadLabel)
	}
}

func TestMeasureAroundOptionsLead_FourHourMoveUsesFourHourStart(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	var points []OptionsHistoryPoint
	// Slow 4h grind: ~3.6% per hour, 15% over 4h. 1h never hits the 5% floor.
	for i := 48; i >= 0; i-- {
		t0 := at.Add(-time.Duration(i) * 5 * time.Minute)
		elapsed := at.Sub(t0)
		oi := 100.0
		if elapsed <= 4*time.Hour {
			oi = 100 + 15*(1-float64(elapsed)/float64(4*time.Hour))
		}
		points = append(points, OptionsHistoryPoint{Time: t0, OpenInterest: oi, HasOI: true})
	}
	o := &AroundOptions{
		Complete: true, Unusual: true, Started: AroundOptionsStartedBefore,
		HasOI: true, OIChangePct: 15, OIDirection: CVDDirUp,
	}
	MeasureAroundOptionsLead(o, points, at)
	if o.LeadWindow != OptionsHistoryWindow4h {
		t.Fatalf("window %q lead %s", o.LeadWindow, o.Lead)
	}
	if o.Lead < 3*time.Hour || o.Lead > 5*time.Hour {
		t.Fatalf("4h move lead %s label %q", o.Lead, o.LeadLabel)
	}
}

func TestMeasureAroundOptionsLead_TwoHourMoveIsNotFourHours(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	var points []OptionsHistoryPoint
	// Flat for 2h, then +10% over the last 2h. 4h change is unusual; the start is not 4h.
	for i := 48; i >= 0; i-- {
		t0 := at.Add(-time.Duration(i) * 5 * time.Minute)
		elapsed := at.Sub(t0)
		oi := 100.0
		if elapsed <= 2*time.Hour {
			oi = 100 + 10*(1-float64(elapsed)/float64(2*time.Hour))
		}
		points = append(points, OptionsHistoryPoint{Time: t0, OpenInterest: oi, HasOI: true})
	}
	o := &AroundOptions{
		Complete: true, Unusual: true, Started: AroundOptionsStartedBefore,
		HasOI: true, OIChangePct: 10, OIDirection: CVDDirUp,
	}
	MeasureAroundOptionsLead(o, points, at)
	if o.Lead < 90*time.Minute || o.Lead > 150*time.Minute {
		t.Fatalf("2h move should not be dated as 4h: lead %s window %q", o.Lead, o.LeadWindow)
	}
}

func TestMeasureAroundOptionsLead_TwentyFourHourFlagIsNotTheStart(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	var points []OptionsHistoryPoint
	// Flat for 18h, then +15% over the last 6h. 24h change is unusual; the start is not 24h.
	for i := 288; i >= 0; i-- {
		t0 := at.Add(-time.Duration(i) * 5 * time.Minute)
		elapsed := at.Sub(t0)
		oi := 100.0
		if elapsed <= 6*time.Hour {
			oi = 100 + 15*(1-float64(elapsed)/float64(6*time.Hour))
		}
		points = append(points, OptionsHistoryPoint{Time: t0, OpenInterest: oi, HasOI: true})
	}
	o := &AroundOptions{
		Complete: true, Unusual: true, Started: AroundOptionsStartedBefore,
		HasOI: true, OIChangePct: 15, OIDirection: CVDDirUp,
	}
	MeasureAroundOptionsLead(o, points, at)
	if o.Lead < 5*time.Hour || o.Lead > 7*time.Hour {
		t.Fatalf("6h start inside a 24h flag: lead %s window %q", o.Lead, o.LeadWindow)
	}
}

func TestMarkAroundOptionsAfter_ContinuedAndReversed(t *testing.T) {
	before := &AroundOptions{Complete: true, Unusual: true, HasOI: true, OIDirection: CVDDirUp, OIChangePct: 12}
	kept := &AroundOptions{Complete: true, HasOI: true, OIDirection: CVDDirUp, OIChangePct: 6}
	MarkAroundOptionsAfter(before, kept)
	if kept.After != AroundOptionsAfterContinued {
		t.Fatalf("continued %q %q", kept.After, kept.AfterSummary)
	}
	turned := &AroundOptions{Complete: true, HasOI: true, OIDirection: CVDDirDown, OIChangePct: -8}
	MarkAroundOptionsAfter(before, turned)
	if turned.After != AroundOptionsAfterReversed || !strings.Contains(turned.AfterSummary, "turned the other way") {
		t.Fatalf("reversed %q %q", turned.After, turned.AfterSummary)
	}
	quiet := &AroundOptions{Complete: true, HasOI: true, OIDirection: CVDDirFlat, OIChangePct: 0.2}
	MarkAroundOptionsAfter(before, quiet)
	if quiet.After != AroundOptionsAfterFaded {
		t.Fatalf("faded %q", quiet.After)
	}
}

func TestFillAroundOptionsWeight_ClusterBeatsSingle(t *testing.T) {
	one := AroundOptions{Complete: true, Unusual: true, HasOI: true, OIChangePct: 12, OIDirection: CVDDirUp}
	fillAroundOptionsWeight(&one)
	if one.Together != 1 || one.Weight != AroundOptionsWeightSingle {
		t.Fatalf("single %+v", one)
	}
	three := AroundOptions{
		Complete: true, Unusual: true,
		HasOI: true, OIChangePct: 12, OIDirection: CVDDirUp,
		HasPutCall: true, PutCallDelta: 0.2, LeanTo: OptionsLeanPuts,
		HasDVOL: true, DVOLDelta: 4,
	}
	fillAroundOptionsWeight(&three)
	if three.Together != 3 || three.Weight != AroundOptionsWeightCluster {
		t.Fatalf("cluster %+v", three)
	}
	s := unusualOptionsLeadSentence(three)
	if !strings.Contains(s, "all moved together") {
		t.Fatalf("sentence %q", s)
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
