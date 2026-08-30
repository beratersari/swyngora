package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetLiquidationCascade_SeparatesVenues(t *testing.T) {
	now := time.Now().UTC()
	book := domain.NewLiquidationBook()
	for i := 1; i <= 40; i++ {
		book.Record(domain.LiquidationEvent{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
			Notional: 80, Time: now.Add(-time.Duration(i*6) * time.Minute),
		})
	}
	for i := 0; i < 6; i++ {
		book.Record(domain.LiquidationEvent{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
			Notional: 2500, Time: now.Add(-time.Duration(3+i) * time.Second),
		})
	}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	got, err := svc.GetLiquidationCascade(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var bn, bb domain.CascadeVenue
	for _, v := range got.Venues {
		if v.Exchange == domain.ExchangeBinance {
			bn = v
		}
		if v.Exchange == domain.ExchangeBybit {
			bb = v
		}
	}
	if domain.CascadeGradeRank(bn.Grade) < domain.CascadeGradeRank(domain.CascadeGradeElevated) {
		t.Fatalf("binance %+v", bn)
	}
	if bb.Grade != domain.CascadeGradeQuiet {
		t.Fatalf("bybit %+v", bb)
	}
}

func TestGetLiquidationCascade_EpisodesAndPrice(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	book := domain.NewLiquidationBook()
	for i := 1; i <= 80; i++ {
		book.Record(domain.LiquidationEvent{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
			Price: 64000, Notional: 80, Time: now.Add(-time.Duration(i*5) * time.Minute),
		})
	}
	start := now.Add(-40 * time.Minute)
	for m := 0; m < 8; m++ {
		t0 := start.Add(time.Duration(m) * time.Minute)
		for i := 0; i < 6; i++ {
			book.Record(domain.LiquidationEvent{
				Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
				Price: 64000 - float64(m)*20, Notional: 2000, Time: t0.Add(time.Duration(8+i) * time.Second),
			})
		}
	}
	bars := []domain.Candle{
		{OpenTime: start, Open: "64100", High: "64100", Low: "63600", Close: "63650", CloseTime: start.Add(time.Minute)},
		{OpenTime: start.Add(7 * time.Minute), Open: "63650", High: "63650", Low: "62800", Close: "62880", CloseTime: start.Add(8 * time.Minute)},
	}
	svc := New(&fakeMarket{candles: bars}, &fakeSupply{}).WithLiquidations(book, nil)
	got, err := svc.GetLiquidationCascade(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Episodes) == 0 {
		t.Fatalf("no episodes %+v", got)
	}
	ep := got.Episodes[0]
	if ep.Side != domain.LiquidationSideLong || ep.Open {
		t.Fatalf("episode %+v", ep)
	}
	if ep.PriceOpen != "64100" || ep.PriceClose != "62880" {
		t.Fatalf("candle price %+v", ep)
	}
}

func TestScanLiquidationCascades_Market(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	got, err := svc.ScanLiquidationCascades(context.Background(), "all")
	if err != nil || got.Market.Symbol != "all" || got.Hits == nil {
		t.Fatalf("%+v %v", got, err)
	}
}
