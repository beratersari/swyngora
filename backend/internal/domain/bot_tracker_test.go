package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBuildBotTracker_FlagsSameSizeInsideOneMinute(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	var prints []TakerPrint
	for i := 0; i < 6; i++ {
		prints = append(prints, TakerPrint{
			Side: TakerSideBuy, Notional: 57889,
			Time: now.Add(-40*time.Second + time.Duration(i)*6*time.Second),
		})
	}
	for i := 0; i < 3; i++ {
		prints = append(prints, TakerPrint{
			Side: TakerSideBuy, Notional: 120000,
			Time: now.Add(-20*time.Second + time.Duration(i)*time.Second),
		})
	}
	got := BuildBotTracker(ExchangeBinance, "BTCUSDT", prints, now)
	if len(got.Clusters) != 1 || got.Clusters[0].Count != 6 || got.Clusters[0].Side != TakerSideBuy {
		t.Fatalf("%+v", got.Clusters)
	}
	if got.BuyCount != 6 || got.Delta != 6 {
		t.Fatalf("counts %+v", got)
	}
	if !strings.Contains(got.Summary, "buy") || !strings.Contains(got.Summary, "× 6") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestBuildBotTracker_IgnoresRepeatsSpreadPastOneMinute(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	var prints []TakerPrint
	for i := 0; i < 6; i++ {
		prints = append(prints, TakerPrint{
			Side: TakerSideSell, Notional: 25000,
			Time: now.Add(-4*time.Minute + time.Duration(i)*40*time.Second),
		})
	}
	got := BuildBotTracker(ExchangeBybit, "ETHUSDT", prints, now)
	if len(got.Clusters) != 0 {
		t.Fatalf("spread repeats should not flag: %+v", got.Clusters)
	}
}

func TestBuildBotTracker_SeparatesSidesAndSkipsDust(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	var prints []TakerPrint
	for i := 0; i < 5; i++ {
		prints = append(prints, TakerPrint{
			Side: TakerSideSell, Notional: 80000,
			Time: now.Add(-30*time.Second + time.Duration(i)*4*time.Second),
		})
		prints = append(prints, TakerPrint{
			Side: TakerSideBuy, Notional: 100,
			Time: now.Add(-30*time.Second + time.Duration(i)*4*time.Second),
		})
	}
	got := BuildBotTracker(ExchangeOKX, "BTCUSDT", prints, now)
	if len(got.Clusters) != 1 || got.Clusters[0].Side != TakerSideSell || got.Clusters[0].Count != 5 {
		t.Fatalf("%+v", got.Clusters)
	}
}
