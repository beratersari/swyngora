package domain

import (
	"math"
	"strings"
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
	ep := rep.Episodes[0]
	if ep.StartedBy != string(ExchangeBinance) || ep.OrderUnknown {
		t.Fatalf("lead %+v", ep)
	}
	if len(ep.VenueStarts) != 2 {
		t.Fatalf("venue starts %+v", ep.VenueStarts)
	}
	if ep.VenueStarts[0].Exchange != string(ExchangeBinance) || ep.VenueStarts[0].LagSec != 0 {
		t.Fatalf("first venue %+v", ep.VenueStarts)
	}
	if ep.VenueStarts[1].Exchange != string(ExchangeBybit) || ep.VenueStarts[1].LagSec < 100 || ep.VenueStarts[1].LagSec > 140 {
		t.Fatalf("follow lag %+v", ep.VenueStarts)
	}
	if !strings.Contains(ep.Summary, "Binance first") || !strings.Contains(ep.Summary, "Bybit +") {
		t.Fatalf("summary %q", ep.Summary)
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

func TestMergeCascadeEpisodes_FollowSeconds(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeBinance, "BTCUSDT", LiquidationSideLong, 20, 8, 5, 64000, 63000)...)
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeBybit, "BTCUSDT", LiquidationSideLong, 20, 8, 17, 64010, 63020)...)
	rep := BuildCascadeReport("BTCUSDT", "all", ev, now)
	if len(rep.Episodes) == 0 || !rep.Episodes[0].Combined {
		t.Fatalf("episodes %+v", rep.Episodes)
	}
	ep := rep.Episodes[0]
	if ep.StartedBy != string(ExchangeBinance) || ep.OrderUnknown {
		t.Fatalf("lead %+v", ep)
	}
	if len(ep.VenueStarts) != 2 || ep.VenueStarts[1].LagSec != 12 {
		t.Fatalf("lag %+v", ep.VenueStarts)
	}
}

func TestMergeCascadeEpisodes_TiedStartIsUnknown(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeBinance, "ETHUSDT", LiquidationSideShort, 16, 6, 5, 3000, 2980)...)
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeBybit, "ETHUSDT", LiquidationSideShort, 16, 6, 5, 3002, 2982)...)
	rep := BuildCascadeReport("ETHUSDT", "all", ev, now)
	if len(rep.Episodes) == 0 || !rep.Episodes[0].Combined {
		t.Fatalf("episodes %+v", rep.Episodes)
	}
	ep := rep.Episodes[0]
	if !ep.OrderUnknown || ep.StartedBy != "" {
		t.Fatalf("expected unknown order %+v", ep)
	}
	if !strings.Contains(ep.Summary, "started together") {
		t.Fatalf("summary %q", ep.Summary)
	}
}

func TestMergeCascadeEpisodes_ThreeVenues(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeBinance, "SOLUSDT", LiquidationSideLong, 22, 8, 5, 140, 136)...)
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeBybit, "SOLUSDT", LiquidationSideLong, 22, 8, 20, 140, 136)...)
	ev = append(ev, cascadeQuietThenBurstOffset(now, ExchangeOKX, "SOLUSDT", LiquidationSideLong, 22, 8, 45, 140, 136)...)
	rep := BuildCascadeReport("SOLUSDT", "all", ev, now)
	if len(rep.Episodes) == 0 || !rep.Episodes[0].Combined {
		t.Fatalf("episodes %+v", rep.Episodes)
	}
	ep := rep.Episodes[0]
	if ep.StartedBy != string(ExchangeBinance) || len(ep.VenueStarts) != 3 {
		t.Fatalf("lead %+v", ep)
	}
	if ep.VenueStarts[1].Exchange != string(ExchangeBybit) || ep.VenueStarts[1].LagSec != 15 {
		t.Fatalf("bybit %+v", ep.VenueStarts)
	}
	if ep.VenueStarts[2].Exchange != string(ExchangeOKX) || ep.VenueStarts[2].LagSec != 40 {
		t.Fatalf("okx %+v", ep.VenueStarts)
	}
}

