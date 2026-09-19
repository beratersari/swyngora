package domain

import (
	"strings"
	"testing"
	"time"
)

func TestAttachBasisCurveHistory_FlipAndSteeper(t *testing.T) {
	now := time.Date(2026, 9, 19, 16, 0, 0, 0, time.UTC)
	nearSym, farSym := "BTCUSDT_260925", "BTCUSDT_261225"
	nearHist := []PriceSample{}
	farHist := []PriceSample{}
	// 7h ago: backwardation (far cheaper). Then flip to contango and steepen.
	for i, pair := range [][2]float64{
		{64000, 63600}, // -0.625% backwardation
		{64000, 63620},
		{64000, 63900},
		{64000, 64100}, // flip to contango
		{64000, 64300},
		{64000, 64600}, // steeper
		{64000, 64700},
	} {
		at := now.Add(time.Duration(i-6) * time.Hour)
		nearHist = append(nearHist, PriceSample{Time: at, Price: pair[0]})
		farHist = append(farHist, PriceSample{Time: at, Price: pair[1]})
	}
	q := BasisQuote{
		Symbol: "BTCUSDT", Time: now, FuturesMark: 64000, SpotIndex: 64000,
		Contracts: []BasisContractQuote{
			{Symbol: "BTCUSDT", Kind: BasisContractPerp, Mark: 64000, Index: 64000},
			{Symbol: nearSym, Kind: BasisContractCurrentQuarter, Expiry: now.Add(40 * 24 * time.Hour), Mark: 64000, Index: 64000, History: nearHist},
			{Symbol: farSym, Kind: BasisContractNextQuarter, Expiry: now.Add(130 * 24 * time.Hour), Mark: 64700, Index: 64000, History: farHist},
		},
	}
	got := AttachBasisCurveHistory(BuildBasisCurve(q), q)
	if !got.Complete || got.Shape != BasisCurveContango {
		t.Fatalf("live %+v", got)
	}
	if len(got.History) < 6 {
		t.Fatalf("history %d", len(got.History))
	}
	var h1, h4 *BasisCurveWindow
	for i := range got.Windows {
		switch got.Windows[i].Window {
		case SnapshotWindow1h:
			h1 = &got.Windows[i]
		case SnapshotWindow4h:
			h4 = &got.Windows[i]
		}
	}
	if h1 == nil || !h1.Complete || h1.Change != BasisCurveChangeSteeper {
		t.Fatalf("1h %+v", got.Windows)
	}
	if h4 == nil || !h4.Complete || h4.Change != BasisCurveChangeFlipped {
		t.Fatalf("4h %+v", got.Windows)
	}
	flipped := false
	steepened := false
	for _, m := range got.Moves {
		if m.Kind == BasisCurveMoveFlipped && m.FromShape == BasisCurveBackwardation && m.ToShape == BasisCurveContango {
			flipped = true
		}
		if m.Kind == BasisCurveMoveSteepened {
			steepened = true
		}
	}
	if !flipped || !steepened {
		t.Fatalf("moves %+v", got.Moves)
	}
	if !strings.Contains(got.Summary, "steeper") && !strings.Contains(got.Summary, "flipped") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestAttachBasisCurveHistory_NoDatedHistory(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	q := BasisQuote{
		Symbol: "BTCUSDT", Time: now, FuturesMark: 64000, SpotIndex: 64000,
		Contracts: []BasisContractQuote{
			{Symbol: "BTCUSDT_260925", Kind: BasisContractCurrentQuarter, Expiry: now.Add(40 * 24 * time.Hour), Mark: 64200},
			{Symbol: "BTCUSDT_261225", Kind: BasisContractNextQuarter, Expiry: now.Add(130 * 24 * time.Hour), Mark: 64600},
		},
	}
	got := AttachBasisCurveHistory(BuildBasisCurve(q), q)
	if !got.Complete || len(got.History) != 1 {
		t.Fatalf("live-only history %+v", got.History)
	}
}

func TestPairCurveMarks_AlignsWithinSlack(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	near := []PriceSample{{Time: t0.Add(time.Minute), Price: 100}}
	far := []PriceSample{{Time: t0, Price: 101}}
	got := pairCurveMarks(near, far)
	if len(got) != 1 || got[0].Shape != BasisCurveContango {
		t.Fatalf("%+v", got)
	}
}
