package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBuildPrintSizeVenue_FewLargeBuys(t *testing.T) {
	now := time.Date(2026, 9, 19, 16, 0, 0, 0, time.UTC)
	var prints []TakerPrint
	// Prior 15m: many 20k buys.
	for i := 0; i < 40; i++ {
		prints = append(prints, TakerPrint{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideBuy,
			Notional: 20_000, Time: now.Add(-20*time.Minute + time.Duration(i)*time.Second),
		})
	}
	// Current 15m: few 180k buys (same total-ish volume from larger prints).
	for i := 0; i < 6; i++ {
		prints = append(prints, TakerPrint{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideBuy,
			Notional: 180_000, Time: now.Add(-10*time.Minute + time.Duration(i)*time.Second),
		})
	}
	// Stable small sells both windows.
	for i := 0; i < 10; i++ {
		prints = append(prints, TakerPrint{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideSell,
			Notional: 25_000, Time: now.Add(-20*time.Minute + time.Duration(i)*time.Second),
		})
		prints = append(prints, TakerPrint{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideSell,
			Notional: 25_000, Time: now.Add(-8*time.Minute + time.Duration(i)*time.Second),
		})
	}
	got := BuildPrintSizeVenue(ExchangeBinance, "btcusdt", prints, now)
	w := pickPrintSizeWindow(got.Windows, PrintSizeWindow15m)
	if w.Buy.Avg < 179_000 || w.Buy.Avg > 181_000 {
		t.Fatalf("buy avg %+v", w.Buy)
	}
	if w.Buy.PriorAvg < 19_000 || w.Buy.PriorAvg > 21_000 {
		t.Fatalf("prior %+v", w.Buy)
	}
	if w.Buy.Kind != PrintSizeKindJumped || w.Buy.Ratio < 8 {
		t.Fatalf("kind %+v", w.Buy)
	}
	if w.Read != PrintSizeReadFewLarge {
		t.Fatalf("read %s", w.Read)
	}
	if !strings.Contains(got.Summary, "180") || !strings.Contains(got.Summary, "few large") {
		t.Fatalf("summary %s", got.Summary)
	}
	if !w.Complete {
		t.Fatal("15m should be complete")
	}
}

func TestBuildPrintSizeOverview_AveragesNotionalOverCount(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	btcBuy := printSizeSide(400, 2, 40, 2)
	ethBuy := printSizeSide(20, 2, 80, 2)
	btc := PrintSizeVenue{Symbol: "BTCUSDT", Windows: []PrintSizeWindow{{
		Window: PrintSizeWindow15m, Buy: btcBuy, Sell: printSizeSide(10, 1, 10, 1), Complete: true,
	}}}
	btc.Windows[0].Read = printSizeRead(btc.Windows[0].Buy, btc.Windows[0].Sell)
	eth := PrintSizeVenue{Symbol: "ETHUSDT", Windows: []PrintSizeWindow{{
		Window: PrintSizeWindow15m, Buy: ethBuy, Sell: printSizeSide(10, 1, 10, 1), Complete: true,
	}}}
	eth.Windows[0].Read = printSizeRead(eth.Windows[0].Buy, eth.Windows[0].Sell)
	quiet := PrintSizeVenue{Symbol: "SOLUSDT", Windows: []PrintSizeWindow{{Window: PrintSizeWindow15m}}}
	got := BuildPrintSizeOverview("binance", "USDT", []PrintSizeVenue{
		eth, btc, quiet, {Symbol: "XRPUSDT", Error: "down"},
	}, now)
	if got.Universe != 3 || got.Coins[0].Symbol != "BTCUSDT" {
		t.Fatalf("rank universe=%d first=%s", got.Universe, got.Coins[0].Symbol)
	}
	w := got.Windows[1]
	if w.Window != PrintSizeWindow15m {
		w = got.Windows[0]
	}
	// 420 / 4 = 105 vs 120 / 4 = 30. Averaging 200 and 10 would hide the busier coin.
	if w.Buy.Avg < 104 || w.Buy.Avg > 106 || w.Buy.PriorAvg < 29 || w.Buy.PriorAvg > 31 || w.Buy.Kind != PrintSizeKindJumped {
		t.Fatalf("market buy %+v", w.Buy)
	}
	if w.LargerCoins+w.FewLargeCoins < 1 || w.SmallerCoins < 1 || w.QuietCoins != 1 {
		t.Fatalf("votes %+v btc=%s eth=%s", w, btc.Windows[0].Read, eth.Windows[0].Read)
	}
}

