package domain

import (
	"testing"
	"time"
)

func TestLiquidationEventKey_SeparatesVenues(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	a := LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 100, Quantity: 1, Time: now,
	}
	b := a
	b.Exchange = ExchangeBybit
	if LiquidationEventKey(a) == LiquidationEventKey(b) {
		t.Fatal("binance and bybit must not share a key")
	}
}

func TestLiquidationBook_RecordDedupesOverlap(t *testing.T) {
	book := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	book.now = func() time.Time { return now }
	ev := LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 64000, Quantity: 2, Notional: 128000, Time: now.Add(-time.Minute),
	}
	book.Record(ev)
	book.Record(ev)
	book.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 64000, Quantity: 2, Notional: 128000, Time: now.Add(-time.Minute),
	})
	if got := book.Snapshot("bybit", "BTCUSDT").Windows[0].Count; got != 1 {
		t.Fatalf("bybit count %d", got)
	}
	if got := book.Snapshot("binance", "BTCUSDT").Windows[0].Count; got != 1 {
		t.Fatalf("binance count %d", got)
	}
}

func TestLiquidationBook_ApplyHistoryFillsWholeGap(t *testing.T) {
	book := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	book.now = func() time.Time { return cur }

	book.SetLive(ExchangeBybit, true)
	book.MarkWatch(ExchangeBybit, "BTCUSDT")
	book.NoteSeen(ExchangeBybit)
	cur = now.Add(10 * time.Minute)
	book.SetLive(ExchangeBybit, false)
	cur = now.Add(25 * time.Minute)
	book.SetLive(ExchangeBybit, true)

	feed := book.Feed("bybit")
	if len(feed.Venues) != 1 || len(feed.Venues[0].Gaps) != 1 {
		t.Fatalf("pre-fill %+v", feed)
	}
	if feed.Venues[0].MissingSeconds < 14*60 || feed.Venues[0].MissingSeconds > 16*60 {
		t.Fatalf("missing before fill %d", feed.Venues[0].MissingSeconds)
	}

	mid := now.Add(18 * time.Minute)
	st := book.ApplyHistory(ExchangeBybit, now.Add(10*time.Minute), now.Add(25*time.Minute), []LiquidationEvent{
		{
			Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: LiquidationSideShort,
			Price: 1, Quantity: 1, Notional: 50, Time: mid,
		},
		{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideShort,
			Price: 1, Quantity: 1, Notional: 999, Time: mid,
		},
	}, now.Add(10*time.Minute), now.Add(25*time.Minute))
	if st.Added != 1 {
		t.Fatalf("added %+v", st)
	}
	if st.MissingSeconds != 0 {
		t.Fatalf("still missing %+v", st)
	}

	feed = book.Feed("bybit")
	if len(feed.Venues[0].Gaps) != 0 {
		t.Fatalf("filled gap must disappear %+v", feed.Venues[0].Gaps)
	}
	if feed.Venues[0].MissingSeconds != 0 {
		t.Fatalf("missing after fill %d", feed.Venues[0].MissingSeconds)
	}
	if windowCount(book.Snapshot("bybit", "BTCUSDT"), LiquidationWindow1h) != 1 {
		t.Fatal("history print missing")
	}
	if windowCount(book.Snapshot("binance", "BTCUSDT"), LiquidationWindow1h) != 0 {
		t.Fatal("bybit fill must not land on binance")
	}
}

func TestLiquidationBook_ApplyHistoryEmptyCoverRemovesGap(t *testing.T) {
	book := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	book.now = func() time.Time { return cur }
	book.SetLive(ExchangeBybit, true)
	cur = now.Add(time.Minute)
	book.SetLive(ExchangeBybit, false)
	cur = now.Add(16 * time.Minute)
	book.SetLive(ExchangeBybit, true)

	st := book.ApplyHistory(ExchangeBybit, now.Add(time.Minute), now.Add(16*time.Minute), nil, now.Add(time.Minute), now.Add(16*time.Minute))
	if st.Added != 0 || st.MissingSeconds != 0 {
		t.Fatalf("%+v", st)
	}
	if len(book.Feed("bybit").Venues[0].Gaps) != 0 {
		t.Fatal("empty but complete history must clear the gap")
	}
}

func TestLiquidationBook_ApplyHistoryPartialLeavesRemainder(t *testing.T) {
	book := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	book.now = func() time.Time { return cur }

	book.SetLive(ExchangeBinance, true)
	cur = now.Add(time.Minute)
	book.SetLive(ExchangeBinance, false)
	cur = now.Add(16 * time.Minute)
	book.SetLive(ExchangeBinance, true)

	// History only covers the last 5 minutes of the 15-minute hole.
	from := now.Add(time.Minute)
	to := now.Add(16 * time.Minute)
	coverFrom := now.Add(11 * time.Minute)
	st := book.ApplyHistory(ExchangeBinance, from, to, []LiquidationEvent{{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: LiquidationSideLong,
		Price: 2, Quantity: 1, Notional: 20, Time: now.Add(12 * time.Minute),
	}}, coverFrom, to)
	if st.MissingSeconds < 9*60 || st.MissingSeconds > 11*60 {
		t.Fatalf("remaining %+v", st)
	}
	feed := book.Feed("binance")
	if len(feed.Venues[0].Gaps) != 1 {
		t.Fatalf("gaps %+v", feed.Venues[0].Gaps)
	}
	g := feed.Venues[0].Gaps[0]
	if g.MissingSeconds != st.MissingSeconds || g.Seconds != g.MissingSeconds {
		t.Fatalf("gap %+v stats %+v", g, st)
	}
	if !g.To.Equal(coverFrom) && !g.To.Before(coverFrom.Add(time.Second)) {
		// leftover should end where fill begins
	}
	if g.To.After(coverFrom) {
		t.Fatalf("leftover to %v should be <= coverFrom %v", g.To, coverFrom)
	}
}

