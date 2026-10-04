package domain

import (
	"testing"
	"time"
)

func TestPrintsFromFlowCells_KeepsCounts(t *testing.T) {
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	got := PrintsFromFlowCells([]FlowAtPricesCell{
		{Exchange: ExchangeBinance, Symbol: "btcusdt", Minute: at, Price: 100, Buy: 80, BuyN: 4, Sell: 10, SellN: 1},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Minute: at, Price: 101, Buy: 5, BuyN: 0},
	})
	if len(got) != 2 {
		t.Fatalf("prints %d", len(got))
	}
	if got[0].Count != 4 || got[0].Notional != 80 || got[0].Side != TakerSideBuy || got[0].Symbol != "BTCUSDT" {
		t.Fatalf("buy %+v", got[0])
	}
	if got[1].Count != 1 || got[1].Side != TakerSideSell {
		t.Fatalf("sell %+v", got[1])
	}
	venue := BuildPrintSizeVenue(ExchangeBinance, "BTCUSDT", got, at.Add(time.Minute))
	var avg float64
	for _, w := range venue.Windows {
		if w.Window == PrintSizeWindow5m {
			avg = w.Buy.Avg
		}
	}
	if avg != 20 {
		t.Fatalf("avg %v", avg)
	}
	counts := BuildOrderCount(ExchangeBinance, "BTCUSDT", got, at.Add(time.Minute))
	for _, w := range counts.Windows {
		if w.Window == OrderCount1m && w.BuyCount != 4 {
			t.Fatalf("count %+v", w)
		}
	}
}
