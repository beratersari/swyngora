package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestPearsonCorr_PerfectAndInverse(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}
	if c := PearsonCorr(x, y); math.Abs(c-1) > 1e-9 {
		t.Fatalf("perfect %v", c)
	}
	inv := []float64{-2, -4, -6, -8, -10}
	if c := PearsonCorr(x, inv); math.Abs(c+1) > 1e-9 {
		t.Fatalf("inverse %v", c)
	}
}

func TestPearsonCorr_Uncorrelated(t *testing.T) {
	// Quarter-cycle sine vs cosine samples.
	x := []float64{0, 1, 0, -1, 0, 1, 0, -1}
	y := []float64{1, 0, -1, 0, 1, 0, -1, 0}
	c := PearsonCorr(x, y)
	if math.Abs(c) > 0.05 {
		t.Fatalf("want ~0 got %v", c)
	}
}

func TestRegressionBeta(t *testing.T) {
	ref := []float64{1, 2, 3, 4, 5}
	asset := []float64{2, 4, 6, 8, 10}
	if b := RegressionBeta(asset, ref); math.Abs(b-2) > 1e-9 {
		t.Fatalf("beta %v", b)
	}
}

func TestSameDirectionPct(t *testing.T) {
	a := []float64{1, 1, -1, -1}
	r := []float64{2, -2, -3, 3}
	// same, opposite, same, opposite → 50
	if p := SameDirectionPct(a, r); math.Abs(p-50) > 1e-9 {
		t.Fatalf("%v", p)
	}
}

func TestBestReturnLag_AssetLags(t *testing.T) {
	// ref moves first; asset is ref delayed by 2 bars.
	ref := []float64{0.1, 0.2, -0.1, 0.3, -0.2, 0.15, 0.05, -0.1, 0.2, 0.1, -0.05, 0.08, 0.12, -0.07, 0.04, 0.09, -0.03, 0.06}
	asset := make([]float64, len(ref))
	copy(asset[2:], ref)
	lag, _ := BestReturnLag(asset, ref, 4)
	if lag != -2 {
		t.Fatalf("lag %d", lag)
	}
	if ClassifyCorrTiming(lag, 0.2, 0.9) != CorrTimingLags {
		t.Fatal("timing")
	}
}

func TestClassifyCorrRelation(t *testing.T) {
	if ClassifyCorrRelation(0.81) != CorrRelationFollows {
		t.Fatal("follows")
	}
	if ClassifyCorrRelation(0.5) != CorrRelationLoose {
		t.Fatal("loose")
	}
	if ClassifyCorrRelation(0.1) != CorrRelationMixed {
		t.Fatal("mixed")
	}
	if ClassifyCorrRelation(-0.5) != CorrRelationInverse {
		t.Fatal("inverse")
	}
}

func TestCompareToReference_Self(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	pts := make([]PricePoint, 20)
	for i := range pts {
		pts[i] = PricePoint{Time: now.Add(time.Duration(i) * time.Minute), Close: 100 + float64(i)}
	}
	got := CompareToReference("BTCUSDT", "BTCUSDT", pts, pts, CorrelationWindows[0])
	if !got.Self || got.Corr != 1 || got.Relation != CorrRelationFollows {
		t.Fatalf("%+v", got)
	}
}

func TestBuildCorrelationWindow_FollowsBTC(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	n := 40
	asset := make([]PricePoint, n)
	btc := make([]PricePoint, n)
	eth := make([]PricePoint, n)
	for i := 0; i < n; i++ {
		t0 := now.Add(time.Duration(i) * time.Minute)
		btc[i] = PricePoint{Time: t0, Close: 100 + float64(i)*0.2}
		eth[i] = PricePoint{Time: t0, Close: 50 + float64(i)*0.05}
		asset[i] = PricePoint{Time: t0, Close: 10 + float64(i)*0.04}
	}
	got := BuildCorrelationWindow("SOLUSDT", CorrelationWindows[0], asset, btc, eth, "BTCUSDT", "ETHUSDT")
	if got.BTC.Relation != CorrRelationFollows || !got.BTC.Complete {
		t.Fatalf("btc %+v", got.BTC)
	}
	if got.ETH.Relation != CorrRelationFollows {
		t.Fatalf("eth %+v", got.ETH)
	}
	if !strings.Contains(strings.ToLower(got.Summary), "follows") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestCorrelationRefs(t *testing.T) {
	btc, eth := CorrelationRefs(ExchangeBinance, "SOLUSDT")
	if btc != "BTCUSDT" || eth != "ETHUSDT" {
		t.Fatalf("%s %s", btc, eth)
	}
	btc, eth = CorrelationRefs(ExchangeCoinbase, "SOL-USD")
	if btc != "BTC-USD" || eth != "ETH-USD" {
		t.Fatalf("%s %s", btc, eth)
	}
	btc, eth = CorrelationRefs(ExchangeBinance, "ETHBTC")
	if btc != "BTCUSDT" || eth != "ETHUSDT" {
		t.Fatalf("crypto quote %s %s", btc, eth)
	}
}

func TestExplainCorrelationReport_IncludesAllWindows(t *testing.T) {
	w := func(id, rel string, corr float64) CorrelationWindow {
		return CorrelationWindow{
			Window: id,
			BTC:    CorrelationVs{Complete: true, Relation: rel, Corr: corr},
			ETH:    CorrelationVs{Complete: true, Relation: CorrRelationLoose, Corr: 0.5},
		}
	}
	got := ExplainCorrelationReport("SOLUSDT", []CorrelationWindow{
		w(CorrWindow1h, CorrRelationLoose, 0.51),
		w(CorrWindow4h, CorrRelationLoose, 0.67),
		w(CorrWindow24h, CorrRelationFollows, 0.72),
	})
	if !strings.Contains(got, "24h") || !strings.Contains(got, "4h") || !strings.Contains(got, "1h") {
		t.Fatalf("%s", got)
	}
}

func TestAlignPricePoints(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0).UTC()
	a := []PricePoint{{Time: t0, Close: 1}, {Time: t0.Add(time.Minute), Close: 2}}
	b := []PricePoint{{Time: t0.Add(time.Minute), Close: 4}, {Time: t0.Add(2 * time.Minute), Close: 5}}
	ac, bc, ts := AlignPricePoints(a, b)
	if len(ac) != 1 || ac[0] != 2 || bc[0] != 4 || !ts[0].Equal(t0.Add(time.Minute)) {
		t.Fatalf("%v %v %v", ac, bc, ts)
	}
}