func TestMergeCascadeEpisodes_VenueShare(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	ev = append(ev, cascadeQuietThenBurstSized(now, ExchangeBinance, "BTCUSDT", LiquidationSideLong, 20, 8, 5, 64000, 63000, 2000)...)
	ev = append(ev, cascadeQuietThenBurstSized(now, ExchangeBybit, "BTCUSDT", LiquidationSideLong, 20, 8, 17, 64010, 63020, 8000)...)
	rep := BuildCascadeReport("BTCUSDT", "all", ev, now)
	if len(rep.Episodes) == 0 || !rep.Episodes[0].Combined {
		t.Fatalf("episodes %+v", rep.Episodes)
	}
	ep := rep.Episodes[0]
	if ep.StartedBy != string(ExchangeBinance) || ep.LargestBy != string(ExchangeBybit) {
		t.Fatalf("lead/largest %+v", ep)
	}
	if len(ep.VenueStarts) != 2 {
		t.Fatalf("venue starts %+v", ep.VenueStarts)
	}
	bn, bb := ep.VenueStarts[0], ep.VenueStarts[1]
	if bn.SharePct < 18 || bn.SharePct > 22 || parseQty(bn.TotalNotional) < 80000 {
		t.Fatalf("binance share %+v", bn)
	}
	if bb.SharePct < 78 || bb.SharePct > 82 || parseQty(bb.TotalNotional) < 300000 {
		t.Fatalf("bybit share %+v", bb)
	}
	if math.Abs(bn.SharePct+bb.SharePct-100) > 0.15 {
		t.Fatalf("shares should sum to 100 %+v", ep.VenueStarts)
	}
	if !strings.Contains(ep.Summary, "Bybit 80%") || !strings.Contains(ep.Summary, "Most of the wave was Bybit") {
		t.Fatalf("summary %q", ep.Summary)
	}
}

func TestCollapseCascadePartsByVenue(t *testing.T) {
	t1 := time.Date(2026, 8, 30, 15, 21, 5, 0, time.UTC)
	t2 := t1.Add(3 * time.Minute)
	got := collapseCascadePartsByVenue([]CascadeEpisode{
		{Exchange: string(ExchangeBybit), TotalNotional: "100", LongNotional: "100", FirstPrintAt: t1, StartedAt: t1, Count: 2},
		{Exchange: string(ExchangeOKX), TotalNotional: "50", LongNotional: "50", FirstPrintAt: t1.Add(time.Second), StartedAt: t1, Count: 1},
		{Exchange: string(ExchangeBybit), TotalNotional: "25", LongNotional: "25", FirstPrintAt: t2, StartedAt: t2, Count: 1},
	})
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].Exchange != string(ExchangeBybit) || parseQty(got[0].TotalNotional) != 125 || got[0].Count != 3 {
		t.Fatalf("bybit %+v", got[0])
	}
	if !got[0].FirstPrintAt.Equal(t1) {
		t.Fatalf("kept earlier print %+v", got[0].FirstPrintAt)
	}
	if got[1].Exchange != string(ExchangeOKX) || parseQty(got[1].TotalNotional) != 50 {
		t.Fatalf("okx %+v", got[1])
	}
}

func TestCascadeSharePcts_Remainder(t *testing.T) {
	got := cascadeSharePcts([]float64{1, 1, 1})
	if len(got) != 3 || got[0]+got[1]+got[2] != 100 {
		t.Fatalf("%+v", got)
	}
}

func cascadeQuietThenBurstOffset(now time.Time, ex Exchange, symbol, side string, burstAgo, burstMins, startOffsetSec int, startPx, endPx float64) []LiquidationEvent {
	return cascadeQuietThenBurstSized(now, ex, symbol, side, burstAgo, burstMins, startOffsetSec, startPx, endPx, 2000)
}

func cascadeQuietThenBurstSized(now time.Time, ex Exchange, symbol, side string, burstAgo, burstMins, startOffsetSec int, startPx, endPx, notional float64) []LiquidationEvent {
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
	if startOffsetSec < 0 {
		startOffsetSec = 0
	}
	if notional <= 0 {
		notional = 2000
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
				Price: px, Quantity: 1, Notional: notional,
				Time: t0.Add(time.Duration(startOffsetSec+i) * time.Second),
			})
		}
	}
	return ev
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
