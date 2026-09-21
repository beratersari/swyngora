package market

import (
	"context"
	"errors"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetPerpDepth_BidHeavyOnePercent(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetPerpDepth(context.Background(), "binance", "BTCUSDT", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	h := got.Venues[0].Headline
	if h.Below.Notional < 39_000_000 || h.Above.Notional < 14_000_000 || h.Lean != domain.PerpDepthLeanBid {
		t.Fatalf("headline %+v", h)
	}
}

func TestGetPerpDepth_Combined(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetPerpDepth(context.Background(), "all", "BTCUSDT", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined == nil || len(got.Venues) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Combined.Headline.Below.Notional < 70_000_000 {
		t.Fatalf("combined %+v", got.Combined.Headline)
	}
}

func TestGetPerpDepth_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetPerpDepth(context.Background(), "all", "", 1); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
