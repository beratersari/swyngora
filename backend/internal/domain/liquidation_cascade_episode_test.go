package domain

import (
	"testing"
	"time"
)

func cascadeQuietThenBurst(now time.Time, ex Exchange, symbol, side string, burstAgo, burstMins int, startPx, endPx float64) []LiquidationEvent {
	var ev []LiquidationEvent
	for i := 1; i <= 80; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ex, Symbol: symbol, Side: side,
			Price: startPx, Quantity: 0.01, Notional: 80,
			Time: now.Add(-time.Duration(i*5) * time.Minute),
		})
	}
	if burstMins < 1 {
		burstMins = 1
	}
	for m := 0; m < burstMins; m++ {
		t0 := now.Add(-time.Duration(burstAgo+burstMins-1-m) * time.Minute)
		px := startPx
		if burstMins > 1 {
			px = startPx + (endPx-startPx)*float64(m)/float64(burstMins-1)
		} else {
			px = endPx
		}
		for i := 0; i < 6; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: symbol, Side: side,
				Price: px, Quantity: 1, Notional: 2000,
				Time: t0.Add(time.Duration(5+i) * time.Second),
			})
		}
	}
	return ev
}

func TestDetectCascadeEpisodes_ClosedWave(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	ev := cascadeQuietThenBurst(now, ExchangeBinance, "BTCUSDT", LiquidationSideLong, 30, 10, 64000, 63000)
	got := DetectCascadeEpisodes(ExchangeBinance, "BTCUSDT", ev, now)
	if len(got) != 1 {
		t.Fatalf("episodes %+v", got)
	}
	ep := got[0]
	if ep.Open || ep.Side != LiquidationSideLong {
		t.Fatalf("open/side %+v", ep)
	}
	if cascadeGradeRank(ep.Grade) < cascadeGradeRank(CascadeGradeCascade) {
		t.Fatalf("grade %+v", ep)
	}
	if ep.DurationSec < 8*60 || ep.DurationSec > 12*60 {
		t.Fatalf("duration %d %+v", ep.DurationSec, ep)
	}
	if parseQty(ep.LongNotional) < 100000 {
		t.Fatalf("long %+v", ep)
	}
	if ep.PriceChangePct == "" || ep.PriceChangePct[0] != '-' {
		t.Fatalf("price move %+v", ep)
	}
	if DetectCascadeEpisodes(ExchangeBybit, "BTCUSDT", ev, now) != nil && len(DetectCascadeEpisodes(ExchangeBybit, "BTCUSDT", ev, now)) != 0 {
		t.Fatalf("bybit should have no wave")
	}
}

func TestDetectCascadeEpisodes_StillOpen(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	ev := cascadeQuietThenBurst(now, ExchangeBinance, "ETHUSDT", LiquidationSideShort, 0, 3, 3000, 3010)
	got := DetectCascadeEpisodes(ExchangeBinance, "ETHUSDT", ev, now)
	if len(got) != 1 || !got[0].Open || got[0].EndedAt.IsZero() == false {
		t.Fatalf("%+v", got)
	}
}

func TestMergeCascadeEpisodes_SameSideOverlap(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	ev = append(ev, cascadeQuietThenBurst(now, ExchangeBinance, "BTCUSDT", LiquidationSideShort, 20, 8, 65000, 64000)...)
	ev = append(ev, cascadeQuietThenBurst(now, ExchangeBybit, "BTCUSDT", LiquidationSideShort, 18, 8, 65010, 64020)...)
	rep := BuildCascadeReport("BTCUSDT", "all", ev, now)
	if len(rep.Episodes) == 0 || !rep.Episodes[0].Combined || rep.Episodes[0].Side != LiquidationSideShort {
		t.Fatalf("episodes %+v", rep.Episodes)
	}
	if parseQty(rep.Episodes[0].ShortNotional) < 150000 {
		t.Fatalf("combined short %+v", rep.Episodes[0])
	}
}

func TestMergeCascadeEpisodes_OppositeSidesStaySeparate(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	ev = append(ev, cascadeQuietThenBurst(now, ExchangeBinance, "SOLUSDT", LiquidationSideLong, 20, 6, 140, 138)...)
	ev = append(ev, cascadeQuietThenBurst(now, ExchangeBybit, "SOLUSDT", LiquidationSideShort, 20, 6, 140, 142)...)
	rep := BuildCascadeReport("SOLUSDT", "all", ev, now)
	combined := 0
	for _, ep := range rep.Episodes {
		if ep.Combined {
			combined++
		}
	}
	if combined != 0 || len(rep.Episodes) < 2 {
		t.Fatalf("expected two venue waves %+v", rep.Episodes)
	}
}

func TestApplyCascadeCandlePrices(t *testing.T) {
	ep := CascadeEpisode{
		Symbol: "BTCUSDT", Exchange: string(ExchangeBinance), Side: LiquidationSideLong,
		Grade: CascadeGradeCascade, StartedAt: time.Date(2026, 8, 30, 15, 20, 0, 0, time.UTC),
		EndedAt: time.Date(2026, 8, 30, 15, 30, 0, 0, time.UTC), DurationSec: 600,
		LongNotional: "10000", ShortNotional: "0", TotalNotional: "10000",
		PriceOpen: "64000", PriceClose: "63900", PriceChangePct: "-0.16",
	}
	bars := []Candle{
		{OpenTime: ep.StartedAt, Open: "64100", High: "64200", Low: "63800", Close: "63950"},
		{OpenTime: ep.StartedAt.Add(5 * time.Minute), Open: "63950", High: "64000", Low: "62800", Close: "62850"},
	}
	ApplyCascadeCandlePrices(&ep, bars, ep.EndedAt)
	if ep.PriceOpen != "64100" || ep.PriceClose != "62850" || ep.PriceChangePct == "" || ep.PriceChangePct[0] != '-' {
		t.Fatalf("%+v", ep)
	}
}
