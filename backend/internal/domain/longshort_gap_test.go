package domain

import (
	"testing"
	"time"
)

func TestAttachLongShortGap_UserExample(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := &LongShortSeries{
		Exchange: ExchangeBinance,
		Current:  LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.222},
		TopPositions: &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{Time: now, LongShare: 0.65, ShortShare: 0.35, Ratio: 1.857},
		},
	}
	AttachLongShortGap(s)
	if s.Gap == nil {
		t.Fatal("gap")
	}
	if s.Gap.Versus != LongShortKindTopPositions {
		t.Fatalf("versus %s", s.Gap.Versus)
	}
	if s.Gap.GapPct != "+10" || s.Gap.Lean != LongShortGapLeanTopLong {
		t.Fatalf("%+v", s.Gap)
	}
	if s.Gap.CrowdLongPct != "55" || s.Gap.TopLongPct != "65" {
		t.Fatalf("pcts %+v", s.Gap)
	}
	if s.Gap.Headline == "" || s.Gap.Detail == "" {
		t.Fatalf("copy %+v", s.Gap)
	}
}

func TestAttachLongShortGap_PrefersSizeOverAccounts(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := &LongShortSeries{
		Current: LongShortPoint{Time: now, LongShare: 0.50, ShortShare: 0.50, Ratio: 1},
		TopAccounts: &LongShortSeries{
			Kind:    LongShortKindTopAccounts,
			Current: LongShortPoint{Time: now, LongShare: 0.80, ShortShare: 0.20, Ratio: 4},
		},
		TopPositions: &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{Time: now, LongShare: 0.40, ShortShare: 0.60, Ratio: 0.667},
		},
	}
	AttachLongShortGap(s)
	if s.Gap == nil || s.Gap.Versus != LongShortKindTopPositions || s.Gap.Lean != LongShortGapLeanTopShort {
		t.Fatalf("%+v", s.Gap)
	}
	if s.Gap.GapPct != "-10" {
		t.Fatalf("gap %s", s.Gap.GapPct)
	}
}

func TestAttachLongShortGap_WindowsWidenAndFlip(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := &LongShortSeries{
		Current: LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.222},
		History: []LongShortPoint{
			{Time: now.Add(-time.Hour), LongShare: 0.54, ShortShare: 0.46, Ratio: 1.174},
			{Time: now.Add(-4 * time.Hour), LongShare: 0.50, ShortShare: 0.50, Ratio: 1},
		},
		TopPositions: &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{Time: now, LongShare: 0.65, ShortShare: 0.35, Ratio: 1.857},
			History: []LongShortPoint{
				{Time: now.Add(-time.Hour), LongShare: 0.58, ShortShare: 0.42, Ratio: 1.381},
				{Time: now.Add(-4 * time.Hour), LongShare: 0.40, ShortShare: 0.60, Ratio: 0.667},
			},
		},
	}
	AttachLongShortGap(s)
	if s.Gap == nil || s.Gap.GapPct != "+10" {
		t.Fatalf("live %+v", s.Gap)
	}
	if len(s.GapHistory) != 3 {
		t.Fatalf("hist %d", len(s.GapHistory))
	}
	if s.GapHistory[0].GapPct != "+10" {
		t.Fatalf("newest first %+v", s.GapHistory[0])
	}
	var h1, h4 LongShortGapWindow
	for _, w := range s.GapWindows {
		switch w.Window {
		case SnapshotWindow1h:
			h1 = w
		case SnapshotWindow4h:
			h4 = w
		}
	}
	if !h1.Complete || h1.Kind != LongShortGapWidened || h1.PastGapPct != "+4" || h1.ChangePct != "+6" {
		t.Fatalf("1h %+v", h1)
	}
	if !h4.Complete || h4.Kind != LongShortGapFlipped || h4.PastLean != LongShortGapLeanTopShort {
		t.Fatalf("4h %+v", h4)
	}
	flipped := false
	for _, m := range s.GapMoves {
		if m.Kind == LongShortGapFlipped {
			flipped = true
		}
	}
	if !flipped {
		t.Fatalf("moves %+v", s.GapMoves)
	}
}

func TestAttachLongShortGap_NoTopLayer(t *testing.T) {
	s := &LongShortSeries{
		Exchange: ExchangeBybit,
		Current:  LongShortPoint{Time: time.Now().UTC(), LongShare: 0.60, ShortShare: 0.40, Ratio: 1.5},
	}
	AttachLongShortGap(s)
	if s.Gap != nil {
		t.Fatalf("bybit gap %+v", s.Gap)
	}
}

func TestEnrichLongShortFromStore_FillsOlderTopPrints(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := &LongShortSeries{
		Exchange: ExchangeBinance,
		Current:  LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.222},
		TopPositions: &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{Time: now, LongShare: 0.65, ShortShare: 0.35, Ratio: 1.857},
		},
	}
	EnrichLongShortFromStore([]*LongShortSeries{s}, []FuturesSnapshot{{
		Exchange: ExchangeBinance, SampledAt: now.Add(-24 * time.Hour),
		LongShare: 0.52, ShortShare: 0.48, SizeLongShare: 0.52,
	}})
	if len(s.History) != 1 || len(s.TopPositions.History) != 1 {
		t.Fatalf("hist crowd=%d top=%d", len(s.History), len(s.TopPositions.History))
	}
	AttachLongShortGap(s)
	var d24 LongShortGapWindow
	for _, w := range s.GapWindows {
		if w.Window == SnapshotWindow24h {
			d24 = w
		}
	}
	if !d24.Complete || d24.Kind != LongShortGapWidened {
		t.Fatalf("24h %+v", d24)
	}
}

func TestLongShortGapKind(t *testing.T) {
	if longShortGapKind(4, 10, 1) != LongShortGapWidened {
		t.Fatal("widen")
	}
	if longShortGapKind(10, 4, 1) != LongShortGapNarrowed {
		t.Fatal("narrow")
	}
	if longShortGapKind(-8, 10, 1) != LongShortGapFlipped {
		t.Fatal("flip")
	}
	if longShortGapKind(4.2, 4.5, 1) != LongShortGapStable {
		t.Fatal("stable")
	}
}

func TestBuildLongShortSnapshot_AttachesGap(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	bin := &LongShortSeries{
		Exchange: ExchangeBinance,
		Current:  LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.222},
		TopPositions: &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{Time: now, LongShare: 0.65, ShortShare: 0.35, Ratio: 1.857},
		},
	}
	byb := &LongShortSeries{
		Exchange: ExchangeBybit,
		Current:  LongShortPoint{Time: now, LongShare: 0.58, ShortShare: 0.42, Ratio: 1.38},
	}
	got := BuildLongShortSnapshot("all", "BTCUSDT", []*LongShortSeries{byb, bin}, now)
	if got.Gap == nil || got.Gap.GapPct != "+10" {
		t.Fatalf("hoist %+v", got.Gap)
	}
	if got.Venues[0].Exchange != "binance" || got.Venues[0].Gap == nil {
		t.Fatalf("venues %+v", got.Venues)
	}
	if got.Venues[1].Gap != nil {
		t.Fatalf("bybit should have no gap %+v", got.Venues[1].Gap)
	}
}
