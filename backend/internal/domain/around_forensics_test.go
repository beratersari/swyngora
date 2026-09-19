package domain

import (
	"strings"
	"testing"
	"time"
)

func TestExplainForensics_Empty(t *testing.T) {
	if got := ExplainForensics(ForensicsReport{}); !strings.Contains(got, "not enough tape") && got != "No forensics tape for this time." {
		if !strings.Contains(got, "not enough") && !strings.Contains(got, "No forensics") {
			t.Fatalf("%q", got)
		}
	}
}

func TestExplainForensicsWhy_SqueezeAndSimilar(t *testing.T) {
	r := forensicsWhyFixture()
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.Headline, "BTC rose") {
		t.Fatalf("headline %q", why.Headline)
	}
	if !strings.Contains(why.Before, "bids were pulled") && !strings.Contains(why.Before, "less support") {
		t.Fatalf("before %q", why.Before)
	}
	if !strings.Contains(why.During, "short-cover") && !strings.Contains(why.During, "covering") && !strings.Contains(why.During, "Open interest fell") {
		t.Fatalf("during %q", why.During)
	}
	if !strings.Contains(why.After, "quiet") && !strings.Contains(why.After, "faded") {
		t.Fatalf("after %q", why.After)
	}
	if !strings.Contains(why.Similar, "more often kept rising") && !strings.Contains(why.Similar, "more often rose") {
		t.Fatalf("similar %q", why.Similar)
	}
	if !strings.Contains(why.Similar, "closest") {
		t.Fatalf("similar closest %q", why.Similar)
	}
	if why.Caution == "" {
		t.Fatal("missing caution")
	}
	// Must explain, not dump raw metric tables.
	if strings.Contains(why.During, "OIChangePct") || strings.Contains(why.Headline, "volume 9000") {
		t.Fatalf("looks like a metric dump: %+v", why)
	}
	joined := ExplainForensics(ForensicsReport{Why: why})
	if !strings.Contains(joined, why.Headline) {
		t.Fatalf("summary %q", joined)
	}
}

func TestExplainForensicsWhy_NewLongs(t *testing.T) {
	r := forensicsWhyFixture()
	during, _ := AroundPhaseByID(*r.Tape.Combined, AroundPhaseDuring)
	during.Futures.OIDirection = CVDDirUp
	during.Futures.OIChangePct = 3
	during.Futures.ShortLiq = 0
	for i := range r.Tape.Combined.Phases {
		if r.Tape.Combined.Phases[i].Phase == AroundPhaseDuring {
			r.Tape.Combined.Phases[i] = during
		}
	}
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.During, "new longs") {
		t.Fatalf("during %q", why.During)
	}
}

func TestExplainForensicsWhy_NoSimilar(t *testing.T) {
	r := forensicsWhyFixture()
	r.Similar = &AroundSimilarReport{Symbol: "BTCUSDT"}
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.Similar, "not back this read") && !strings.Contains(why.Similar, "similar enough") {
		t.Fatalf("similar %q", why.Similar)
	}
}

func TestExplainForensicsWhy_SimilarError(t *testing.T) {
	r := forensicsWhyFixture()
	r.Similar = nil
	r.SimilarError = "timeout"
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.Similar, "could not be loaded") {
		t.Fatalf("similar %q", why.Similar)
	}
}

func TestExplainForensicsWhy_UnusualOptionsBefore(t *testing.T) {
	r := forensicsWhyFixture()
	before, _ := AroundPhaseByID(*r.Tape.Combined, AroundPhaseBefore)
	before.Options = &AroundOptions{
		Complete: true, Unusual: true, Started: AroundOptionsStartedBefore,
		HasOI: true, OIDirection: CVDDirUp, OIChangePct: 9,
		HasDVOL: true, DVOLDelta: 3.2, Together: 2, Weight: AroundOptionsWeightPair,
		Lead: 40 * time.Minute, LeadLabel: "about 40 minutes",
		Summary: "Listed options were already unusual about 40 minutes before the price move.",
	}
	for i := range r.Tape.Combined.Phases {
		if r.Tape.Combined.Phases[i].Phase == AroundPhaseBefore {
			r.Tape.Combined.Phases[i] = before
		}
	}
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.Headline, "about 40 minutes") {
		t.Fatalf("headline %q", why.Headline)
	}
	if !strings.Contains(why.Before, "about 40 minutes") || !strings.Contains(why.Before, "two options series") {
		t.Fatalf("before %q", why.Before)
	}
}

