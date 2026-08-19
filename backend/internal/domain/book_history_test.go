package domain

import (
	"strings"
	"testing"
	"time"
)

func sampleRawBookAt(mid float64, bidQty, askQty float64) RawOrderBook {
	return RawOrderBook{
		Symbol: "BTCUSDT",
		Bids: []PriceLevel{
			{Price: mid - 1, Quantity: bidQty},
			{Price: mid - 2, Quantity: bidQty / 2},
			{Price: mid - 10, Quantity: bidQty * 8},
		},
		Asks: []PriceLevel{
			{Price: mid + 1, Quantity: askQty},
			{Price: mid + 2, Quantity: askQty / 2},
			{Price: mid + 10, Quantity: askQty * 3},
		},
		Live:      true,
		FetchedAt: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
	}
}

func TestCaptureBookHistory_HasLevelsAndSpread(t *testing.T) {
	raw := sampleRawBookAt(100, 5, 4)
	got := CaptureBookHistory(ExchangeBinance, raw, time.Date(2026, 8, 16, 12, 0, 22, 0, time.UTC))
	if got.Symbol != "BTCUSDT" || got.Exchange != ExchangeBinance {
		t.Fatalf("%+v", got)
	}
	if got.SampledAt != time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) {
		t.Fatalf("bucket %s", got.SampledAt)
	}
	if got.Mid <= 0 || got.Spread <= 0 || got.BidNotional <= 0 || got.AskNotional <= 0 {
		t.Fatalf("totals %+v", got)
	}
	if len(got.Bids) == 0 || len(got.Asks) == 0 {
		t.Fatalf("levels bids=%d asks=%d", len(got.Bids), len(got.Asks))
	}
}

func TestCompareBookHistory_GainedAndLost(t *testing.T) {
	from := CaptureBookHistory(ExchangeBinance, sampleRawBookAt(100, 5, 4), time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC))
	toRaw := sampleRawBookAt(102, 2, 9)
	to := CaptureBookHistory(ExchangeBinance, toRaw, time.Date(2026, 8, 16, 12, 5, 0, 0, time.UTC))
	diff := CompareBookHistory(from, to)
	if diff.MidDelta <= 0 {
		t.Fatalf("mid should rise %+v", diff)
	}
	if len(diff.Gained)+len(diff.Lost) == 0 {
		t.Fatalf("expected level changes %+v", diff)
	}
	if !strings.Contains(diff.Summary, "from") || !strings.Contains(strings.ToLower(diff.Summary), "liquidity") {
		t.Fatalf("summary %s", diff.Summary)
	}
}

func TestNearestBookSnapshot_PrefersBefore(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	rows := []BookHistorySnapshot{
		{SampledAt: t0.Add(-2 * time.Minute), Mid: 99},
		{SampledAt: t0.Add(4 * time.Minute), Mid: 101},
	}
	got, ok := NearestBookSnapshot(rows, t0, 3*time.Minute)
	if !ok || got.Mid != 99 {
		t.Fatalf("%v %+v", ok, got)
	}
}

func TestExplainBookHistory_Empty(t *testing.T) {
	if !strings.Contains(ExplainBookHistory(BookHistorySnapshot{}), "No stored") {
		t.Fatal("empty")
	}
}
