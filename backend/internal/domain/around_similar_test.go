package domain

import (
	"testing"
	"time"
)

func similarPhase(volRatio, oiPct, bidDelta float64, dir string) AroundPhase {
	return AroundPhase{
		Phase:    AroundPhaseBefore,
		Complete: true,
		From:     time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC),
		Price:    AroundPrice{Direction: CVDDirFlat, ChangePct: 0.1},
		Flow:     AroundFlow{TypicalKnown: true, VolumeRatio: volRatio, VolumeGrade: VolumeSurgeElevated},
		Book:     &AroundBook{BidNotionalDelta: bidDelta, AskNotionalDelta: 100, Complete: true},
		Futures:  &AroundFutures{OIDirection: dir, OIChangePct: oiPct, Complete: true},
	}
}

func similarHit(at time.Time, ret float64, before, during AroundPhase) AroundMoveHit {
	during.Phase = AroundPhaseDuring
	during.Complete = true
	during.Price = AroundPrice{ChangePct: ret, Direction: changeDir(ret)}
	return AroundMoveHit{
		AroundMove: AroundMove{At: at, Direction: changeDir(ret), ReturnPct: ret},
		Around: &AroundReport{
			Combined: &AroundVenue{Phases: []AroundPhase{before, during}},
		},
	}
}

func TestMatchAroundSimilar_RanksClosestAndReportsAfter(t *testing.T) {
	nowBefore := similarPhase(2.3, 4, -1500, CVDDirUp)
	nowBefore.From = time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC)
	closeAt := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	farAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	past := []AroundMoveHit{
		similarHit(closeAt, 3.5, similarPhase(2.2, 3.5, -1400, CVDDirUp), AroundPhase{}),
		similarHit(farAt, -2.8, similarPhase(0.8, -3, 2000, CVDDirDown), AroundPhase{}),
	}
	got := MatchAroundSimilar(nowBefore, past, 5, DefaultAroundSimilarFields())
	if len(got) < 1 {
		t.Fatal("expected a match")
	}
	if !got[0].Move.At.Equal(closeAt) {
		t.Fatalf("closest %+v", got[0])
	}
	if got[0].AfterReturnPct < 3 {
		t.Fatalf("after %+v", got[0])
	}
	if got[0].Similarity < got[len(got)-1].Similarity && len(got) > 1 {
		t.Fatalf("not ranked %+v", got)
	}
}

func TestMatchAroundSimilar_SkipsCurrentWindow(t *testing.T) {
	cur := similarPhase(2, 2, -100, CVDDirUp)
	cur.From = time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC)
	inside := similarHit(cur.From.Add(time.Minute), 2, similarPhase(2, 2, -100, CVDDirUp), AroundPhase{})
	got := MatchAroundSimilar(cur, []AroundMoveHit{inside}, 5, DefaultAroundSimilarFields())
	if len(got) != 0 {
		t.Fatalf("should skip current window %+v", got)
	}
}

func TestAroundPhaseSimilarity_ThinDataCannotScoreHigh(t *testing.T) {
	// Only price is present; volume/book/OI requested but missing.
	a := AroundPhase{Complete: true, Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat}}
	b := AroundPhase{Complete: true, Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat}}
	sc := aroundPhaseSimilarity(a, b, DefaultAroundSimilarFields())
	if sc.Similarity >= 40 {
		t.Fatalf("thin overlap scored too high %+v", sc)
	}
	if len(sc.Missing) < 2 || sc.Coverage > 30 {
		t.Fatalf("coverage %+v", sc)
	}
	foundPrice := false
	for _, c := range sc.Compared {
		if c.Name == AroundSimilarFieldPrice && c.Used {
			foundPrice = true
		}
	}
	if !foundPrice {
		t.Fatalf("price should be listed as used %+v", sc.Compared)
	}
}

func TestAroundPhaseSimilarity_SelectedFieldsOnly(t *testing.T) {
	a := similarPhase(2.3, 4, -1500, CVDDirUp)
	b := similarPhase(2.2, 3.8, -1400, CVDDirUp)
	// Opposite price path should not matter when price is off.
	a.Price.ChangePct = 2
	b.Price.ChangePct = -2
	want := AroundSimilarFields{Volume: true, Book: true, OI: true}
	sc := aroundPhaseSimilarity(a, b, want)
	if sc.Similarity < 70 {
		t.Fatalf("volume+book+oi should still match %+v", sc)
	}
	for _, n := range sc.Used {
		if n == AroundSimilarFieldPrice {
			t.Fatalf("price was excluded %+v", sc)
		}
	}
}

func TestParseAroundSimilarFields(t *testing.T) {
	got, err := ParseAroundSimilarFields("volume, orderbook, open_interest")
	if err != nil || !got.Volume || !got.Book || !got.OI || got.Price || got.Takers {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := ParseAroundSimilarFields("rsi"); err == nil {
		t.Fatal("expected error")
	}
}

func TestFinishAroundSimilar_CountsAfter(t *testing.T) {
	r := AroundSimilarReport{
		Symbol: "BTCUSDT",
		Matches: []AroundSimilarHit{
			{AfterReturnPct: 4, AfterDirection: CVDDirUp, Similarity: 80, Move: AroundMove{At: time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)}},
			{AfterReturnPct: -2, AfterDirection: CVDDirDown, Similarity: 70, Move: AroundMove{At: time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)}},
		},
	}
	FinishAroundSimilar(&r)
	if r.UpAfter != 1 || r.DownAfter != 1 || r.Summary == "" {
		t.Fatalf("%+v", r)
	}
}
