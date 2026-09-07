package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestNeedsTape_OffByDefault(t *testing.T) {
	t.Parallel()
	def := DefaultCustomSignalWeights()
	if def.NeedsTape() || def.NeedsForensics() || def.NeedsSimilarSetup() {
		t.Fatalf("%+v", def)
	}
	w, err := ParseCustomSignalWeightMap(map[string]float64{
		CustomFactorRSIRecovery:  40,
		CustomFactorForensics:    30,
		CustomFactorSimilarSetup: 30,
	})
	if err != nil || !w.NeedsTape() || !w.NeedsForensics() || !w.NeedsSimilarSetup() {
		t.Fatalf("%+v %v", w, err)
	}
}

func TestSimilarSetupFactorRaw_SupportAndAgainst(t *testing.T) {
	t.Parallel()
	if got := SimilarSetupFactorRaw(nil, AroundSimilarHorizon4h); got.Status != CustomFactorMissing {
		t.Fatalf("%+v", got)
	}
	empty := &AroundSimilarReport{}
	if got := SimilarSetupFactorRaw(empty, AroundSimilarHorizon4h); got.Status != CustomFactorMissing {
		t.Fatalf("empty %+v", got)
	}
	up := &AroundSimilarReport{
		Matches: []AroundSimilarHit{
			hitAfter(AroundSimilarHorizon1h, CVDDirUp, AroundSimilarHit{Similarity: 80}),
			hitAfter(AroundSimilarHorizon1h, CVDDirUp, AroundSimilarHit{Similarity: 78}),
		},
	}
	got := SimilarSetupFactorRaw(up, AroundSimilarHorizon1h)
	if got.Status != CustomFactorUsed || got.Score <= 50 {
		t.Fatalf("support %+v", got)
	}
	if !strings.Contains(got.Detail, "supports") {
		t.Fatalf("detail %q", got.Detail)
	}
	down := &AroundSimilarReport{
		Matches: []AroundSimilarHit{
			hitAfter(AroundSimilarHorizon1h, CVDDirDown, AroundSimilarHit{Similarity: 80}),
			hitAfter(AroundSimilarHorizon1h, CVDDirDown, AroundSimilarHit{Similarity: 78}),
		},
	}
	got = SimilarSetupFactorRaw(down, AroundSimilarHorizon1h)
	if got.Status != CustomFactorUsed || got.Score >= 50 {
		t.Fatalf("against %+v", got)
	}
	if !strings.Contains(got.Detail, "goes against") {
		t.Fatalf("detail %q", got.Detail)
	}
	mixed := &AroundSimilarReport{
		Matches: []AroundSimilarHit{
			hitAfter(AroundSimilarHorizon1h, CVDDirUp, AroundSimilarHit{Similarity: 70}),
			hitAfter(AroundSimilarHorizon1h, CVDDirDown, AroundSimilarHit{Similarity: 70}),
		},
	}
	if got = SimilarSetupFactorRaw(mixed, AroundSimilarHorizon1h); got.Status != CustomFactorMissing {
		t.Fatalf("mixed %+v", got)
	}
}

func TestSimilarSetupFactorRaw_UsesSignalHorizonNotOneHour(t *testing.T) {
	t.Parallel()
	both := AroundSimilarHit{
		Similarity: 80,
		HorizonAfter: []AroundSimilarHorizonAfter{
			{Horizon: AroundSimilarHorizon1h, Dir: CVDDirUp, Pct: 2},
			{Horizon: AroundSimilarHorizon4h, Dir: CVDDirDown, Pct: -3},
		},
	}
	rep := &AroundSimilarReport{Matches: []AroundSimilarHit{both, both}}
	// 4h swing must use the 4h after-path (against), not the 1h rise.
	got := SimilarSetupFactorRaw(rep, AroundSimilarHorizon4h)
	if got.Status != CustomFactorUsed || got.Score >= 50 {
		t.Fatalf("4h against %+v", got)
	}
	if !strings.Contains(got.Detail, "4h") {
		t.Fatalf("detail should name 4h: %q", got.Detail)
	}
	oneH := SimilarSetupFactorRaw(rep, AroundSimilarHorizon1h)
	if oneH.Status != CustomFactorUsed || oneH.Score <= 50 {
		t.Fatalf("1h support %+v", oneH)
	}
}

