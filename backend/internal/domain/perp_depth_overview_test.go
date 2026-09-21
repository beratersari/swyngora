package domain

import (
	"strings"
	"testing"
	"time"
)

func TestParsePerpDepthOverviewLimit(t *testing.T) {
	if got := ParsePerpDepthOverviewLimit(0); got != PerpDepthOverviewDefaultN {
		t.Fatalf("%d", got)
	}
	if got := ParsePerpDepthOverviewLimit(2); got != PerpDepthOverviewMinN {
		t.Fatalf("%d", got)
	}
	if got := ParsePerpDepthOverviewLimit(99); got != PerpDepthOverviewMaxN {
		t.Fatalf("%d", got)
	}
}

func TestBuildPerpDepthOverview_RanksCoinShareOfTotal(t *testing.T) {
	asOf := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	got := BuildPerpDepthOverview("all", 1, []PerpDepthCoinInput{
		{Symbol: "ETHUSDT", Base: "ETH", Bid: 10_000_000, Ask: 10_000_000, Lean: PerpDepthLeanEven, Complete: true},
		{Symbol: "BTCUSDT", Base: "BTC", Bid: 80_000_000, Ask: 20_000_000, Lean: PerpDepthLeanBid, Complete: true},
		{Symbol: "SOLUSDT", Base: "SOL", Error: "no perpetual depth"},
	}, asOf)
	if got.Known != 2 || got.Universe != 3 {
		t.Fatalf("known=%d universe=%d", got.Known, got.Universe)
	}
	if got.Bid != 90_000_000 || got.Ask != 30_000_000 {
		t.Fatalf("totals bid=%v ask=%v", got.Bid, got.Ask)
	}
	if got.Lean != "bid" {
		t.Fatalf("lean %s", got.Lean)
	}
	if len(got.Coins) != 3 || got.Coins[0].Base != "BTC" || got.Coins[1].Base != "ETH" {
		t.Fatalf("order %+v", got.Coins)
	}
	btc := got.Coins[0]
	if btc.SharePct < 82 || btc.SharePct > 84 { // 100/120 = 83.33
		t.Fatalf("btc share %v", btc.SharePct)
	}
	if btc.BidSharePct < 88 || btc.BidSharePct > 90 { // 80/90
		t.Fatalf("btc bid share %v", btc.BidSharePct)
	}
	if len(got.Leaders) < 2 || got.Leaders[0] != "BTC" || got.Leaders[1] != "ETH" {
		t.Fatalf("leaders %v", got.Leaders)
	}
	if len(got.Missing) != 1 || got.Missing[0] != "SOLUSDT" {
		t.Fatalf("missing %v", got.Missing)
	}
	if !strings.Contains(got.Summary, "2 liquid perps") || !strings.Contains(got.Summary, "BTC") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestBuildPerpDepthOverview_Empty(t *testing.T) {
	got := BuildPerpDepthOverview("binance", 1, nil, time.Time{})
	if got.Known != 0 || got.Summary == "" {
		t.Fatalf("%+v", got)
	}
}

func TestExplainPerpDepthOverview_NoBook(t *testing.T) {
	got := ExplainPerpDepthOverview(PerpDepthOverview{})
	if !strings.Contains(got, "No perpetual depth") {
		t.Fatalf("%s", got)
	}
}

func TestIsPerpDepthEligible_DropsStables(t *testing.T) {
	if !IsPerpDepthEligible("SOL") || IsPerpDepthEligible("USDC") || IsPerpDepthEligible("BTCUP") {
		t.Fatal("eligible")
	}
}

func TestPerpOverviewLean_MapsBuySell(t *testing.T) {
	if perpOverviewLean("buy", 2, 1) != "bid" || perpOverviewLean("sell", 1, 2) != "ask" {
		t.Fatal("map")
	}
	if perpOverviewLean("", 80, 20) != "bid" {
		t.Fatal("from sizes")
	}
}
