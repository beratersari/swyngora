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
