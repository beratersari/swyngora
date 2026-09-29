package domain

import (
	"testing"
	"time"
)

func TestFuturesHistoryPageBudgetCoversNinetyDays(t *testing.T) {
	if FuturesHistoryMaxLookback != 90*24*time.Hour {
		t.Fatalf("lookback %v", FuturesHistoryMaxLookback)
	}
	hours := int(FuturesHistoryMaxLookback / time.Hour)
	if hours > MaxHuntHeatmapHistory {
		t.Fatalf("hourly rows %d exceed heatmap cap %d", hours, MaxHuntHeatmapHistory)
	}
	for _, size := range []int{100, 200, 500} {
		pages := FuturesHistoryPageBudget(size)
		if pages*size < hours {
			t.Fatalf("page size %d budget %d covers %d hours, need %d", size, pages, pages*size, hours)
		}
	}
	if FuturesHistoryPageBudget(0) != 1 {
		t.Fatal("empty page size")
	}
}

func TestPublicFuturesHistoryLookbackFollowsVenueDocs(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	if got := PublicFuturesHistoryLookback(ExchangeBinance, FuturesMetricOpenInterest, now); got != BinanceOpenInterestHistoryLookback {
		t.Fatalf("binance oi %v", got)
	}
	if got := PublicFuturesHistoryLookback(ExchangeBinance, FuturesMetricLongShort, now); got != BinanceLongShortHistoryLookback {
		t.Fatalf("binance long short %v", got)
	}
	if got := PublicFuturesHistoryLookback(ExchangeBinance, FuturesMetricFunding, now); got != FuturesHistoryMaxLookback {
		t.Fatalf("binance funding %v", got)
	}
	for _, metric := range []string{FuturesMetricOpenInterest, FuturesMetricLongShort, FuturesMetricFunding} {
		if got := PublicFuturesHistoryLookback(ExchangeBybit, metric, now); got != FuturesHistoryMaxLookback {
			t.Fatalf("bybit %s %v", metric, got)
		}
	}
	if got := PublicFuturesHistoryLookback(ExchangeOKX, FuturesMetricOpenInterest, now); got != OKXOpenInterestHistoryEntries*time.Hour {
		t.Fatalf("okx oi %v", got)
	}
	if got := PublicFuturesHistoryLookback(ExchangeOKX, FuturesMetricLongShort, now); got != 0 {
		t.Fatalf("okx long short %v", got)
	}
	// 28 Sep is 92 days after 28 Jun, so three months is longer than 90 days.
	if got := PublicFuturesHistoryLookback(ExchangeOKX, FuturesMetricFunding, now); got != FuturesHistoryMaxLookback {
		t.Fatalf("okx funding september %v", got)
	}
	may := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	want := may.Sub(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if got := PublicFuturesHistoryLookback(ExchangeOKX, FuturesMetricFunding, may); got != want {
		t.Fatalf("okx funding may %v want %v", got, want)
	}
	wideFrom := now.Add(-200 * 24 * time.Hour)
	clamped := ClampPublicHistoryFrom(ExchangeBinance, FuturesMetricOpenInterest, wideFrom, now)
	if !clamped.Equal(now.Add(-BinanceOpenInterestHistoryLookback)) {
		t.Fatalf("clamped binance oi from %s", clamped)
	}
	shortFrom := now.Add(-2 * time.Hour)
	if got := ClampPublicHistoryFrom(ExchangeBinance, FuturesMetricOpenInterest, shortFrom, now); !got.Equal(shortFrom) {
		t.Fatalf("short window moved to %s", got)
	}
}