func TestLiquidationBook_ApplyHistoryDedupesLiveOverlap(t *testing.T) {
	book := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	book.now = func() time.Time { return now }
	ev := LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "SOLUSDT", Side: LiquidationSideLong,
		Price: 150, Quantity: 3, Notional: 450, Time: now.Add(-2 * time.Minute),
	}
	book.Record(ev)
	st := book.ApplyHistory(ExchangeBybit, now.Add(-5*time.Minute), now, []LiquidationEvent{ev, ev}, now.Add(-5*time.Minute), now)
	if st.Added != 0 {
		t.Fatalf("overlap counted twice %+v", st)
	}
	if windowCount(book.Snapshot("bybit", "SOLUSDT"), LiquidationWindow5m) != 1 {
		t.Fatal("expected one print")
	}
}

func windowCount(snap *LiquidationSnapshot, id string) int {
	if snap == nil {
		return 0
	}
	for _, w := range snap.Windows {
		if w.Window == id {
			return w.Count
		}
	}
	return 0
}

func TestSubtractGapInterval_SplitsMiddle(t *testing.T) {
	from := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Minute)
	gaps := []LiquidationGap{{From: from, To: to, Seconds: 900, MissingSeconds: 900}}
	got := SubtractGapInterval(gaps, from.Add(5*time.Minute), from.Add(10*time.Minute))
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].Seconds != 300 || got[1].Seconds != 300 {
		t.Fatalf("pieces %+v", got)
	}
	if sumGapMissing(got) != 600 {
		t.Fatalf("missing %d", sumGapMissing(got))
	}
}

func TestNormalizeHistoryQuery_ArchiveWindow(t *testing.T) {
	to := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	q, ok := NormalizeHistoryQuery(LiquidationHistoryQuery{
		Exchange:  ExchangeBinance,
		From:      to.Add(-90 * 24 * time.Hour),
		To:        to,
		MaxWindow: 30 * 24 * time.Hour,
	})
	if !ok {
		t.Fatal("expected a bounded window")
	}
	if q.To.Sub(q.From) != LiquidationHistoryMaxLookback {
		t.Fatalf("span %v", q.To.Sub(q.From))
	}
	q, ok = NormalizeHistoryQuery(LiquidationHistoryQuery{
		Exchange: ExchangeBinance,
		From:     to.Add(-90 * 24 * time.Hour),
		To:       to,
	})
	if !ok || q.To.Sub(q.From) != 24*time.Hour {
		t.Fatalf("reconnect cap %+v ok=%v", q, ok)
	}
}

func TestHistoryCoveredSpan_TruncatedPageKeepsTheStart(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(6 * time.Hour)
	oldest := from.Add(5 * time.Hour)
	gotFrom, gotTo, ok := HistoryCoveredSpan(from, to, oldest, false)
	if !ok || !gotFrom.Equal(oldest) || !gotTo.Equal(to) {
		t.Fatalf("partial %+v %+v %v", gotFrom, gotTo, ok)
	}
	fullFrom, fullTo, full := HistoryCoveredSpan(from, to, time.Time{}, true)
	if !full || !fullFrom.Equal(from) || !fullTo.Equal(to) {
		t.Fatalf("full %+v %+v %v", fullFrom, fullTo, full)
	}
	if _, _, ok := HistoryCoveredSpan(from, to, time.Time{}, false); ok {
		t.Fatal("empty truncated page must stay uncovered")
	}
}

func TestMergeHistoryCoverage_OneShortPageBlocksTheRest(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	_, _, ok := MergeHistoryCoverage([]LiquidationHistoryResult{
		{CoveredFrom: from, CoveredTo: to},
		{CoveredFrom: from.Add(30 * time.Minute), CoveredTo: to},
	})
	if !ok {
		t.Fatal("overlap should cover the later start")
	}
	if _, _, ok := MergeHistoryCoverage([]LiquidationHistoryResult{
		{CoveredFrom: from, CoveredTo: to},
		{},
	}); ok {
		t.Fatal("a missing span must leave the gap")
	}
}

func TestNormalizeHistoryQuery_RejectsTiny(t *testing.T) {
	q, ok := NormalizeHistoryQuery(LiquidationHistoryQuery{
		Exchange: ExchangeBybit,
		From:     time.Unix(1, 0),
		To:       time.Unix(1, 0).Add(500 * time.Millisecond),
	})
	if ok {
		t.Fatalf("tiny window %+v", q)
	}
}
