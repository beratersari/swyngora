package domain

import (
	"strings"
	"testing"
	"time"
)

func TestForensicsScanHitFromSimilar_KeepsMatches(t *testing.T) {
	rep := &AroundSimilarReport{
		Symbol: "BTCUSDT", Exchange: "binance", Events: 3,
		UpAfter: 2, DownAfter: 1, MedianAfterPct: 1.4,
		Matches: []AroundSimilarHit{{
			Similarity: 78, Coverage: 90, AfterDirection: CVDDirUp, AfterReturnPct: 3.2,
			Move: AroundMove{At: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)},
			Compared: []AroundSimilarFieldScore{
				{Name: AroundSimilarFieldOI, Used: true, Score: 91},
				{Name: AroundSimilarFieldBook, Used: true, Score: 84},
				{Name: AroundSimilarFieldPrice, Used: true, Score: 72},
				{Name: AroundSimilarFieldVolume, Used: true, Score: 41},
				{Name: AroundSimilarFieldTakers, Used: false},
			},
		}},
		AfterHorizons: []AroundSimilarHorizonStat{
			{Horizon: "1h", Up: 3, Down: 0, Sample: 3, Events: 3, MedianPct: 2.1},
		},
	}
	hit, ok := ForensicsScanHitFromSimilar(rep)
	if !ok {
		t.Fatal("expected hit")
	}
	if hit.Symbol != "BTCUSDT" || hit.Similarity != 78 || hit.Lean != CVDDirUp {
		t.Fatalf("%+v", hit)
	}
	if !strings.Contains(hit.AfterStory, "kept rising") {
		t.Fatalf("after %q", hit.AfterStory)
	}
	if !strings.Contains(hit.Summary, "BTC") {
		t.Fatalf("summary %q", hit.Summary)
	}
	if len(hit.Compared) != 5 || !strings.Contains(hit.WhySimilar, "open interest") || !strings.Contains(hit.WhySimilar, "order book") {
		t.Fatalf("why %+v %q", hit.Compared, hit.WhySimilar)
	}
	if !strings.Contains(hit.WhySimilar, "volume") || !strings.Contains(hit.WhySimilar, "taker flow") {
		t.Fatalf("parts %q", hit.WhySimilar)
	}
}

func TestExplainForensicsScanWhy_Parts(t *testing.T) {
	got := ExplainForensicsScanWhy(ForensicsScanHit{Compared: []AroundSimilarFieldScore{
		{Name: AroundSimilarFieldPrice, Used: true, Score: 80},
		{Name: AroundSimilarFieldVolume, Used: true, Score: 55},
		{Name: AroundSimilarFieldOI, Used: false},
	}})
	if !strings.Contains(got, "price") || !strings.Contains(got, "somewhat similar") || !strings.Contains(got, "open interest") {
		t.Fatalf("%q", got)
	}
	empty := ExplainForensicsScanWhy(ForensicsScanHit{})
	if empty != "" {
		t.Fatalf("empty %q", empty)
	}
}

func TestForensicsScanHitFromSimilar_Empty(t *testing.T) {
	if _, ok := ForensicsScanHitFromSimilar(&AroundSimilarReport{}); ok {
		t.Fatal("empty should skip")
	}
}

func TestExplainForensicsScan_RanksAndCounts(t *testing.T) {
	r := ForensicsScanReport{
		Compared: 4,
		Hits: []ForensicsScanHit{
			{Symbol: "ETHUSDT", Similarity: 70, Lean: CVDDirDown, AfterStory: "fell"},
			{Symbol: "BTCUSDT", Similarity: 88, Lean: CVDDirUp, AfterStory: "rose"},
		},
	}
	FinishForensicsScan(&r)
	if r.Hits[0].Symbol != "BTCUSDT" {
		t.Fatalf("rank %+v", r.Hits)
	}
	if !strings.Contains(r.Summary, "2 of 4") || !strings.Contains(r.Summary, "BTC") {
		t.Fatalf("summary %q", r.Summary)
	}
}

func TestForensicsScanLean(t *testing.T) {
	if ForensicsScanLean(3, 0) != CVDDirUp {
		t.Fatal("up")
	}
	if ForensicsScanLean(1, 3) != CVDDirDown {
		t.Fatal("down")
	}
	if ForensicsScanLean(2, 2) != "mixed" {
		t.Fatal("mixed")
	}
	if ForensicsScanLean(0, 0) != "unknown" {
		t.Fatal("unknown")
	}
}

func TestClampForensicsScanLimit(t *testing.T) {
	if ClampForensicsScanLimit(0) != DefaultForensicsScanLimit {
		t.Fatal("default")
	}
	if ClampForensicsScanLimit(99) != MaxForensicsScanLimit {
		t.Fatal("max")
	}
}
