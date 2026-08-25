package domain

import (
	"strings"
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
	before.Phase = AroundPhaseBefore
	before.Complete = true
	before.From = at.Add(-time.Hour)
	before.To = at
	return AroundMoveHit{
		AroundMove: AroundMove{At: at, Until: at.Add(30 * time.Minute), Direction: changeDir(ret), ReturnPct: ret},
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
	got, _ := MatchAroundSimilar(nowBefore, past, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
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
	got, _ := MatchAroundSimilar(cur, []AroundMoveHit{inside}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
	if len(got) != 0 {
		t.Fatalf("should skip current window %+v", got)
	}
}

func TestAroundPhaseSimilarity_ThinDataCannotScoreHigh(t *testing.T) {
	// Only price is present; volume/book/OI requested but missing.
	a := AroundPhase{Complete: true, Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat}}
	b := AroundPhase{Complete: true, Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat}}
	sc := aroundPhaseSimilarity(a, b, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights())
	if len(sc.Missing) < 2 || sc.Coverage > 30 {
		t.Fatalf("coverage %+v", sc)
	}
	foundPrice := false
	for _, c := range sc.Compared {
		if c.Name == AroundSimilarFieldPrice && c.Used {
			foundPrice = true
			continue
		}
		if !c.Used && c.Score != 0 {
			t.Fatalf("uncompared %s should not have a score %+v", c.Name, c)
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
	w, _ := ParseAroundSimilarWeights("", want)
	sc := aroundPhaseSimilarity(a, b, want, w)
	if sc.Similarity < 70 {
		t.Fatalf("volume+book+oi should still match %+v", sc)
	}
	for _, n := range sc.Used {
		if n == AroundSimilarFieldPrice {
			t.Fatalf("price was excluded %+v", sc)
		}
	}
}

func TestMatchAroundSimilar_LowCoverageIsSkipped(t *testing.T) {
	cur := AroundPhase{
		Complete: true, From: time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC),
		Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat},
	}
	past := similarHit(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC), 3,
		AroundPhase{Phase: AroundPhaseBefore, Complete: true, Price: AroundPrice{ChangePct: 0.1, Direction: CVDDirFlat}}, AroundPhase{})
	keep, drop := MatchAroundSimilar(cur, []AroundMoveHit{past}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 60)
	if len(keep) != 0 {
		t.Fatalf("thin coverage should not be a normal match %+v", keep)
	}
	if len(drop) != 1 || drop[0].Coverage >= 60 || len(drop[0].Missing) == 0 {
		t.Fatalf("expected skipped with missing %+v", drop)
	}
}

func TestAroundPhaseSimilarity_BookAndOIWeighMoreThanVolume(t *testing.T) {
	cur := similarPhase(2.2, 4, -1500, CVDDirUp)
	volTwin := similarPhase(2.2, -4, 1500, CVDDirDown) // volume same, book+oi opposite
	bookTwin := similarPhase(0.6, 4, -1500, CVDDirUp)  // volume different, book+oi same
	fields := DefaultAroundSimilarFields()
	w := DefaultAroundSimilarWeights()
	volScore := aroundPhaseSimilarity(cur, volTwin, fields, w)
	bookScore := aroundPhaseSimilarity(cur, bookTwin, fields, w)
	if bookScore.Similarity <= volScore.Similarity {
		t.Fatalf("book+oi should outweigh volume: book=%v vol=%v", bookScore.Similarity, volScore.Similarity)
	}
}

