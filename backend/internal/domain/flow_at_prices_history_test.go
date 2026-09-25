package domain

import (
	"testing"
	"time"
)

func TestAccumulateFlowPrints_SkipsAlreadyStored(t *testing.T) {
	minute := time.Date(2026, 9, 25, 12, 4, 0, 0, time.UTC)
	prints := []TakerPrint{
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideSell, Price: 65010, Notional: 3, Time: minute.Add(-time.Minute)},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideBuy, Price: 65010, Notional: 5, Time: minute.Add(10 * time.Second)},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideSell, Price: 65040, Notional: 2, Time: minute.Add(20 * time.Second)},
	}
	cells, newest := AccumulateFlowPrints(prints, minute)
	if len(cells) != 2 || newest.Before(minute) {
		t.Fatalf("cells=%d newest=%s", len(cells), newest)
	}
	var buy, sell float64
	var buyN, sellN int
	for _, c := range cells {
		buy += c.Buy
		sell += c.Sell
		buyN += c.BuyN
		sellN += c.SellN
	}
	if buy != 5 || sell != 2 || buyN != 1 || sellN != 1 {
		t.Fatalf("buy=%v sell=%v n=%d/%d", buy, sell, buyN, sellN)
	}
}

func TestBuildFlowAtPricesFromCells_RollsHoursOntoRows(t *testing.T) {
	start := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	var cells []FlowAtPricesCell
	for i := 0; i < 4; i++ {
		cells = append(cells, FlowAtPricesCell{
			Exchange: ExchangeBybit, Symbol: "BTCUSDT",
			Minute: start.Add(time.Duration(i) * time.Hour),
			Price:  65000, Buy: 10, Sell: 1, BuyN: 3, SellN: 1,
		})
	}
	got := BuildFlowAtPricesFromCells(ExchangeBybit, "BTCUSDT", cells, 100, 65100, 0, 0, start, start.Add(4*time.Hour))
	if got.Error != "" || len(got.Levels) != 1 {
		t.Fatalf("%+v", got)
	}
	lv := got.Levels[0]
	if lv.BuyVolume != 40 || lv.SellVolume != 4 || lv.Trades != 16 || lv.Delta != 36 {
		t.Fatalf("level %+v", lv)
	}
	if !got.Complete {
		t.Fatalf("coverage %+v", got)
	}
}
