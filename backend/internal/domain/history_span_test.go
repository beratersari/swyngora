package domain

import (
	"testing"
	"time"
)

func TestBuildHistorySpan_DropsEmptyAndSetsSpan(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	since := now.Add(-90 * time.Minute)
	got := BuildHistorySpan("BTCUSDT", now, []RecordedSeries{
		{Kind: RecordedFuturesOI, Exchange: ExchangeBinance, Since: since},
		{Kind: RecordedBook, Exchange: ExchangeBybit},
		{Kind: "", Exchange: ExchangeOKX, Since: since},
	})
	if got.Symbol != "BTCUSDT" || len(got.Series) != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Series[0].SpanSec != 90*60 || got.Series[0].Exchange != ExchangeBinance {
		t.Fatalf("%+v", got.Series[0])
	}
	if got.Note == "" {
		t.Fatal("missing note")
	}
}

func TestAttachRecordedSpans_SetsSpanOnTheBoard(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	board := &DerivativesBoard{
		AsOf:  now,
		Coins: []DerivativesCoin{{Symbol: "BTCUSDT"}, {Symbol: "ETHUSDT"}},
	}
	AttachRecordedSpans(board, map[string][]RecordedSeries{
		"BTCUSDT": {{Kind: RecordedLiquidations, Exchange: ExchangeBybit, Since: now.Add(-48 * time.Hour)}},
	})
	if len(board.Coins[0].Recorded) != 1 || board.Coins[0].Recorded[0].SpanSec != 48*3600 {
		t.Fatalf("%+v", board.Coins[0].Recorded)
	}
	if len(board.Coins[1].Recorded) != 0 {
		t.Fatalf("eth %+v", board.Coins[1].Recorded)
	}
}
