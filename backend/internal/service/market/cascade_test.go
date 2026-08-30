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

func TestScanLiquidationCascades_Market(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	got, err := svc.ScanLiquidationCascades(context.Background(), "all")
	if err != nil || got.Market.Symbol != "all" || got.Hits == nil {
		t.Fatalf("%+v %v", got, err)
	}
}
