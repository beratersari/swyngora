package domain

import (
	"strings"
	"testing"
	"time"
)

func TestTakerDominant(t *testing.T) {
	if TakerDominant(60, 40) != TakerSideBuy {
		t.Fatal("buy")
	}
	if TakerDominant(40, 60) != TakerSideSell {
		t.Fatal("sell")
	}
	if TakerDominant(50, 50) != TakerSideEven {
		t.Fatal("even")
	}
}

func TestBuildTakerVenueFlow_Windows(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	buckets := []TakerBucket{
		{Start: now.Add(-2 * time.Minute), BuyNotional: 100, SellNotional: 40},
		{Start: now.Add(-20 * time.Minute), BuyNotional: 10, SellNotional: 80},
		{Start: now.Add(-2 * time.Hour), BuyNotional: 5, SellNotional: 5},
	}
	got := BuildTakerVenueFlow(ExchangeBinance, "BTCUSDT", buckets, now, now.Add(-5*time.Hour))
	if len(got.Windows) != 3 {
		t.Fatalf("%+v", got.Windows)
	}
	if got.Windows[0].Window != TakerWindow5m || got.Windows[0].BuyNotional != 100 || !got.Windows[0].Complete {
		t.Fatalf("5m %+v", got.Windows[0])
	}
	if got.Windows[1].SellNotional != 120 { // 40+80
		t.Fatalf("1h %+v", got.Windows[1])
	}
	if got.Windows[1].NeedSec != 3600 || got.Windows[1].HaveSec != 3600 || !got.Windows[1].Complete {
		t.Fatalf("1h span %+v", got.Windows[1])
	}
	short := BuildTakerVenueFlow(ExchangeBybit, "BTCUSDT", buckets, now, now.Add(-20*time.Minute))
	if short.Windows[1].Complete || short.Windows[1].HaveSec < 1100 || short.Windows[1].HaveSec > 1300 {
		t.Fatalf("short 1h collection must not look complete: %+v", short.Windows[1])
	}
	if got.Windows[0].Dominant != TakerSideBuy {
		t.Fatalf("dom %s", got.Windows[0].Dominant)
	}
}

func TestCombineTakerVenues(t *testing.T) {
	a := TakerVenueFlow{Exchange: ExchangeBinance, Windows: []TakerWindowFlow{
		SummarizeTakerWindow(100, 40, TakerWindow5m, true),
	}}
	b := TakerVenueFlow{Exchange: ExchangeBybit, Windows: []TakerWindowFlow{
		SummarizeTakerWindow(20, 80, TakerWindow5m, true),
	}}
	got := CombineTakerVenues("BTCUSDT", []TakerVenueFlow{a, b})
	if got.Windows[0].BuyNotional != 120 || got.Windows[0].SellNotional != 120 {
		t.Fatalf("%+v", got.Windows[0])
	}
	if got.Dominant != TakerSideEven {
		t.Fatalf("dom %s", got.Dominant)
	}
}

func TestExplainTakerFlow_LongBuildup(t *testing.T) {
	v := TakerVenueFlow{
		Exchange: ExchangeBinance,
		Windows: []TakerWindowFlow{
			SummarizeTakerWindow(800, 200, TakerWindow5m, true),
		},
	}
	s := ExplainTakerFlow(v, TakerFlowContext{
		PriceChange1hPct: 1.2, OIChange1hPct: 2, FundingRate: 0.0001,
		Positioning: RegimeLongBuildup,
	})
	if !strings.Contains(strings.ToLower(s), "buy") || !strings.Contains(s, "long buildup") {
		t.Fatalf("%s", s)
	}
}

func TestTakerBook_RecordAndSnapshot(t *testing.T) {
	b := NewTakerBook()
	now := time.Date(2026, 8, 14, 16, 10, 30, 0, time.UTC)
	b.now = func() time.Time { return now }
	b.Record(TakerPrint{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideBuy, Notional: 50, Time: now})
	b.Record(TakerPrint{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideSell, Notional: 10, Time: now.Add(-30 * time.Second)})
	got := b.Snapshot(ExchangeBybit, "btcusdt")
	if got.Windows[0].BuyNotional != 50 || got.Windows[0].SellNotional != 10 {
		t.Fatalf("%+v", got.Windows[0])
	}
}
