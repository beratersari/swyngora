package domain

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseCustomSignalWeights_DefaultAndCustom(t *testing.T) {
	t.Parallel()
	def, err := ParseCustomSignalWeights(func(string) string { return "" })
	if err != nil || def.Source != CustomSignalMixDefault || def.Pct[CustomFactorEMACrossover] != 16 {
		t.Fatalf("%+v %v", def, err)
	}
	if def.NeedsHunt() {
		t.Fatal("default mix must not load hunt")
	}

	get := func(k string) string {
		switch k {
		case "weightEmaCrossover":
			return "40"
		case "weightHuntUp":
			return "60"
		default:
			return ""
		}
	}
	got, err := ParseCustomSignalWeights(get)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != CustomSignalMixCustom || got.Pct[CustomFactorEMACrossover] != 40 || got.Pct[CustomFactorHuntUp] != 60 {
		t.Fatalf("%+v", got)
	}
	if !got.NeedsHunt() {
		t.Fatal("expected hunt on")
	}
	if got.Pct[CustomFactorMACD] != 0 {
		t.Fatalf("omitted factor should be 0, got %v", got.Pct[CustomFactorMACD])
	}

	_, err = ParseCustomSignalWeights(func(k string) string {
		if k == "weightEmaCrossover" {
			return "50"
		}
		return ""
	})
	if err == nil {
		t.Fatal("sum 50 must fail")
	}
}

func TestParseCustomSignalWeightMap(t *testing.T) {
	t.Parallel()
	w, err := ParseCustomSignalWeightMap(map[string]float64{
		CustomFactorRSIRecovery: 55,
		CustomFactorHuntUp:      45,
	})
	if err != nil || w.Pct[CustomFactorRSIRecovery] != 55 || w.Pct[CustomFactorSupertrend] != 0 {
		t.Fatalf("%+v %v", w, err)
	}
	_, err = ParseCustomSignalWeightMap(map[string]float64{CustomFactorHuntUp: -1})
	if err == nil {
		t.Fatal("negative must fail")
	}
}

func TestScoreCustomSignal_MissingHuntOmitted(t *testing.T) {
	t.Parallel()
	raws := BuildCustomSignalRaws([]SwingPattern{
		{Name: CustomFactorEMACrossover, Score: 90, Description: "cross"},
	}, nil)
	w := CustomSignalWeights{Source: CustomSignalMixCustom, Pct: map[string]float64{
		CustomFactorEMACrossover: 50,
		CustomFactorHuntUp:       50,
	}}
	got := ScoreCustomSignal(raws, w)
	// Only EMA is present: same as engine on one pattern → 90 * (0.9 + 0.1/5) = 82.8
	if math.Abs(got.Score-82.8) > 0.05 {
		t.Fatalf("score=%v want 82.8 factors=%+v", got.Score, got.Factors)
	}
	var hunt CustomSignalFactor
	for _, f := range got.Factors {
		if f.ID == CustomFactorHuntUp {
			hunt = f
		}
	}
	if hunt.Status != CustomFactorMissing || hunt.Effect != 0 {
		t.Fatalf("hunt %+v", hunt)
	}
}

func TestScoreCustomSignal_IdlePatternDoesNotPullDown(t *testing.T) {
	t.Parallel()
	raws := BuildCustomSignalRaws(nil, &CustomSignalRaw{Score: 80, Status: CustomFactorUsed, Detail: "up"})
	w := CustomSignalWeights{Source: CustomSignalMixCustom, Pct: map[string]float64{
		CustomFactorEMACrossover: 50,
		CustomFactorHuntUp:       50,
	}}
	got := ScoreCustomSignal(raws, w)
	// Idle EMA is omitted; only hunt 80 with one-factor breadth → 80 * 0.92 = 73.6
	if math.Abs(got.Score-73.6) > 0.05 {
		t.Fatalf("score=%v want 73.6 mix=%+v", got.Score, got.Mix)
	}
	var ema CustomSignalFactor
	for _, f := range got.Factors {
		if f.ID == CustomFactorEMACrossover {
			ema = f
		}
	}
	if ema.Status != CustomFactorIdle || ema.Effect != 0 {
		t.Fatalf("ema %+v", ema)
	}
}

