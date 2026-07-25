package domain

import (
	"context"
	"testing"
	"time"
)

type stubMarketData struct{}

func (stubMarketData) GetCandles(context.Context, CandleQuery) ([]Candle, error) {
	return nil, nil
}

func (stubMarketData) GetTicker24h(context.Context, string) (*Ticker24h, error) {
	return nil, nil
}

func (stubMarketData) ListSpotMarkets(context.Context) ([]SpotMarket, error) {
	return nil, nil
}

type stubSupplyPort struct{}

func (stubSupplyPort) GetSupply(context.Context, string) (*AssetSupply, error) {
	return nil, nil
}

func (stubSupplyPort) Refresh(context.Context) (int, error) {
	return 0, nil
}

func TestMarketDataPort_Assignable(t *testing.T) {
	var _ MarketDataPort = stubMarketData{}
	var p MarketDataPort = stubMarketData{}
	if _, err := p.GetCandles(context.Background(), CandleQuery{
		Symbol: "BTCUSDT", Interval: Interval1h, Limit: 1,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSupplyPort_Assignable(t *testing.T) {
	var _ SupplyPort = stubSupplyPort{}
	var p SupplyPort = stubSupplyPort{}
	if _, err := p.GetSupply(context.Background(), "BTC"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCandleQuery_ZeroValues(t *testing.T) {
	q := CandleQuery{
		Symbol: "ETHUSDT", Interval: Interval15m, Limit: 10,
		StartTime: time.Unix(1, 0).UTC(), EndTime: time.Unix(2, 0).UTC(),
	}
	if q.Symbol != "ETHUSDT" {
		t.Fatalf("query=%+v", q)
	}
}