func TestParseAroundSimilarWeightsAndCoverage(t *testing.T) {
	f, _ := ParseAroundSimilarFields("volume,book,oi")
	w, err := ParseAroundSimilarWeights("book:4,oi:4,volume:0.5", f)
	if err != nil || w.Book != 4 || w.OI != 4 || w.Volume != 0.5 || w.Price != 0 {
		t.Fatalf("%+v %v", w, err)
	}
	if _, err := ParseAroundSimilarWeights("price:2", f); err == nil {
		t.Fatal("expected error for weight on unselected field")
	}
	if _, err := ParseAroundSimilarWeights("book:0", f); err == nil {
		t.Fatal("expected error for non-positive weight")
	}
	c, err := ParseAroundSimilarMinCoverage("")
	if err != nil || c != 60 {
		t.Fatalf("default coverage %v %v", c, err)
	}
	zero, err := ParseAroundSimilarMinCoverage("0")
	if err != nil || zero != 0 {
		t.Fatalf("zero coverage %v %v", zero, err)
	}
	if _, err := ParseAroundSimilarMinCoverage("101"); err == nil {
		t.Fatal("expected error for coverage > 100")
	}
	hs, err := ParseAroundSimilarHorizons("6h,30m,2h,30m")
	if err != nil || len(hs) != 3 || hs[0].ID != "30m" || hs[1].ID != "2h" || hs[2].ID != "6h" {
		t.Fatalf("horizons %+v %v", hs, err)
	}
	if _, err := ParseAroundSimilarHorizons("rsi"); err == nil {
		t.Fatal("expected bad horizon")
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
	if r.UpAfter != 1 || r.DownAfter != 1 || r.Events != 2 || r.Summary == "" {
		t.Fatalf("%+v", r)
	}
}

func TestMatchAroundSimilar_CollapsesOverlapAndSameMove(t *testing.T) {
	cur := similarPhase(2.3, 4, -1500, CVDDirUp)
	cur.From = time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC)
	t0 := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	best := similarHit(t0, 3.5, similarPhase(2.2, 3.5, -1400, CVDDirUp), AroundPhase{})
	overlap := similarHit(t0.Add(15*time.Minute), 2.0, similarPhase(2.0, 3.0, -1200, CVDDirUp), AroundPhase{})
	// Opposite-direction close by should stay a separate event.
	opp := similarHit(t0.Add(50*time.Minute), -2.2, similarPhase(2.0, 3.0, -1200, CVDDirUp), AroundPhase{})
	// Far later, same direction — a different event.
	later := similarHit(t0.Add(4*time.Hour), 2.5, similarPhase(2.1, 3.2, -1300, CVDDirUp), AroundPhase{})

	got, _ := MatchAroundSimilar(cur, []AroundMoveHit{overlap, best, later}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
	if len(got) != 2 {
		t.Fatalf("want 2 distinct moves, got %d %+v", len(got), got)
	}
	if !got[0].Move.At.Equal(t0) {
		t.Fatalf("should keep highest similarity of the overlapping pair %+v", got[0])
	}
	if got[0].DataTo.After(got[0].Move.At) || got[0].DataFrom.IsZero() {
		t.Fatalf("data window must end at the move %+v", got[0])
	}

	// Setup windows overlap; the price moves do not — two events.
	sep := similarHit(t0.Add(45*time.Minute), 2.2, similarPhase(2.1, 3.2, -1300, CVDDirUp), AroundPhase{})
	sep.AroundMove.Until = t0.Add(60 * time.Minute)
	bestShort := similarHit(t0, 3.5, similarPhase(2.2, 3.5, -1400, CVDDirUp), AroundPhase{})
	bestShort.AroundMove.Until = t0.Add(15 * time.Minute)
	apart, _ := MatchAroundSimilar(cur, []AroundMoveHit{bestShort, sep}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
	if len(apart) != 2 {
		t.Fatalf("distinct moves must stay distinct even if setup windows overlap %+v", apart)
	}

	// Nearby opposite is a different price move.
	withOpp, _ := MatchAroundSimilar(cur, []AroundMoveHit{bestShort, opp}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
	if len(withOpp) != 2 {
		t.Fatalf("opposite move should stay separate %+v", withOpp)
	}

	// Overlapping move ranges (not just setup windows) collapse.
	a := similarHit(t0, 3.2, similarPhase(2.2, 3.5, -1400, CVDDirUp), AroundPhase{})
	b := similarHit(t0.Add(20*time.Minute), 2.4, similarPhase(2.1, 3.2, -1300, CVDDirUp), AroundPhase{})
	c := similarHit(t0.Add(25*time.Minute), 2.1, similarPhase(2.0, 3.0, -1200, CVDDirUp), AroundPhase{})
	chain, _ := MatchAroundSimilar(cur, []AroundMoveHit{c, a, b}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
	if len(chain) != 1 || !chain[0].Move.At.Equal(t0) {
		t.Fatalf("overlapping move ranges should collapse %+v", chain)
	}
}

func TestMatchAroundSimilar_ClipsPostMoveData(t *testing.T) {
	cur := similarPhase(2.3, 4, -1500, CVDDirUp)
	cur.From = time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC)
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	hit := similarHit(at, 3.5, similarPhase(2.2, 3.5, -1400, CVDDirUp), AroundPhase{})
	if p, ok := AroundPhaseByID(*hit.Around.Combined, AroundPhaseBefore); ok {
		p.To = at.Add(30 * time.Minute)
		p.Events = []AroundEvent{{Kind: AroundEventSweep, At: at.Add(10 * time.Minute)}}
		hit.Around.Combined.Phases[0] = p
	}
	got, _ := MatchAroundSimilar(cur, []AroundMoveHit{hit}, 5, DefaultAroundSimilarFields(), DefaultAroundSimilarWeights(), 0)
	if len(got) != 1 {
		t.Fatalf("expected a clipped match %+v", got)
	}
	if !got[0].DataTo.Equal(at) || got[0].DataFrom.After(at) || !got[0].DataFrom.Before(got[0].DataTo) {
		t.Fatalf("data window %+v–%+v", got[0].DataFrom, got[0].DataTo)
	}
	if got[0].Before.To.After(at) {
		t.Fatalf("before leaked past the move %+v", got[0].Before)
	}
	if len(got[0].Before.Events) != 0 {
		t.Fatalf("post-move event should be dropped %+v", got[0].Before.Events)
	}
}

func TestSummarizeAroundSimilarHorizons_SkipsShortTape(t *testing.T) {
	start := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	// 15m bars from 13:45 through 15:00 — enough for 15m and 1h, not 4h.
	var bars []AroundBar
	px := 100.0
	for i := -1; i < 5; i++ {
		open := px
		if i >= 0 {
			px = 100 + float64(i+1) // 101, 102, 103, 104, 105
		}
		bars = append(bars, AroundBar{
			Time: start.Add(time.Duration(i) * 15 * time.Minute),
			Open: open, Close: px,
		})
	}
	matches := []AroundSimilarHit{{Move: AroundMove{At: start, Open: 100}}}
	asOf := start.Add(2 * time.Hour)
	got := SummarizeAroundSimilarHorizons(matches, bars, asOf, nil)
	if len(got) != 3 {
		t.Fatalf("horizons %+v", got)
	}
	if got[0].Horizon != AroundSimilarHorizon15m || got[0].Sample != 1 || got[0].Up != 1 {
		t.Fatalf("15m %+v", got[0])
	}
	if got[0].AveragePct < 0.5 {
		t.Fatalf("15m avg %+v", got[0])
	}
	if got[1].Horizon != AroundSimilarHorizon1h || got[1].Sample != 1 {
		t.Fatalf("1h %+v", got[1])
	}
	if got[2].Horizon != AroundSimilarHorizon4h || got[2].Sample != 0 {
		t.Fatalf("4h should drop the short sample %+v", got[2])
	}

	// Too soon for 1h as well.
	early := SummarizeAroundSimilarHorizons(matches, bars, start.Add(20*time.Minute), nil)
	if early[0].Sample != 1 || early[1].Sample != 0 || early[2].Sample != 0 {
		t.Fatalf("as-of clip %+v", early)
	}

	startB := start.Add(2 * time.Hour)
	longBars := append([]AroundBar(nil), bars...)
	px = 105
	for i := 5; i < 20; i++ {
		open := px
		px = 100 + float64(i+1)
		longBars = append(longBars, AroundBar{
			Time: start.Add(time.Duration(i) * 15 * time.Minute),
			Open: open, Close: px,
		})
	}
	two := SummarizeAroundSimilarHorizons([]AroundSimilarHit{
		{Move: AroundMove{At: start, Open: 100}},
		{Move: AroundMove{At: startB, Open: 108}},
	}, longBars, start.Add(5*time.Hour), nil)
	if two[0].Sample != 2 || two[1].Sample != 2 || two[2].Sample != 1 {
		t.Fatalf("per-horizon sample %+v", two)
	}
	if two[0].Events != 2 || two[2].Events != 1 {
		t.Fatalf("per-horizon events %+v", two)
	}

	// Two overlapping starts count as one event for every horizon they can fill.
	overlap := SummarizeAroundSimilarHorizons([]AroundSimilarHit{
		{Move: AroundMove{At: start, Until: start.Add(30 * time.Minute), Open: 100, Direction: CVDDirUp}, Similarity: 90},
		{Move: AroundMove{At: start.Add(15 * time.Minute), Until: start.Add(45 * time.Minute), Open: 101, Direction: CVDDirUp}, Similarity: 70},
	}, longBars, start.Add(5*time.Hour), nil)
	if overlap[0].Events != 1 || overlap[0].Sample != 1 || overlap[1].Events != 1 {
		t.Fatalf("overlap should be one event %+v", overlap)
	}

	custom, err := ParseAroundSimilarHorizons("30m,2h")
	if err != nil || len(custom) != 2 {
		t.Fatalf("parse horizons %v %v", custom, err)
	}
	picked := SummarizeAroundSimilarHorizons(matches, longBars, start.Add(5*time.Hour), custom)
	if len(picked) != 2 || picked[0].Horizon != "30m" || picked[1].Horizon != "2h" {
		t.Fatalf("custom horizons %+v", picked)
	}
	if picked[0].Sample != 1 || picked[0].Up+picked[0].Down == 0 {
		t.Fatalf("30m stats %+v", picked[0])
	}
}

func TestFinishAroundSimilar_ExplainsHorizons(t *testing.T) {
	r := AroundSimilarReport{
		Symbol: "BTCUSDT",
		Matches: []AroundSimilarHit{
			{AfterReturnPct: 4, AfterDirection: CVDDirUp, Similarity: 80, Move: AroundMove{At: time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)}},
		},
		AfterHorizons: []AroundSimilarHorizonStat{
			{Horizon: AroundSimilarHorizon15m, Sample: 2, Up: 2, Down: 0, AveragePct: 1.2, MedianPct: 1.1},
			{Horizon: AroundSimilarHorizon1h, Sample: 2, Up: 1, Down: 1, AveragePct: 0.4, MedianPct: 0.3},
			{Horizon: AroundSimilarHorizon4h, Sample: 0},
		},
	}
	FinishAroundSimilar(&r)
	if !strings.Contains(r.Summary, "rose 2, fell 0") || !strings.Contains(r.Summary, "2 unique event") || !strings.Contains(r.Summary, "not enough data") {
		t.Fatalf("summary %q", r.Summary)
	}
}

func TestFinishAroundSimilar_SkippedOnlyMentionsCoverage(t *testing.T) {
	r := AroundSimilarReport{
		Symbol: "BTCUSDT", MinCoverage: 60,
		Skipped: []AroundSimilarHit{
			{Coverage: 20, Missing: []string{AroundSimilarFieldBook, AroundSimilarFieldOI}},
		},
	}
	FinishAroundSimilar(&r)
	if !strings.Contains(r.Summary, "lacked enough") {
		t.Fatalf("summary %q", r.Summary)
	}
}
