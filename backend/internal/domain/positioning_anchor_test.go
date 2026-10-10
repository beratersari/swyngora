package domain

import (
	"testing"
	"time"
)

func TestBuildPositioningAnchor_RebasesLevelsAndNamesALateVenue(t *testing.T) {
	anchor := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	before := anchor.Add(-time.Hour)
	after := anchor.Add(time.Hour)
	got := BuildPositioningAnchor(PositioningAnchorInput{
		Symbol: "btcusdt", Exchange: "all", Anchor: anchor, Bucket: 5 * time.Minute,
		OpenInterest: []FuturesSnapshot{
			{Exchange: ExchangeBinance, SampledAt: before, Value: 100},
			{Exchange: ExchangeBinance, SampledAt: after, Value: 140},
			{Exchange: ExchangeOKX, SampledAt: after, Value: 80},
		},
		Funding: []FuturesSnapshot{
			{Exchange: ExchangeBinance, SampledAt: before, FundingRate: 0.0001, Predicted: false},
			{Exchange: ExchangeBinance, SampledAt: after, FundingRate: 0.0008, Predicted: true},
			{Exchange: ExchangeBinance, SampledAt: after, FundingRate: 0.0003, Predicted: false},
			{Exchange: ExchangeBybit, SampledAt: after, FundingRate: -0.0002, Predicted: true},
		},
		Liquidations: []LiquidationEvent{
			{Exchange: ExchangeBinance, Time: before, Side: "short", Notional: 5},
			{Exchange: ExchangeBinance, Time: anchor, Side: "short", Notional: 4},
			{Exchange: ExchangeBinance, Time: after, Side: "long", Notional: 1},
			{Exchange: ExchangeOKX, Time: after, Side: "short", Notional: 9},
		},
	})
	if got.Symbol != "BTCUSDT" || got.Bucket != "5m" {
		t.Fatalf("symbol %s bucket %s", got.Symbol, got.Bucket)
	}
	if len(got.OpenInterest) != 1 || got.OpenInterest[0].Exchange != "binance" || got.OpenInterest[0].Change != "40" {
		t.Fatalf("oi %+v", got.OpenInterest)
	}
	if got.OpenInterest[0].Points[0].Time != anchor || got.OpenInterest[0].Points[0].Value != "0" {
		t.Fatalf("oi start %+v", got.OpenInterest[0].Points[0])
	}
	if len(got.MissingOpenInterest) != 3 || got.MissingOpenInterest[0] != "bybit" || got.MissingOpenInterest[1] != "okx" || got.MissingOpenInterest[2] != "hyperliquid" {
		t.Fatalf("missing oi %v", got.MissingOpenInterest)
	}
	if len(got.Funding) != 1 || got.Funding[0].Change != "0.02" {
		t.Fatalf("funding %+v", got.Funding)
	}
	if got.Funding[0].Points[1].Value != "0.02" {
		t.Fatalf("settled funding won: %+v", got.Funding[0].Points)
	}
	if len(got.MissingFunding) != 3 || got.MissingFunding[0] != "bybit" || got.MissingFunding[2] != "hyperliquid" {
		t.Fatalf("missing funding %v", got.MissingFunding)
	}
	if len(got.Liquidations) != 2 {
		t.Fatalf("liq %+v", got.Liquidations)
	}
	binance := got.Liquidations[0]
	if !binance.Complete || binance.Change != "3" || binance.Points[0].Value != "4" {
		t.Fatalf("binance liq %+v", binance)
	}
	okx := got.Liquidations[1]
	if okx.Complete || okx.Exchange != "okx" || okx.Points[0].Time != after || okx.Change != "9" {
		t.Fatalf("okx liq %+v", okx)
	}
	if len(got.LateLiquidations) != 1 || got.LateLiquidations[0] != "okx" {
		t.Fatalf("late %v", got.LateLiquidations)
	}
	if len(got.MissingLiquidations) != 1 || got.MissingLiquidations[0] != "bybit" {
		t.Fatalf("missing liq %v", got.MissingLiquidations)
	}
	for _, name := range got.MissingLiquidations {
		if name == "hyperliquid" {
			t.Fatal("hyperliquid has no liquidation feed and is not a missing print")
		}
	}
}

func TestBuildPositioningAnchor_QuietVenueStartsAtZero(t *testing.T) {
	anchor := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	got := BuildPositioningAnchor(PositioningAnchorInput{
		Exchange: "binance", Anchor: anchor, Bucket: time.Hour,
		Liquidations: []LiquidationEvent{
			{Exchange: ExchangeBinance, Time: anchor.Add(-time.Minute), Side: "long", Notional: 2},
		},
	})
	if len(got.Liquidations) != 1 || !got.Liquidations[0].Complete || got.Liquidations[0].Change != "0" {
		t.Fatalf("%+v", got.Liquidations)
	}
	if got.Liquidations[0].Points[0].Time != anchor {
		t.Fatalf("start %+v", got.Liquidations[0].Points)
	}
	if len(got.MissingOpenInterest) != 1 || got.MissingOpenInterest[0] != "binance" {
		t.Fatalf("single venue %v", got.MissingOpenInterest)
	}
}

func TestValidatePositioningAnchor(t *testing.T) {
	if err := ValidatePositioningAnchor(time.Time{}); err == nil {
		t.Fatal("empty anchor")
	}
	if err := ValidatePositioningAnchor(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}