func TestExplainForensicsWhy_GammaBefore(t *testing.T) {
	r := forensicsWhyFixture()
	before, _ := AroundPhaseByID(*r.Tape.Combined, AroundPhaseBefore)
	before.Options = &AroundOptions{
		Complete: true, HasGamma: true, GammaRegime: GammaRegimeShort,
		HasGammaFlip: true, GammaFlip: 97200, GammaFlipSide: GammaFlipAbove,
	}
	for i := range r.Tape.Combined.Phases {
		if r.Tape.Combined.Phases[i].Phase == AroundPhaseBefore {
			r.Tape.Combined.Phases[i] = before
		}
	}
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.Before, "short gamma") || !strings.Contains(why.Before, "above the flip") {
		t.Fatalf("before %q", why.Before)
	}
}

func TestExplainForensicsWhy_OptionsStartedDuring(t *testing.T) {
	r := forensicsWhyFixture()
	during, _ := AroundPhaseByID(*r.Tape.Combined, AroundPhaseDuring)
	during.Options = &AroundOptions{
		Complete: true, Unusual: true, Started: AroundOptionsStartedDuring,
		HasOI: true, OIDirection: CVDDirUp, OIChangePct: 11,
	}
	for i := range r.Tape.Combined.Phases {
		if r.Tape.Combined.Phases[i].Phase == AroundPhaseDuring {
			r.Tape.Combined.Phases[i] = during
		}
	}
	why := ExplainForensicsWhy(r)
	if !strings.Contains(why.Headline, "only became unusual") && !strings.Contains(why.During, "only became unusual") {
		t.Fatalf("headline %q during %q", why.Headline, why.During)
	}
}

func TestExplainForensicsWhy_EmptyTape(t *testing.T) {
	why := ExplainForensicsWhy(ForensicsReport{Symbol: "ETHUSDT"})
	if !strings.Contains(why.Headline, "not enough tape") {
		t.Fatalf("%q", why.Headline)
	}
}

func forensicsWhyFixture() ForensicsReport {
	before := AroundPhase{
		Phase:    AroundPhaseBefore,
		Complete: true,
		Price:    AroundPrice{Open: 100, Close: 100.4, ChangePct: 0.4, Direction: CVDDirUp},
		Flow: AroundFlow{
			Volume: 2000, TypicalKnown: true, VolumeRatio: 1.8, VolumeGrade: VolumeSurgeElevated,
			BuySellKnown: true, Dominant: TakerSideBuy,
		},
		Book:    &AroundBook{BidNotionalDelta: -4000, AskNotionalDelta: 200, WallsRemoved: 2, Complete: true},
		Futures: &AroundFutures{OIDirection: CVDDirDown, OIChangePct: -1.5, Complete: true},
	}
	during := AroundPhase{
		Phase:    AroundPhaseDuring,
		Complete: true,
		Price:    AroundPrice{Open: 100.4, Close: 107, ChangePct: 6.6, Direction: CVDDirUp},
		Flow: AroundFlow{
			Volume: 9000, TypicalKnown: true, VolumeRatio: 4.2, VolumeGrade: VolumeSurgeHigh,
			BuySellKnown: true, Dominant: TakerSideBuy, VWAP: 104, VsVWAP: VolumeProfileVsAbove,
		},
		Futures: &AroundFutures{OIDirection: CVDDirDown, OIChangePct: -3, ShortLiq: 8e6, LongLiq: 1e6, Complete: true},
	}
	after := AroundPhase{
		Phase:    AroundPhaseAfter,
		Complete: true,
		Price:    AroundPrice{Open: 107, Close: 106.8, ChangePct: -0.2, Direction: CVDDirFlat},
		Flow:     AroundFlow{Volume: 1500, TypicalKnown: true, VolumeRatio: 0.8, VolumeGrade: VolumeSurgeTypical},
	}
	venue := AroundVenue{
		Exchange: "all", Symbol: "BTCUSDT",
		Phases:  []AroundPhase{before, during, after},
		Changes: []AroundChange{{Metric: "price", Path: AroundPathFaded, Direction: CVDDirUp}},
	}
	return ForensicsReport{
		Symbol: "BTCUSDT",
		Tape:   &AroundReport{Symbol: "BTCUSDT", Combined: &venue},
		Similar: &AroundSimilarReport{
			Symbol: "BTCUSDT", Events: 4,
			Matches: []AroundSimilarHit{{
				Similarity: 82, AfterDirection: CVDDirUp, AfterReturnPct: 3.5,
			}},
			AfterHorizons: []AroundSimilarHorizonStat{
				{Horizon: "1h", Up: 3, Down: 1, Sample: 4, Events: 4, MedianPct: 2.1},
			},
		},
	}
}
