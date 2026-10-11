package domain

import (
	"strings"
	"testing"
	"time"
)

func TestEstimateNetPosition_SplitsOIAcrossSides(t *testing.T) {
	// 100 buy, 40 sell, open interest +30.
	// Longs +45, shorts −15. They add up to +30 and their gap is the taker delta.
	got := EstimateNetPosition(100, 40, 30)
	if !got.Complete || got.NetLong != 45 || got.NetShort != -15 {
		t.Fatalf("%+v", got)
	}
	if got.NetLong+got.NetShort != 30 {
		t.Fatalf("sum %v", got.NetLong+got.NetShort)
	}
}

func TestEstimateNetPosition_NoTakerVolume(t *testing.T) {
	got := EstimateNetPosition(0, 0, 12)
	if got.Complete || got.NetLong != 0 || got.Unexplained != 12 {
		t.Fatalf("%+v", got)
	}
}

func TestEstimateNetPosition_ClampsOIBeyondVolume(t *testing.T) {
	got := EstimateNetPosition(100, 40, 200)
	if got.Unexplained != 60 {
		t.Fatalf("unexplained %v", got.Unexplained)
	}
	if got.NetLong+got.NetShort != 140 {
		t.Fatalf("clamped sum %v %+v", got.NetLong+got.NetShort, got)
	}
}

func TestBuildNetPosition_Windows(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	series := &OpenInterestSeries{
		Exchange: ExchangeBinance,
		Symbol:   "BTCUSDT",
		Current:  OpenInterestPoint{Time: now, Value: 130},
		History: []OpenInterestPoint{
			{Time: now.Add(-4 * time.Hour), Value: 100},
			{Time: now.Add(-time.Hour), Value: 110},
			{Time: now.Add(-15 * time.Minute), Value: 120},
		},
	}
	buckets := []TakerBucket{
		{Start: now.Add(-4 * time.Hour), BuyNotional: 50, SellNotional: 10},
		{Start: now.Add(-time.Hour), BuyNotional: 40, SellNotional: 20},
		{Start: now.Add(-15 * time.Minute), BuyNotional: 30, SellNotional: 10},
		{Start: now.Add(-5 * time.Minute), BuyNotional: 20, SellNotional: 10},
	}
	got := BuildNetPosition("BTCUSDT", "binance", []NetPositionVenueInput{{
		Exchange: ExchangeBinance,
		Series:   series,
		Buckets:  buckets,
		Bar:      5 * time.Minute,
	}}, now)
	if len(got.Windows) != 3 {
		t.Fatalf("windows %d", len(got.Windows))
	}
	m15 := got.Windows[0]
	if m15.Window != "15m" || !m15.Complete {
		t.Fatalf("15m %+v", m15)
	}
	// Bars overlapping the last 15m: the -15m bar and the -5m bar.
	// buy 50, sell 20, oi 130-120 = 10.
	// netLong = (50-20+10)/2 = 20, netShort = (20-50+10)/2 = -10.
	if m15.Buy != 50 || m15.Sell != 20 || m15.OIDelta != 10 {
		t.Fatalf("15m inputs %+v", m15)
	}
	if m15.NetLong != 20 || m15.NetShort != -10 {
		t.Fatalf("15m nets %+v", m15)
	}
	if got.Summary == "" || got.Note == "" {
		t.Fatal("missing summary or note")
	}
}

func TestBuildNetPosition_MissingVenueBlanksWindow(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	series := &OpenInterestSeries{
		Exchange: ExchangeBinance,
		Current:  OpenInterestPoint{Time: now, Value: 130},
		History: []OpenInterestPoint{
			{Time: now.Add(-15 * time.Minute), Value: 120},
		},
	}
	buckets := []TakerBucket{
		{Start: now.Add(-15 * time.Minute), BuyNotional: 30, SellNotional: 10},
		{Start: now.Add(-5 * time.Minute), BuyNotional: 20, SellNotional: 10},
	}
	got := BuildNetPosition("BTCUSDT", "all", []NetPositionVenueInput{
		{Exchange: ExchangeBinance, Series: series, Buckets: buckets, Bar: 5 * time.Minute},
		{Exchange: ExchangeBybit, Error: "oi down"},
	}, now)
	m15 := got.Windows[0]
	if m15.Complete || m15.NetLong != 0 || m15.NetShort != 0 || m15.Buy != 0 {
		t.Fatalf("%+v", m15)
	}
	if len(m15.Missing) != 1 || m15.Missing[0] != "bybit" || !strings.Contains(m15.Summary, "bybit") {
		t.Fatalf("summary %+v", m15)
	}
}
