package domain

import (
	"testing"
	"time"
)

func TestLongShortRatioAndBias(t *testing.T) {
	if got := LongShortRatioFromShares(0.63, 0.37); got < 1.70 || got > 1.71 {
		t.Fatalf("ratio %v", got)
	}
	if LongShortBias(1.72) != "long" || LongShortBias(0.8) != "short" || LongShortBias(1.01) != "balanced" {
		t.Fatal("bias")
	}
}

func TestBuildLongShortSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	bin := &LongShortSeries{
		Exchange: ExchangeBinance,
		Current:  LongShortPoint{Time: now, LongShare: 0.63, ShortShare: 0.37, Ratio: 1.7027},
		History: []LongShortPoint{
			{Time: now.Add(-5 * time.Minute), LongShare: 0.60, ShortShare: 0.40, Ratio: 1.5},
		},
	}
	byb := &LongShortSeries{
		Exchange: ExchangeBybit,
		Current:  LongShortPoint{Time: now, LongShare: 0.58, ShortShare: 0.42, Ratio: 1.3810},
	}
	one := BuildLongShortSnapshot("binance", "btc-usd", []*LongShortSeries{bin}, now)
	if one.Symbol != "BTCUSDT" || one.Current == nil || one.Current.Bias != "long" {
		t.Fatalf("one %+v", one)
	}
	if len(one.History) != 1 || one.Venues[0].Change != "+0.2027" {
		t.Fatalf("hist %+v", one.Venues[0])
	}
	all := BuildLongShortSnapshot("all", "BTCUSDT", []*LongShortSeries{byb, bin}, now)
	if all.Current != nil || all.VenueCount != 2 || all.Venues[0].Exchange != "binance" {
		t.Fatalf("all %+v", all)
	}
}

func TestClampLongShortHistoryLimit(t *testing.T) {
	if ClampLongShortHistoryLimit(0) != DefaultLongShortHistoryLimit {
		t.Fatal("default")
	}
	if ClampLongShortHistoryLimit(999) != MaxLongShortHistoryLimit {
		t.Fatal("max")
	}
}
