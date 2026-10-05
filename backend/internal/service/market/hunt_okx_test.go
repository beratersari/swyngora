package market

import (
	"context"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetLiquidationHunt_OKXUsesSwapMark(t *testing.T) {
	now := time.Now().UTC()
	bars := []domain.Candle{{
		OpenTime: now.Add(-time.Hour), CloseTime: now, Close: "64000", High: "64100", Low: "63900",
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{}).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeOKX: huntOISeries{points: []domain.OpenInterestPoint{{
			Time: now, Value: 2_000_000,
		}}},
	}).WithFuturesCandles(map[domain.Exchange]domain.FuturesCandlePort{
		domain.ExchangeOKX: huntCandles{bars: bars},
	})
	got, err := svc.GetLiquidationHunt(context.Background(), "okx", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 || got.Venues[0].Exchange != domain.ExchangeOKX {
		t.Fatalf("venues %+v", got.Venues)
	}
	v := got.Venues[0]
	if v.Error != "" || v.Price < 63900 || v.Price > 64100 {
		t.Fatalf("okx price %+v", v)
	}
	if v.EstShortNotional <= 0 || len(v.UpPressure) == 0 {
		t.Fatalf("bands %+v", v)
	}
}

func TestGetLiquidationHunt_OKXWithoutMarkStaysOut(t *testing.T) {
	now := time.Now().UTC()
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{}).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeOKX: huntOISeries{points: []domain.OpenInterestPoint{{
			Time: now, Value: 2_000_000,
		}}},
	})
	got, err := svc.GetLiquidationHunt(context.Background(), "okx", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 {
		t.Fatalf("venues %+v", got.Venues)
	}
	v := got.Venues[0]
	if v.Error == "" || strings.Contains(v.Error, "order book") {
		t.Fatalf("error %q", v.Error)
	}
	if v.Price != 0 {
		t.Fatalf("price %v", v.Price)
	}
}
