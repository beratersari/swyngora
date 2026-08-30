package domain

import (
	"testing"
	"time"
)

func TestDetectCascadeVenue_BurstVsTypical(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	// Quiet hour: one small long every 5 minutes.
	for i := 1; i <= 70; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Price: 64000, Quantity: 0.01, Notional: 80, Time: now.Add(-time.Duration(i*5) * time.Minute),
		})
	}
	// Last minute: many longs.
	for i := 0; i < 6; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Price: 64000, Quantity: 1, Notional: 2000, Time: now.Add(-time.Duration(5+i) * time.Second),
		})
	}
	got := DetectCascadeVenue(ExchangeBinance, "BTCUSDT", ev, now)
	if got.Side != LiquidationSideLong {
		t.Fatalf("side %+v", got)
	}
	if cascadeGradeRank(got.Grade) < cascadeGradeRank(CascadeGradeCascade) {
		t.Fatalf("grade %s score %.1f windows %+v", got.Grade, got.Score, got.Windows)
	}
	quiet := DetectCascadeVenue(ExchangeBybit, "BTCUSDT", ev, now)
	if quiet.Grade != CascadeGradeQuiet {
		t.Fatalf("bybit must stay quiet %+v", quiet)
	}
}

func TestDetectCascadeBoth_SameSide(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit} {
		for i := 1; i <= 40; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: "ETHUSDT", Side: LiquidationSideShort,
				Notional: 60, Time: now.Add(-time.Duration(i*6) * time.Minute),
			})
		}
		for i := 0; i < 6; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: "ETHUSDT", Side: LiquidationSideShort,
				Notional: 3000, Time: now.Add(-time.Duration(4+i) * time.Second),
			})
		}
	}
	rep := BuildCascadeReport("ETHUSDT", "all", ev, now)
	if rep.Both == nil || !rep.Both.Agree || rep.Both.Side != LiquidationSideShort {
		t.Fatalf("both %+v venues %+v", rep.Both, rep.Venues)
	}
}

func TestBuildCascadeScan_MarketAndCoins(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	for i := 1; i <= 40; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: LiquidationSideLong,
			Notional: 50, Time: now.Add(-time.Duration(i*6) * time.Minute),
		})
	}
	for i := 0; i < 6; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: LiquidationSideLong,
			Notional: 2500, Time: now.Add(-time.Duration(3+i) * time.Second),
		})
	}
	ev = append(ev, LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "AAAUSDT", Side: LiquidationSideShort,
		Notional: 40, Time: now.Add(-2 * time.Hour),
	})
	scan := BuildCascadeScan("all", ev, now)
	if len(scan.Hits) == 0 || scan.Hits[0].Symbol != "SOLUSDT" {
		t.Fatalf("hits %+v", scan.Hits)
	}
	if scan.Market.Symbol != "all" || len(scan.Market.Venues) != 2 {
		t.Fatalf("market %+v", scan.Market)
	}
}

func TestEventsSince_AllAndVenue(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Now().UTC()
	b.Record(LiquidationEvent{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong, Notional: 1, Time: now})
	b.Record(LiquidationEvent{Exchange: ExchangeBybit, Symbol: "ETHUSDT", Side: LiquidationSideShort, Notional: 2, Time: now})
	all := b.EventsSince("all", "all", now.Add(-time.Minute))
	if len(all) != 2 {
		t.Fatalf("all %d", len(all))
	}
	bn := b.EventsSince("binance", "", now.Add(-time.Minute))
	if len(bn) != 1 || bn[0].Exchange != ExchangeBinance {
		t.Fatalf("binance %+v", bn)
	}
}
