package domain

import (
	"context"
	"testing"
	"time"
)

// Compile-time / smoke checks that port interfaces stay assignable and
// constructible. No runtime I/O — these types are contracts only.

type stubMarketData struct{}

func (stubMarketData) GetCandles(context.Context, CandleQuery) ([]Candle, error) {
	return nil, nil
}

func (stubMarketData) GetTicker24h(context.Context, string) (*Ticker24h, error) {
	return nil, nil
}

type stubSupplyPort struct{}

func (stubSupplyPort) GetSupply(context.Context, string) (*AssetSupply, error) {
	return nil, nil
}

func TestMarketDataPort_Assignable(t *testing.T) {
	var _ MarketDataPort = stubMarketData{}
	var p MarketDataPort = stubMarketData{}
	if _, err := p.GetCandles(context.Background(), CandleQuery{
		Symbol:   "BTCUSDT",
		Interval: Interval1h,
		Limit:    1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetTicker24h(context.Background(), "BTCUSDT"); err != nil {
		t.Fatal(err)
	}
}

func TestSupplyPort_Assignable(t *testing.T) {
	var _ SupplyPort = stubSupplyPort{}
	var p SupplyPort = stubSupplyPort{}
	if _, err := p.GetSupply(context.Background(), "BTC"); err != nil {
		t.Fatal(err)
	}
}

func TestCandleQuery_ZeroValues(t *testing.T) {
	var q CandleQuery
	if !q.StartTime.IsZero() || !q.EndTime.IsZero() {
		t.Fatal("expected zero times")
	}
	q = CandleQuery{
		Symbol:    "ETHUSDT",
		Interval:  Interval15m,
		Limit:     10,
		StartTime: time.Unix(1, 0).UTC(),
		EndTime:   time.Unix(2, 0).UTC(),
	}
	if q.Symbol != "ETHUSDT" || q.Interval != Interval15m || q.Limit != 10 {
		t.Fatalf("query=%+v", q)
	}
}
