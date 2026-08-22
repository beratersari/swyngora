package domain

import (
	"testing"
	"time"
)

func TestDelistVisibleOnTradingList(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	inTwoWeeks := now.Add(14 * 24 * time.Hour)
	inTwoMonths := now.Add(60 * 24 * time.Hour)
	yesterday := now.Add(-12 * time.Hour)
	lastWeek := now.Add(-8 * 24 * time.Hour)
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	fortyDaysAgo := now.Add(-40 * 24 * time.Hour)
	if !DelistVisibleOnTradingList(inTwoWeeks, now) {
		t.Fatal("14d should show")
	}
	if DelistVisibleOnTradingList(inTwoMonths, now) {
		t.Fatal("60d should not promote onto TRADING list")
	}
	if !DelistVisibleOnTradingList(yesterday, now) {
		t.Fatal("12h ago should still show")
	}
	if !DelistVisibleOnTradingList(lastWeek, now) {
		t.Fatal("8d ago should still show")
	}
	if !DelistVisibleOnTradingList(thirtyDaysAgo, now) {
		t.Fatal("exactly 30d ago should still show")
	}
	if DelistVisibleOnTradingList(fortyDaysAgo, now) {
		t.Fatal("40d ago should not show")
	}
	if DelistVisibleOnTradingList(time.Time{}, now) {
		t.Fatal("zero time")
	}
}

func TestTickerFromLastCandle(t *testing.T) {
	got := TickerFromLastCandle("vanryusdt", Candle{Open: "1", Close: "1.1", High: "1.2", Low: "0.9", Volume: "10", QuoteVolume: "11"})
	if got.Symbol != "VANRYUSDT" || got.LastPrice != "1.1" || got.QuoteVolume != "11" {
		t.Fatalf("%+v", got)
	}
	if got.PriceChangePercent != "10.00" {
		t.Fatalf("pct=%s", got.PriceChangePercent)
	}
}

func TestApplyTickerToSpotDoesNotOverwriteLive(t *testing.T) {
	m := SpotMarket{LastPrice: "5", Volume: ""}
	ApplyTickerToSpot(&m, Ticker24h{LastPrice: "9", Volume: "100"})
	if m.LastPrice != "5" || m.Volume != "100" {
		t.Fatalf("%+v", m)
	}
}
