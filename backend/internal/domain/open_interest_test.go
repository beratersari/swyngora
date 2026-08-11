package domain

import (
	"strings"
	"testing"
	"time"
)

func TestParseOpenInterestExchange(t *testing.T) {
	got, err := ParseOpenInterestExchange("")
	if err != nil || got != "all" {
		t.Fatalf("empty → all, got %s %v", got, err)
	}
	got, err = ParseOpenInterestExchange("BINANCE")
	if err != nil || got != "binance" {
		t.Fatalf("got %s %v", got, err)
	}
	if _, err := ParseOpenInterestExchange("coinbase"); err == nil {
		t.Fatal("coinbase must be rejected")
	}
}

func TestFindOpenInterestSample(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	hist := []OpenInterestPoint{
		{Time: now.Add(-2 * time.Hour), Contracts: 80, Value: 800},
		{Time: now.Add(-6 * time.Minute), Contracts: 90, Value: 900},
		{Time: now.Add(-4 * time.Minute), Contracts: 100, Value: 1000},
	}
	got, complete := FindOpenInterestSample(hist, now.Add(-5*time.Minute), 3*time.Minute)
	if !complete || got.Contracts != 90 {
		t.Fatalf("want 90 complete, got %+v complete=%v", got, complete)
	}
	got, complete = FindOpenInterestSample(hist, now.Add(-time.Hour), 10*time.Minute)
	if complete || got.Contracts != 80 {
		t.Fatalf("old sample incomplete: %+v complete=%v", got, complete)
	}
	if _, ok := FindOpenInterestSample(nil, now, time.Minute); ok {
		t.Fatal("empty hist")
	}
}

func TestSummarizeOpenInterestWindow_Up(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	s := &OpenInterestSeries{
		Exchange: ExchangeBinance,
		Symbol:   "BTCUSDT",
		Current:  OpenInterestPoint{Time: now, Contracts: 110, Value: 11000},
		History: []OpenInterestPoint{
			{Time: now.Add(-5 * time.Minute), Contracts: 100, Value: 10000},
		},
	}
	w := SummarizeOpenInterestWindow(s, OpenInterestWindow5m, 5*time.Minute, now)
	if !w.Complete || w.Direction != "up" || w.Change != "+10" || !strings.HasPrefix(w.ChangePct, "+") {
		t.Fatalf("%+v", w)
	}
	if w.ChangeValue != "+1000" {
		t.Fatalf("value change %s", w.ChangeValue)
	}
}

func TestBuildOpenInterestSnapshot_Combined(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	bin := &OpenInterestSeries{
		Exchange: ExchangeBinance,
		Symbol:   "BTCUSDT",
		Current:  OpenInterestPoint{Time: now, Contracts: 100, Value: 10_000},
		History: []OpenInterestPoint{
			{Time: now.Add(-5 * time.Minute), Contracts: 90, Value: 9000},
			{Time: now.Add(-time.Hour), Contracts: 80, Value: 8000},
			{Time: now.Add(-4 * time.Hour), Contracts: 70, Value: 7000},
			{Time: now.Add(-24 * time.Hour), Contracts: 50, Value: 5000},
		},
	}
	byb := &OpenInterestSeries{
		Exchange: ExchangeBybit,
		Symbol:   "BTCUSDT",
		Current:  OpenInterestPoint{Time: now, Contracts: 50, Value: 5000},
		History: []OpenInterestPoint{
			{Time: now.Add(-5 * time.Minute), Contracts: 40, Value: 4000},
			{Time: now.Add(-time.Hour), Contracts: 40, Value: 4000},
			{Time: now.Add(-4 * time.Hour), Contracts: 30, Value: 3000},
			{Time: now.Add(-24 * time.Hour), Contracts: 20, Value: 2000},
		},
	}
	snap := BuildOpenInterestSnapshot("all", "btc-usd", []*OpenInterestSeries{byb, bin}, now)
	if snap.Symbol != "BTCUSDT" || snap.Unit != "BTC" || snap.VenueCount != 2 {
		t.Fatalf("%+v", snap)
	}
	if snap.Current.Contracts != "150" || snap.Current.Value != "15000" {
		t.Fatalf("current %+v", snap.Current)
	}
	if len(snap.Venues) != 2 || snap.Venues[0].Exchange != "binance" {
		t.Fatalf("venues %+v", snap.Venues)
	}
	var w5 OpenInterestWindow
	for _, w := range snap.Windows {
		if w.Window == OpenInterestWindow5m {
			w5 = w
		}
	}
	if !w5.Complete || w5.Change != "+20" || w5.Direction != "up" {
		t.Fatalf("5m %+v", w5)
	}
}

func TestFormatSignedQtyAndPct(t *testing.T) {
	if FormatSignedQty(12.5) != "+12.5" {
		t.Fatalf("qty %s", FormatSignedQty(12.5))
	}
	if FormatSignedQty(-3) != "-3" {
		t.Fatalf("neg %s", FormatSignedQty(-3))
	}
	if FormatSignedPct(1.234) != "+1.23" {
		t.Fatalf("pct %s", FormatSignedPct(1.234))
	}
	if OpenInterestDirection(0.01) != "flat" || OpenInterestDirection(-1) != "down" {
		t.Fatal("direction")
	}
}

func TestSortOpenInterestHistory_Dedup(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0).UTC()
	got := SortOpenInterestHistory([]OpenInterestPoint{
		{Time: t0.Add(time.Minute), Contracts: 2, Value: 2},
		{Time: t0, Contracts: 1, Value: 1},
		{Time: t0, Contracts: 1.5, Value: 1.5},
	})
	if len(got) != 2 || got[0].Contracts != 1.5 || got[1].Contracts != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestValidateOpenInterestSymbol(t *testing.T) {
	s, err := ValidateOpenInterestSymbol(" eth-usd ")
	if err != nil || s != "ETHUSDT" {
		t.Fatalf("%s %v", s, err)
	}
	if _, err := ValidateOpenInterestSymbol("  "); err == nil {
		t.Fatal("empty")
	}
}
