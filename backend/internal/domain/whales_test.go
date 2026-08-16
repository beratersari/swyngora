package domain

import (
	"strings"
	"testing"
	"time"
)

func TestClusterWhalePrints_MergesNearby(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	prints := []TakerPrint{
		{Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: TakerSideBuy, Price: 100, Quantity: 600, Notional: 60_000, Time: t0},
		{Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: TakerSideBuy, Price: 101, Quantity: 600, Notional: 60_600, Time: t0.Add(time.Second)},
		{Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: TakerSideSell, Price: 99, Quantity: 2000, Notional: 198_000, Time: t0.Add(3 * time.Second)},
	}
	got := ClusterWhalePrints(prints, 50_000, 2*time.Second)
	if len(got) != 2 {
		t.Fatalf("events %d %+v", len(got), got)
	}
	SortWhalesBiggestFirst(got)
	if got[0].Side != TakerSideSell || got[0].Notional < 190_000 {
		t.Fatalf("biggest %+v", got[0])
	}
	buy := got[1]
	if buy.Prints != 2 || buy.AvgPrice < 100.4 || buy.AvgPrice > 100.6 {
		t.Fatalf("cluster %+v", buy)
	}
}

func TestAnnotateWhaleMcap_UnusualSmallCap(t *testing.T) {
	ev := WhaleEvent{Notional: 200_000}
	AnnotateWhaleMcap(&ev, 10_000_000) // 2% of mcap
	if !ev.Unusual || ev.NotionalMcapPct < 1 {
		t.Fatalf("%+v", ev)
	}
	small := WhaleEvent{Notional: 200_000}
	AnnotateWhaleMcap(&small, 5_000_000_000) // 0.004%
	if small.Unusual {
		t.Fatalf("should be normal %+v", small)
	}
}

func TestWhaleFromLiquidation(t *testing.T) {
	_, ok := WhaleFromLiquidation(LiquidationEvent{Notional: 10}, 1000)
	if ok {
		t.Fatal("too small")
	}
	got, ok := WhaleFromLiquidation(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 63000, Quantity: 2, Notional: 126_000, Time: time.Now().UTC(),
	}, 100_000)
	if !ok || got.Kind != WhaleKindLiquidation || got.Side != WhaleSideLong {
		t.Fatalf("%+v %v", got, ok)
	}
}

func TestParseWhaleBounds(t *testing.T) {
	if ParseWhaleMinNotional(0) != DefaultWhaleMinNotional {
		t.Fatal("default min")
	}
	if ParseWhaleMinNotional(1) != MinWhaleMinNotional {
		t.Fatal("clamp low")
	}
	if ParseWhaleLimit(0) != DefaultWhaleLimit || ParseWhaleLimit(999) != MaxWhaleLimit {
		t.Fatal("limit clamp")
	}
}

func TestExplainWhales(t *testing.T) {
	got := ExplainWhales([]WhaleEvent{{
		Symbol: "PEPEUSDT", Side: TakerSideBuy, Kind: WhaleKindTrade,
		Notional: 500_000, Quantity: 1e12, AvgPrice: 0.000001,
		Prints: 3, Unusual: true,
		FirstTime: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
		LastTime:  time.Date(2026, 8, 16, 12, 0, 1, 0, time.UTC),
	}}, "")
	if !strings.Contains(got, "Largest") || !strings.Contains(strings.ToLower(got), "market cap") {
		t.Fatalf("%s", got)
	}
}
