package domain

import (
	"strconv"
	"testing"
	"time"
)

func TestMergeLiquidationSymbolTail_SkipsStoredAndKeepsNewer(t *testing.T) {
	storedAt := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	rows := []LiquidationSymbolSum{{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Long: 100, Count: 1,
	}}
	latest := map[Exchange]time.Time{ExchangeBinance: storedAt}
	events := []LiquidationEvent{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong, Notional: 50, Time: storedAt},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: LiquidationSideShort, Notional: 20, Time: storedAt.Add(time.Second)},
	}
	got := MergeLiquidationSymbolTail(rows, latest, events, storedAt.Add(-time.Hour), storedAt.Add(time.Minute))
	if len(got) != 2 {
		t.Fatalf("rows %+v", got)
	}
	if got[0].Long != 100 || got[0].Count != 1 {
		t.Fatalf("stored row doubled %+v", got[0])
	}
	if got[1].Symbol != "ETHUSDT" || got[1].Short != 20 || got[1].Count != 1 {
		t.Fatalf("tail %+v", got[1])
	}
}

func TestLiquidationCoinsFromSums_AddsVenues(t *testing.T) {
	rows := []LiquidationSymbolSum{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Long: 10, Count: 1},
		{Exchange: ExchangeBybit, Symbol: "ETHUSDT", Short: 5, Count: 1},
		{Exchange: ExchangeOKX, Symbol: "BTCUSDT", Long: 40, Count: 2},
	}
	got := LiquidationCoinsFromSums(rows, 10)
	if len(got) != 2 || got[0].Symbol != "BTCUSDT" || got[0].Count != 2 {
		t.Fatalf("%+v", got)
	}
	eth, _ := strconv.ParseFloat(got[1].TotalNotional, 64)
	if got[1].Symbol != "ETHUSDT" || eth != 15 {
		t.Fatalf("eth %+v", got[1])
	}
}

func TestLiquidationTimeBarsFromBuckets_PlacesStoreAndTail(t *testing.T) {
	origin := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	step := time.Hour
	scan := LiquidationBucketScan{
		Buckets: []LiquidationBucket{{Start: origin.Add(2 * time.Hour), Long: 80, Count: 1}},
		Latest:  map[Exchange]time.Time{ExchangeBinance: origin.Add(2 * time.Hour)},
	}
	events := []LiquidationEvent{{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: LiquidationSideShort,
		Notional: 20, Time: origin.Add(3*time.Hour + time.Minute),
	}}
	bars := LiquidationTimeBarsFromBuckets(scan, events, origin, step, 4)
	if bars[0].Count != 0 || bars[2].Count != 1 || bars[3].Count != 1 {
		t.Fatalf("%+v", bars)
	}
	longN, _ := strconv.ParseFloat(bars[2].LongNotional, 64)
	shortN, _ := strconv.ParseFloat(bars[3].ShortNotional, 64)
	if longN != 80 || shortN != 20 {
		t.Fatalf("long=%v short=%v", longN, shortN)
	}
}

func TestLiquidationBarGrid_Caps30Days(t *testing.T) {
	from := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	to := from.Add(30 * 24 * time.Hour)
	origin, until, step, n := LiquidationBarGrid(from, to, LiquidationTimeBarStep(to.Sub(from)))
	if n != maxLiquidationTimeBars || !origin.Equal(from) || until.After(to) || step <= time.Hour {
		t.Fatalf("origin %s until %s step %s n %d", origin, until, step, n)
	}
}
