package domain

import (
	"strconv"
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

func TestHaltCandleEnd(t *testing.T) {
	halt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	if !HaltCandleEnd(halt, now).Equal(halt.Add(24 * time.Hour)) {
		t.Fatal("want halt+24h")
	}
	sameDay := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	if !HaltCandleEnd(halt, sameDay).Equal(sameDay) {
		t.Fatal("must not request after now")
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
	if !got.Halted {
		t.Fatal("last-print ticker must be marked halted")
	}
}

func TestApplyTickerToSpotDoesNotOverwriteLive(t *testing.T) {
	m := SpotMarket{LastPrice: "5", Volume: ""}
	ApplyTickerToSpot(&m, Ticker24h{LastPrice: "9", Volume: "100"})
	if m.LastPrice != "5" || m.Volume != "100" {
		t.Fatalf("%+v", m)
	}
}

func TestSyncTickerChangeFromOpenLast(t *testing.T) {
	tkr := Ticker24h{OpenPrice: "90", LastPrice: "100", PriceChangePercent: "1.5"}
	SyncTickerChangeFromOpenLast(&tkr)
	if tkr.PriceChange != "10" {
		t.Fatalf("change=%s", tkr.PriceChange)
	}
	pct, err := strconv.ParseFloat(tkr.PriceChangePercent, 64)
	if err != nil || pct < 11.11 || pct > 11.12 {
		t.Fatalf("pct=%s", tkr.PriceChangePercent)
	}
}

func TestBlankHaltedSpotChange(t *testing.T) {
	past := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	future := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	items := []SpotMarket{
		{Symbol: "BTCUSDT", PriceChangePercent: "1.2", PriceChange: "100"},
		{Symbol: "PYRUSDT", PriceChangePercent: "-57.1", PriceChange: "-0.03", DelistTime: &past},
		{Symbol: "ICXUSDT", PriceChangePercent: "-8.2", PriceChange: "-0.001", DelistTime: &future},
	}
	BlankHaltedSpotChange(items, now)
	if items[0].PriceChangePercent != "1.2" {
		t.Fatalf("live row cleared: %+v", items[0])
	}
	if items[1].PriceChangePercent != "" || items[1].PriceChange != "" {
		t.Fatalf("halted 24h still live: %+v", items[1])
	}
	if items[2].PriceChangePercent != "-8.2" {
		t.Fatalf("upcoming delist should keep 24h: %+v", items[2])
	}
}
