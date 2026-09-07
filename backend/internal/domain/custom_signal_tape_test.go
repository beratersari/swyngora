package domain

import (
	"math"
	"strings"
	"testing"
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
	if got := SimilarSetupFactorRaw(nil); got.Status != CustomFactorMissing {
		t.Fatalf("%+v", got)
	}
	empty := &AroundSimilarReport{}
	if got := SimilarSetupFactorRaw(empty); got.Status != CustomFactorMissing {
		t.Fatalf("empty %+v", got)
	}
	up := &AroundSimilarReport{
		Matches: []AroundSimilarHit{{Similarity: 80}},
		UpAfter: 5, DownAfter: 1,
		AfterHorizons: []AroundSimilarHorizonStat{{Horizon: "1h", Sample: 6, Up: 5, Down: 1}},
	}
	got := SimilarSetupFactorRaw(up)
	if got.Status != CustomFactorUsed || got.Score <= 50 {
		t.Fatalf("support %+v", got)
	}
	if !strings.Contains(got.Detail, "supports") {
		t.Fatalf("detail %q", got.Detail)
	}
	down := &AroundSimilarReport{
		Matches: []AroundSimilarHit{{Similarity: 80}},
		UpAfter: 1, DownAfter: 5,
		AfterHorizons: []AroundSimilarHorizonStat{{Horizon: "1h", Sample: 6, Up: 1, Down: 5}},
	}
	got = SimilarSetupFactorRaw(down)
	if got.Status != CustomFactorUsed || got.Score >= 50 {
		t.Fatalf("against %+v", got)
	}
	if !strings.Contains(got.Detail, "goes against") {
		t.Fatalf("detail %q", got.Detail)
	}
	mixed := &AroundSimilarReport{
		Matches: []AroundSimilarHit{{Similarity: 70}},
		UpAfter: 3, DownAfter: 3,
		AfterHorizons: []AroundSimilarHorizonStat{{Horizon: "1h", Sample: 6, Up: 3, Down: 3}},
	}
	if got = SimilarSetupFactorRaw(mixed); got.Status != CustomFactorMissing {
		t.Fatalf("mixed %+v", got)
	}
}

func TestForensicsFactorRaw_ScalesWithSimilarity(t *testing.T) {
	t.Parallel()
	if got := ForensicsFactorRaw(nil); got.Status != CustomFactorMissing {
		t.Fatalf("%+v", got)
	}
	strong := &AroundSimilarReport{
		Matches: []AroundSimilarHit{{Similarity: 90, AfterDirection: CVDDirUp}},
		UpAfter: 4, DownAfter: 0, Events: 4,
		AfterHorizons: []AroundSimilarHorizonStat{{Horizon: "1h", Sample: 4, Up: 4, Down: 0}},
	}
	weak := &AroundSimilarReport{
		Matches: []AroundSimilarHit{{Similarity: 45, AfterDirection: CVDDirUp}},
		UpAfter: 4, DownAfter: 0, Events: 4,
		AfterHorizons: []AroundSimilarHorizonStat{{Horizon: "1h", Sample: 4, Up: 4, Down: 0}},
	}
	a := ForensicsFactorRaw(strong)
	b := ForensicsFactorRaw(weak)
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
		Matches: []AroundSimilarHit{{Similarity: 80}},
		UpAfter: 0, DownAfter: 5,
		AfterHorizons: []AroundSimilarHorizonStat{{Horizon: "1h", Sample: 5, Up: 0, Down: 5}},
	})
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
