package domain

import (
	"math"
	"testing"
	"time"
)

func TestSqueezeLevelFromScore(t *testing.T) {
	if SqueezeLevelFromScore(90) != SqueezeLevelExtreme {
		t.Fatal("extreme")
	}
	if SqueezeLevelFromScore(20) != SqueezeLevelLow {
		t.Fatal("low")
	}
}

func TestBuildSqueezeVenue_CrowdedLongsHighLongSqueeze(t *testing.T) {
	got := BuildSqueezeVenue(SqueezeInputs{
		Exchange:          ExchangeBinance,
		Symbol:            "BTCUSDT",
		Price:             64000,
		OIValue:           5_000_000_000,
		OIChange1hPct:     3.5,
		OIChange4hPct:     6,
		PriceChange24hPct: 2,
		LongShare:         0.72,
		ShortShare:        0.28,
		FundingRate:       0.0004,
		FundingAvg3:       0.0003,
		HasFundingAvg:     true,
		LongLiq1h:         2_000_000,
		LongLiq24h:        10_000_000,
		LongPressureNear:  0.25,
		ShortPressureNear: 0.05,
	})
	if got.CrowdedSide != SqueezeSideLong {
		t.Fatalf("crowded %s", got.CrowdedSide)
	}
	if got.LongSqueeze.Score <= got.ShortSqueeze.Score {
		t.Fatalf("long should dominate: L=%.1f S=%.1f", got.LongSqueeze.Score, got.ShortSqueeze.Score)
	}
	if got.HigherRisk != SqueezeSideLong {
		t.Fatalf("higher %s", got.HigherRisk)
	}
	if got.LongSqueeze.Level == SqueezeLevelLow {
		t.Fatalf("expected elevated+ long squeeze, got %s (%.1f)", got.LongSqueeze.Level, got.LongSqueeze.Score)
	}
	if len(got.LongSqueeze.Reasons) == 0 || len(got.LongSqueeze.Factors) < 4 {
		t.Fatalf("%+v", got.LongSqueeze)
	}
}

func TestBuildSqueezeVenue_CrowdedShortsHighShortSqueeze(t *testing.T) {
	got := BuildSqueezeVenue(SqueezeInputs{
		Exchange:          ExchangeBybit,
		Symbol:            "ETHUSDT",
		Price:             3000,
		OIValue:           1_000_000_000,
		OIChange1hPct:     2,
		PriceChange24hPct: -3,
		LongShare:         0.32,
		ShortShare:        0.68,
		FundingRate:       -0.00025,
		ShortLiq1h:        1_500_000,
		ShortPressureNear: 0.3,
	})
	if got.HigherRisk != SqueezeSideShort {
		t.Fatalf("higher %s L=%.1f S=%.1f", got.HigherRisk, got.LongSqueeze.Score, got.ShortSqueeze.Score)
	}
	if got.ShortSqueeze.Score < 50 {
		t.Fatalf("short score %.1f", got.ShortSqueeze.Score)
	}
}

func TestBuildSqueezeVenue_BalancedLow(t *testing.T) {
	got := BuildSqueezeVenue(SqueezeInputs{
		Exchange:      ExchangeBinance,
		Symbol:        "BTCUSDT",
		OIValue:       1e9,
		LongShare:     0.51,
		ShortShare:    0.49,
		FundingRate:   0.00001,
		OIChange1hPct: -0.5,
	})
	if got.LongSqueeze.Score > 60 && got.ShortSqueeze.Score > 60 {
		t.Fatalf("both high: %+v", got)
	}
}

func TestCombineSqueezeReports_OIWeighted(t *testing.T) {
	bin := BuildSqueezeVenue(SqueezeInputs{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", OIValue: 9e9,
		LongShare: 0.7, ShortShare: 0.3, FundingRate: 0.0003, OIChange1hPct: 2,
	})
	byb := BuildSqueezeVenue(SqueezeInputs{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT", OIValue: 1e9,
		LongShare: 0.35, ShortShare: 0.65, FundingRate: -0.0002, OIChange1hPct: 1,
	})
	c := CombineSqueezeReports([]SqueezeVenueReport{bin, byb})
	if c == nil || c.DominantVenue != "binance" {
		t.Fatalf("%+v", c)
	}
	// Binance dominates OI → combined should lean long squeeze
	if c.LongSqueeze.Score <= c.ShortSqueeze.Score {
		t.Fatalf("expected long-weighted: L=%.1f S=%.1f", c.LongSqueeze.Score, c.ShortSqueeze.Score)
	}
}

func TestNearLiquidationPressureShare(t *testing.T) {
	// 2% covers 100x (~0.6%) and 50x (~1.6%) and 75x etc.
	s := NearLiquidationPressureShare(true, 2, 0.0002)
	if s < 0.1 || s > 0.8 {
		t.Fatalf("share %v", s)
	}
	if NearLiquidationPressureShare(true, 0.1, 0) > 0.05 {
		t.Fatal("tiny window should have almost no pocket")
	}
}

func TestOIChangePctFromSeries(t *testing.T) {
	now := time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC)
	ser := &OpenInterestSeries{
		Current: OpenInterestPoint{Time: now, Contracts: 110, Value: 1100},
		History: []OpenInterestPoint{
			{Time: now.Add(-time.Hour), Contracts: 100, Value: 1000},
		},
	}
	p := OIChangePctFromSeries(ser, time.Hour, now)
	if math.IsNaN(p) || math.Abs(p-10) > 0.01 {
		t.Fatalf("pct %v", p)
	}
}

func TestSumLiquidationNotional(t *testing.T) {
	now := time.Now().UTC()
	ev := []LiquidationEvent{
		{Side: LiquidationSideLong, Notional: 10, Time: now.Add(-30 * time.Minute)},
		{Side: LiquidationSideShort, Notional: 5, Time: now.Add(-30 * time.Minute)},
		{Side: LiquidationSideLong, Notional: 7, Time: now.Add(-2 * time.Hour)},
	}
	if SumLiquidationNotional(ev, LiquidationSideLong, now.Add(-time.Hour)) != 10 {
		t.Fatal("1h long")
	}
	if SumLiquidationNotional(ev, LiquidationSideLong, now.Add(-3*time.Hour)) != 17 {
		t.Fatal("24h long")
	}
}