func TestCustomTapeQuality_MissingTapeIsNotFullEvidence(t *testing.T) {
	t.Parallel()
	fullHit := hitAfter(AroundSimilarHorizon4h, CVDDirUp, completeTapeHit(80))
	full := &AroundSimilarReport{Matches: []AroundSimilarHit{fullHit, fullHit}}
	thinHit := hitAfter(AroundSimilarHorizon4h, CVDDirUp, partialTapeHit(80))
	thin := &AroundSimilarReport{Matches: []AroundSimilarHit{thinHit, thinHit}}
	a := SimilarSetupFactorRaw(full, AroundSimilarHorizon4h)
	b := SimilarSetupFactorRaw(thin, AroundSimilarHorizon4h)
	if a.Status != CustomFactorUsed || b.Status != CustomFactorUsed {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
	if b.Score >= a.Score {
		t.Fatalf("thin tape should be weaker evidence: full=%.1f thin=%.1f", a.Score, b.Score)
	}
	if !strings.Contains(a.Detail, "Used 2 of 2 setups") {
		t.Fatalf("full detail %q", a.Detail)
	}
	gone := &AroundSimilarReport{
		Matches: []AroundSimilarHit{hitAfter(AroundSimilarHorizon4h, CVDDirUp, missingTapeHit(80))},
	}
	got := SimilarSetupFactorRaw(gone, AroundSimilarHorizon4h)
	if got.Status != CustomFactorMissing {
		t.Fatalf("mostly missing tape must not be used: %+v", got)
	}
	if !strings.Contains(got.Detail, "Used 0 of 1 setups; 1 had missing tape") {
		t.Fatalf("gone detail %q", got.Detail)
	}
}

func TestFreshFuturesSnapshotAtOrBefore_DropsStale(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	rows := []FuturesSnapshot{
		{SampledAt: at.Add(-40 * time.Minute), Value: 1000},
	}
	if got := FreshFuturesSnapshotAtOrBefore(rows, at, AroundTapeFreshSlack); got != nil {
		t.Fatalf("stale %+v", got)
	}
	rows = append(rows, FuturesSnapshot{SampledAt: at.Add(-3 * time.Minute), Value: 1100})
	got := FreshFuturesSnapshotAtOrBefore(rows, at, AroundTapeFreshSlack)
	if got == nil || got.Value != 1100 {
		t.Fatalf("%+v", got)
	}
}

func TestCustomSignalTapeQueryFor_FourHourSwing(t *testing.T) {
	t.Parallel()
	q := CustomSignalTapeQueryFor("4h")
	if q.Window != "4h" || q.Horizon != AroundSimilarHorizon4h || q.MoveInterval != "1h" {
		t.Fatalf("%+v", q)
	}
	h := CustomSignalTapeQueryFor("1h")
	if h.Window != "1h" || h.Horizon != AroundSimilarHorizon1h {
		t.Fatalf("%+v", h)
	}
}

func TestForensicsFactorRaw_ScalesWithSimilarity(t *testing.T) {
	t.Parallel()
	if got := ForensicsFactorRaw(nil, AroundSimilarHorizon4h); got.Status != CustomFactorMissing {
		t.Fatalf("%+v", got)
	}
	strong := &AroundSimilarReport{
		Matches: []AroundSimilarHit{hitAfter(AroundSimilarHorizon1h, CVDDirUp, AroundSimilarHit{Similarity: 90, AfterDirection: CVDDirUp})},
	}
	weak := &AroundSimilarReport{
		Matches: []AroundSimilarHit{hitAfter(AroundSimilarHorizon1h, CVDDirUp, AroundSimilarHit{Similarity: 45, AfterDirection: CVDDirUp})},
	}
	a := ForensicsFactorRaw(strong, AroundSimilarHorizon1h)
	b := ForensicsFactorRaw(weak, AroundSimilarHorizon1h)
	if a.Status != CustomFactorUsed || b.Status != CustomFactorUsed {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
	if a.Score <= b.Score {
		t.Fatalf("strong sim should lean harder: %.1f vs %.1f", a.Score, b.Score)
	}
	if !strings.Contains(a.Detail, "supports") || !strings.Contains(a.Detail, "similar") {
		t.Fatalf("detail %q", a.Detail)
	}
}

func TestScoreCustomSignal_TapeAgainstPullsDown(t *testing.T) {
	t.Parallel()
	raws := BuildCustomSignalRaws([]SwingPattern{
		{Name: CustomFactorEMACrossover, Score: 90, Description: "cross"},
	}, nil)
	raws[CustomFactorSimilarSetup] = SimilarSetupFactorRaw(&AroundSimilarReport{
		Matches: []AroundSimilarHit{
			hitAfter(AroundSimilarHorizon4h, CVDDirDown, AroundSimilarHit{Similarity: 80}),
			hitAfter(AroundSimilarHorizon4h, CVDDirDown, AroundSimilarHit{Similarity: 78}),
		},
	}, AroundSimilarHorizon4h)
	w := CustomSignalWeights{Source: CustomSignalMixCustom, Pct: map[string]float64{
		CustomFactorEMACrossover: 50,
		CustomFactorSimilarSetup: 50,
	}}
	got := ScoreCustomSignal(raws, w)
	if got.Score >= 82 {
		t.Fatalf("against similar should pull below EMA-only, got %.1f", got.Score)
	}
	var sim CustomSignalFactor
	for _, f := range got.Factors {
		if f.ID == CustomFactorSimilarSetup {
			sim = f
		}
	}
	if sim.Status != CustomFactorUsed || sim.Effect >= 0 {
		t.Fatalf("similar %+v", sim)
	}
}

func TestClassifyCustomTapeMatches_DropsThinFromAfterPath(t *testing.T) {
	t.Parallel()
	complete := hitAfter(AroundSimilarHorizon4h, CVDDirUp, completeTapeHit(82))
	thin := hitAfter(AroundSimilarHorizon4h, CVDDirDown, missingTapeHit(80))
	rep := &AroundSimilarReport{
		Matches: []AroundSimilarHit{complete, complete, complete, thin, thin},
	}
	usable, stats, _ := ClassifyCustomTapeMatches(rep.Matches)
	if stats.Setups != 5 || stats.Used != 3 || stats.Thin != 2 {
		t.Fatalf("stats %+v", stats)
	}
	if len(usable) != 3 {
		t.Fatalf("usable %d", len(usable))
	}
	up, down, when := similarAfterCountsFromHits(usable, AroundSimilarHorizon4h)
	if up != 3 || down != 0 || when != AroundSimilarHorizon4h {
		t.Fatalf("after from usable only: up=%d down=%d when=%s", up, down, when)
	}
	// Report-wide mix would be 3 up / 2 down → mixed. Complete setups rose.
	got := SimilarSetupFactorRaw(rep, AroundSimilarHorizon4h)
	if got.Status != CustomFactorUsed || got.Score <= 50 {
		t.Fatalf("complete setups support a long: %+v", got)
	}
	if !strings.Contains(got.Detail, "Used 3 of 5 setups; 2 had missing tape") {
		t.Fatalf("detail %q", got.Detail)
	}
	if !strings.Contains(got.Detail, "supports") {
		t.Fatalf("detail should use complete after-path: %q", got.Detail)
	}
}

func TestForensicsFactorRaw_UsesEveryCompleteSetup(t *testing.T) {
	t.Parallel()
	complete := hitAfter(AroundSimilarHorizon4h, CVDDirUp, completeTapeHit(90))
	thin := hitAfter(AroundSimilarHorizon4h, CVDDirDown, missingTapeHit(88))
	rep := &AroundSimilarReport{
		Matches: []AroundSimilarHit{thin, complete, complete, complete, thin},
	}
	got := ForensicsFactorRaw(rep, AroundSimilarHorizon4h)
	if got.Status != CustomFactorUsed || got.Score <= 50 {
		t.Fatalf("must not follow the thin first hit: %+v", got)
	}
	if !strings.Contains(got.Detail, "Used 3 of 5 setups; 2 had missing tape") {
		t.Fatalf("detail %q", got.Detail)
	}
}

func hitAfter(horizon, dir string, extra AroundSimilarHit) AroundSimilarHit {
	extra.HorizonAfter = append(append([]AroundSimilarHorizonAfter(nil), extra.HorizonAfter...), AroundSimilarHorizonAfter{
		Horizon: horizon, Dir: dir, Pct: 2,
	})
	if dir == CVDDirDown {
		extra.HorizonAfter[len(extra.HorizonAfter)-1].Pct = -2
	}
	return extra
}

func completeTapeHit(sim float64) AroundSimilarHit {
	return AroundSimilarHit{
		Similarity: sim, Coverage: 100,
		Used: []string{"price", "volume", "takers", "oi", "book"},
		Compared: []AroundSimilarFieldScore{
			{Name: AroundSimilarFieldTakers, Used: true, Weight: 1.5},
			{Name: AroundSimilarFieldOI, Used: true, Weight: 3},
			{Name: AroundSimilarFieldBook, Used: true, Weight: 3},
		},
	}
}

func partialTapeHit(sim float64) AroundSimilarHit {
	return AroundSimilarHit{
		Similarity: sim, Coverage: 62,
		Missing: []string{AroundSimilarFieldTakers, AroundSimilarFieldOI},
		Compared: []AroundSimilarFieldScore{
			{Name: AroundSimilarFieldTakers, Used: false, Weight: 1.5},
			{Name: AroundSimilarFieldOI, Used: false, Weight: 3},
			{Name: AroundSimilarFieldBook, Used: true, Weight: 3},
		},
	}
}

func missingTapeHit(sim float64) AroundSimilarHit {
	return AroundSimilarHit{
		Similarity: sim, Coverage: 40,
		Missing: []string{AroundSimilarFieldTakers, AroundSimilarFieldOI, AroundSimilarFieldBook, AroundSimilarFieldOptions},
		Compared: []AroundSimilarFieldScore{
			{Name: AroundSimilarFieldTakers, Used: false, Weight: 1.5},
			{Name: AroundSimilarFieldOI, Used: false, Weight: 3},
			{Name: AroundSimilarFieldBook, Used: false, Weight: 3},
			{Name: AroundSimilarFieldOptions, Used: false, Weight: 2},
		},
	}
}

func TestScoreCustomSignal_DisabledTapeDoesNotLoad(t *testing.T) {
	t.Parallel()
	raws := BuildCustomSignalRaws([]SwingPattern{
		{Name: CustomFactorEMACrossover, Score: 90},
	}, nil)
	got := ScoreCustomSignal(raws, DefaultCustomSignalWeights())
	for _, f := range got.Factors {
		if f.ID == CustomFactorForensics || f.ID == CustomFactorSimilarSetup {
			if f.Status != CustomFactorDisabled {
				t.Fatalf("%s %+v", f.ID, f)
			}
			if math.Abs(f.Effect) > 0.01 {
				t.Fatalf("disabled tape must not move score: %+v", f)
			}
		}
	}
}
