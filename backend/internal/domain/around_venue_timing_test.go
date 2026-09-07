package domain

import (
	"strings"
	"testing"
	"time"
)

func TestFindAroundPriceRunStart_FindsFirstPrint(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	plan := AroundPlan{
		At: at, DuringFrom: at, DuringTo: at.Add(15 * time.Minute),
		BeforeFrom: at.Add(-time.Hour),
	}
	var bars []AroundBar
	px := 100.0
	for i := -20; i < 15; i++ {
		t0 := at.Add(time.Duration(i) * time.Minute)
		if i >= 3 {
			px = 101.5 + float64(i-3)*0.2
		}
		bars = append(bars, AroundBar{Time: t0, Open: px, Close: px, High: px, Low: px})
	}
	got := FindAroundPriceRunStart(bars, plan)
	if got.Before(at.Add(2*time.Minute)) || got.After(at.Add(5*time.Minute)) {
		t.Fatalf("start %s", got)
	}
}

func TestCompareAroundVenueTiming_BinanceEarlier(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	got := CompareAroundVenueTiming([]AroundVenue{
		{Exchange: ExchangeBinance, MoveStart: at.Add(-3 * time.Minute)},
		{Exchange: ExchangeBybit, MoveStart: at.Add(5 * time.Minute)},
	})
	if got == nil || got.Agreement != AroundVenueBinanceEarlier {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "Binance started") {
		t.Fatalf("summary %q", got.Summary)
	}
	same := CompareAroundVenueTiming([]AroundVenue{
		{Exchange: ExchangeBinance, MoveStart: at},
		{Exchange: ExchangeBybit, MoveStart: at.Add(time.Minute)},
	})
	if same == nil || same.Agreement != AroundVenueSame {
		t.Fatalf("same %+v", same)
	}
}
