package domain

import (
	"testing"
	"time"
)

func TestBuildCandleFootprints_SplitsBuyAndSellInsideEachCandle(t *testing.T) {
	from := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)
	cells := []FlowAtPricesCell{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Minute: from.Add(10 * time.Minute), Price: 100, Buy: 9, Sell: 1, BuyN: 3, SellN: 1},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Minute: from.Add(40 * time.Minute), Price: 100, Buy: 1, Sell: 0, BuyN: 1},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Minute: from.Add(70 * time.Minute), Price: 110, Buy: 2, Sell: 8, BuyN: 1, SellN: 4},
	}
	got := BuildCandleFootprints("btcusdt", "binance", "1h", time.Hour, from, to, to, []Exchange{ExchangeBinance}, []CandleFootprintSource{{
		Exchange: ExchangeBinance, Cells: cells, Present: true,
	}}, false)
	if !got.Definite || got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("book %+v", got)
	}
	candles := got.Venues[0].Candles
	if len(candles) != 2 {
		t.Fatalf("candles %d", len(candles))
	}
	first := candles[0]
	if !first.Open.Equal(from) || first.Buy != 10 || first.Sell != 1 || first.BuyN != 4 || first.SellN != 1 {
		t.Fatalf("first %+v", first)
	}
	if len(first.Rows) != 1 || !first.Rows[0].IsPoc || first.Rows[0].Imbalance != TakerSideBuy || first.POC != 100 {
		t.Fatalf("first row %+v", first.Rows)
	}
	second := candles[1]
	if !second.Open.Equal(from.Add(time.Hour)) || second.Rows[0].Imbalance != TakerSideSell || second.Buy != 2 || second.Sell != 8 {
		t.Fatalf("second %+v", second)
	}
}

func TestBuildCandleFootprints_OneSidedDustIsNotAnImbalance(t *testing.T) {
	from := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	cells := []FlowAtPricesCell{
		{Minute: from, Price: 100, Buy: 100, Sell: 100},
		{Minute: from.Add(time.Minute), Price: 101, Buy: 0.2, Sell: 0},
		{Minute: from.Add(2 * time.Minute), Price: 102, Buy: 5, Sell: 0},
	}
	got := BuildCandleFootprints("BTCUSDT", "binance", "1h", time.Hour, from, from.Add(time.Hour), from, []Exchange{ExchangeBinance}, []CandleFootprintSource{{
		Exchange: ExchangeBinance, Cells: cells, Present: true,
	}}, false)
	rows := got.Venues[0].Candles[0].Rows
	byPrice := map[float64]string{}
	for _, row := range rows {
		byPrice[row.Price] = row.Imbalance
	}
	if byPrice[101] != "" || byPrice[102] != TakerSideBuy {
		t.Fatalf("%v", byPrice)
	}
}

func TestBuildCandleFootprints_RollsRowsAndKeepsTotals(t *testing.T) {
	from := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	cells := make([]FlowAtPricesCell, 0, 30)
	for i := 0; i < 30; i++ {
		cells = append(cells, FlowAtPricesCell{Minute: from, Price: 100 + float64(i), Buy: 1})
	}
	got := BuildCandleFootprints("BTCUSDT", "binance", "1h", time.Hour, from, from.Add(time.Hour), from, []Exchange{ExchangeBinance}, []CandleFootprintSource{{
		Exchange: ExchangeBinance, Cells: cells, Present: true,
	}}, false)
	candle := got.Venues[0].Candles[0]
	if len(candle.Rows) > CandleFootprintMaxRows {
		t.Fatalf("rows %d", len(candle.Rows))
	}
	if candle.Buy != 30 || candle.Sell != 0 {
		t.Fatalf("totals %+v", candle)
	}
	if !candle.Rows[0].IsPoc {
		t.Fatal("expected a poc")
	}
}

func TestBuildCandleFootprints_CombinedNeedsEveryVenue(t *testing.T) {
	from := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	expect := []Exchange{ExchangeBinance, ExchangeBybit, ExchangeOKX}
	sources := []CandleFootprintSource{
		{Exchange: ExchangeBinance, Present: true, Cells: []FlowAtPricesCell{{Minute: from, Price: 100, Buy: 4, Sell: 1}}},
		{Exchange: ExchangeBybit, Present: true, Cells: []FlowAtPricesCell{{Minute: from, Price: 100, Buy: 1, Sell: 3}}},
		{Exchange: ExchangeOKX, Present: false},
	}
	got := BuildCandleFootprints("BTCUSDT", "all", "1h", time.Hour, from, to, to, expect, sources, false)
	if got.Definite || got.Combined != nil || len(got.Missing) != 1 || got.Missing[0] != ExchangeOKX {
		t.Fatalf("%+v", got)
	}
	if len(got.Venues) != 2 {
		t.Fatalf("venues %d", len(got.Venues))
	}

	sources[2].Present = true
	got = BuildCandleFootprints("BTCUSDT", "all", "1h", time.Hour, from, to, to, expect, sources, false)
	if !got.Definite || got.Combined == nil || len(got.Combined.Candles) != 1 {
		t.Fatalf("combined %+v", got)
	}
	row := got.Combined.Candles[0]
	if row.Buy != 5 || row.Sell != 4 {
		t.Fatalf("sum %+v", row)
	}
}

func TestCandleFootprintOpen_AlignsHourAndWeek(t *testing.T) {
	at := time.Date(2026, 10, 7, 5, 40, 0, 0, time.UTC) // Wednesday
	if got := CandleFootprintOpen(at, 4*time.Hour); !got.Equal(time.Date(2026, 10, 7, 4, 0, 0, 0, time.UTC)) {
		t.Fatalf("4h %s", got)
	}
	week := CandleFootprintOpen(at, 7*24*time.Hour)
	if week.Weekday() != time.Monday || !week.Equal(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("week %s", week)
	}
}

func TestResolveCandleFootprintRange_KeepsNewestCandles(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC)
	start := now.Add(-1000 * time.Minute)
	from, to, truncated, err := ResolveCandleFootprintRange(time.Minute, &start, &now, now)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("expected truncated")
	}
	if to.Sub(from) != time.Duration(CandleFootprintMaxCandles)*time.Minute {
		t.Fatalf("span %s", to.Sub(from))
	}
	if _, _, err := ParseCandleFootprintInterval("nope"); err == nil {
		t.Fatal("expected bad interval")
	}
}