func TestParsePrintSizeOverviewLimit(t *testing.T) {
	if ParsePrintSizeOverviewLimit(0) != 20 || ParsePrintSizeOverviewLimit(2) != 5 || ParsePrintSizeOverviewLimit(99) != 30 {
		t.Fatal("limit clamp")
	}
}

func TestBuildPrintSizeVenue_ManySmallBuys(t *testing.T) {
	now := time.Date(2026, 9, 19, 16, 0, 0, 0, time.UTC)
	var prints []TakerPrint
	for i := 0; i < 10; i++ {
		prints = append(prints, TakerPrint{
			Side: TakerSideBuy, Notional: 50_000,
			Time: now.Add(-20*time.Minute + time.Duration(i)*time.Second),
		})
	}
	for i := 0; i < 80; i++ {
		prints = append(prints, TakerPrint{
			Side: TakerSideBuy, Notional: 50_000,
			Time: now.Add(-10*time.Minute + time.Duration(i)*time.Second),
		})
	}
	got := BuildPrintSizeVenue(ExchangeBybit, "BTCUSDT", prints, now)
	w := pickPrintSizeWindow(got.Windows, PrintSizeWindow15m)
	if w.Read != PrintSizeReadManySmall {
		t.Fatalf("read %s %+v", w.Read, w.Buy)
	}
}

func TestCombinePrintSizeVenues(t *testing.T) {
	now := time.Date(2026, 9, 19, 16, 0, 0, 0, time.UTC)
	bin := BuildPrintSizeVenue(ExchangeBinance, "BTCUSDT", []TakerPrint{
		{Side: TakerSideBuy, Notional: 100_000, Time: now.Add(-2 * time.Minute)},
		{Side: TakerSideBuy, Notional: 100_000, Time: now.Add(-17 * time.Minute)},
	}, now)
	byb := BuildPrintSizeVenue(ExchangeBybit, "BTCUSDT", []TakerPrint{
		{Side: TakerSideBuy, Notional: 50_000, Time: now.Add(-2 * time.Minute)},
		{Side: TakerSideBuy, Notional: 50_000, Time: now.Add(-17 * time.Minute)},
	}, now)
	got := CombinePrintSizeVenues("BTCUSDT", []PrintSizeVenue{bin, byb})
	if got == nil {
		t.Fatal("nil")
	}
	w := pickPrintSizeWindow(got.Windows, PrintSizeWindow5m)
	if w.Buy.Count != 1 && w.Buy.Avg != 0 {
		// 5m current: bin 100k + byb 50k = 150k / 2 = 75k
	}
	w5 := pickPrintSizeWindow(got.Windows, PrintSizeWindow5m)
	if w5.Buy.Count != 2 || w5.Buy.Avg < 74_000 || w5.Buy.Avg > 76_000 {
		t.Fatalf("combined 5m %+v", w5.Buy)
	}
}

func TestPrintSizeSideKinds(t *testing.T) {
	j := printSizeSide(180_000, 1, 20_000, 1)
	if j.Kind != PrintSizeKindJumped || j.Ratio < 8 {
		t.Fatalf("%+v", j)
	}
	f := printSizeSide(10_000, 1, 40_000, 1)
	if f.Kind != PrintSizeKindFaded {
		t.Fatalf("%+v", f)
	}
	s := printSizeSide(22_000, 1, 20_000, 1)
	if s.Kind != PrintSizeKindStable {
		t.Fatalf("%+v", s)
	}
}