func TestScoreCustomSignal_DefaultMatchesEngine(t *testing.T) {
	t.Parallel()
	pats := []SwingPattern{
		{Name: CustomFactorEMACrossover, Score: 90},
		{Name: CustomFactorVolumeBreakout, Score: 70},
	}
	engine := computeSwingScore(pats)
	got := ScoreCustomSignal(BuildCustomSignalRaws(pats, nil), DefaultCustomSignalWeights())
	if math.Abs(got.Score-engine) > 0.05 {
		t.Fatalf("custom=%v engine=%v", got.Score, engine)
	}
}

func TestAlignCustomDefaultToEngine(t *testing.T) {
	t.Parallel()
	cmp := CompareCustomSignal(BuildCustomSignalRaws([]SwingPattern{
		{Name: CustomFactorEMACrossover, Score: 90},
	}, nil), DefaultCustomSignalWeights())
	AlignCustomDefaultToEngine(&cmp, 61.5)
	if cmp.Default.Score != 61.5 {
		t.Fatalf("%+v", cmp.Default)
	}
}

func TestDuplicateCustomSignalProfileName(t *testing.T) {
	t.Parallel()
	if DuplicateCustomSignalProfileName("Hunt lean") != "Hunt lean (copy)" {
		t.Fatal(DuplicateCustomSignalProfileName("Hunt lean"))
	}
	long := strings.Repeat("a", MaxCustomSignalProfileNameLen)
	got := DuplicateCustomSignalProfileName(long)
	if utf8.RuneCountInString(got) > MaxCustomSignalProfileNameLen || !strings.HasSuffix(got, "(copy)") {
		t.Fatalf("%q len=%d", got, utf8.RuneCountInString(got))
	}
}

func TestCompareCustomSignal_LargestChange(t *testing.T) {
	t.Parallel()
	raws := BuildCustomSignalRaws([]SwingPattern{
		{Name: CustomFactorEMACrossover, Score: 90},
		{Name: CustomFactorVolumeBreakout, Score: 20},
	}, &CustomSignalRaw{Score: 80, Status: CustomFactorUsed})
	applied := CustomSignalWeights{Source: CustomSignalMixCustom, Pct: map[string]float64{
		CustomFactorHuntUp:         100,
		CustomFactorEMACrossover:   0,
		CustomFactorVolumeBreakout: 0,
	}}
	cmp := CompareCustomSignal(raws, applied)
	if cmp.Applied.Score <= cmp.Default.Score && cmp.Delta.LargestChange == nil {
		t.Fatalf("%+v", cmp)
	}
	if cmp.Delta.LargestChange == nil || cmp.Delta.LargestChange.ID == "" {
		t.Fatalf("expected largest change %+v", cmp.Delta)
	}
}

func TestHuntUpRaw(t *testing.T) {
	t.Parallel()
	if HuntUpRaw(nil).Status != CustomFactorMissing {
		t.Fatal("nil report")
	}
	rep := &HuntReport{Bias: &HuntBias{
		UpScore: 72, DownScore: 41, Lean: HuntLeanUp,
		Coverage: HuntCoverage{Usable: true},
	}}
	got := HuntUpRaw(rep)
	if got.Status != CustomFactorUsed || got.Score != 72 {
		t.Fatalf("%+v", got)
	}
	rep.Bias.Coverage.Usable = false
	if HuntUpRaw(rep).Status != CustomFactorMissing {
		t.Fatal("unusable must be missing")
	}
}

func TestNormalizeCustomSignalProfileName(t *testing.T) {
	t.Parallel()
	if _, err := NormalizeCustomSignalProfileName("  "); err == nil {
		t.Fatal("empty")
	}
	got, err := NormalizeCustomSignalProfileName("  Hunt lean  ")
	if err != nil || got != "Hunt lean" {
		t.Fatalf("%q %v", got, err)
	}
	long := make([]rune, MaxCustomSignalProfileNameLen+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := NormalizeCustomSignalProfileName(string(long)); err == nil {
		t.Fatal("too long")
	}
}

func TestCustomSignalCatalog(t *testing.T) {
	t.Parallel()
	cat := CustomSignalCatalog()
	if len(cat) != len(CustomSignalFactorIDs) {
		t.Fatalf("len=%d", len(cat))
	}
	var huntPct float64
	for _, r := range cat {
		if r.ID == CustomFactorHuntUp {
			if r.Group != "hunt" {
				t.Fatalf("%+v", r)
			}
			huntPct = r.DefaultPct
		}
	}
	if huntPct != 0 {
		t.Fatalf("hunt default %v", huntPct)
	}
}
