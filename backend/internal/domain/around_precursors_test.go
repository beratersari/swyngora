package domain

import (
	"testing"
	"time"
)

func precursorHit(dir string, before AroundPhase) AroundMoveHit {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	before.Phase = AroundPhaseBefore
	before.Complete = true
	return AroundMoveHit{
		AroundMove: AroundMove{At: at, Direction: dir, ReturnPct: 3, Grade: AroundMoveGradeStrong},
		Around: &AroundReport{
			Symbol: "BTCUSDT", Combined: &AroundVenue{Exchange: "all", Phases: []AroundPhase{before}},
		},
	}
}

func TestSummarizeAroundPrecursors_FindsCommonVolumeAndTakers(t *testing.T) {
	var moves []AroundMoveHit
	for i := 0; i < 5; i++ {
		moves = append(moves, precursorHit(CVDDirUp, AroundPhase{
			Price: AroundPrice{Direction: CVDDirFlat, ChangePct: 0.1},
			Flow: AroundFlow{
				TypicalKnown: true, VolumeRatio: 2.2, VolumeGrade: VolumeSurgeElevated,
				BuySellKnown: true, Dominant: TakerSideBuy, BuyShare: 0.62,
			},
		}))
	}
	moves = append(moves, precursorHit(CVDDirDown, AroundPhase{
		Price: AroundPrice{Direction: CVDDirFlat, ChangePct: -0.05},
		Flow:  AroundFlow{TypicalKnown: true, VolumeRatio: 0.9, VolumeGrade: VolumeSurgeTypical},
	}))
	got := SummarizeAroundPrecursors(moves)
	if got.UpMoves != 5 || got.DownMoves != 1 || got.Sampled != 6 {
		t.Fatalf("counts %+v", got)
	}
	var vol, buy, quiet AroundPrecursorPattern
	for _, p := range got.Patterns {
		switch {
		case p.Metric == "volume_elevated" && p.Side == CVDDirUp:
			vol = p
		case p.Metric == "takers_buy" && p.Side == CVDDirUp:
			buy = p
		case p.Metric == "price_quiet" && p.Side == CVDDirUp:
			quiet = p
		}
	}
	if !vol.Common || vol.Hits != 5 || vol.Sample != 5 {
		t.Fatalf("volume %+v", vol)
	}
	if !buy.Common {
		t.Fatalf("takers %+v", buy)
	}
	if !quiet.Common {
		t.Fatalf("quiet %+v", quiet)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestSummarizeAroundPrecursors_NeedsSamples(t *testing.T) {
	got := SummarizeAroundPrecursors([]AroundMoveHit{
		{AroundMove: AroundMove{Direction: CVDDirUp}},
	})
	if got.Sampled != 0 || got.Summary == "" {
		t.Fatalf("%+v", got)
	}
}
