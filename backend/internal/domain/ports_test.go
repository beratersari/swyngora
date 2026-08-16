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

func (stubMarketData) GetOrderBook(context.Context, OrderBookQuery) (*RawOrderBook, error) {
	return &RawOrderBook{}, nil
}

func (stubMarketData) ListSpotMarkets(context.Context) ([]SpotMarket, error) {
	return nil, nil
}

func (stubMarketData) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func (stubMarketData) ListProductTags(context.Context) ([]string, error) {
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

type stubCatalogPort struct{}

func (stubCatalogPort) LookupAsset(context.Context, string) (*AssetCatalogEntry, error) {
	return &AssetCatalogEntry{Asset: "BTC", CMCID: 1}, nil
}

type stubHoldersPort struct{}

func (stubHoldersPort) GetHolders(context.Context, string) (*AssetHolders, error) {
	return &AssetHolders{Asset: "BTC", HolderCount: 1}, nil
}

func TestCatalogAndHoldersPorts_Assignable(t *testing.T) {
	var _ AssetCatalogPort = stubCatalogPort{}
	var _ HoldersPort = stubHoldersPort{}
	cat, err := stubCatalogPort{}.LookupAsset(context.Background(), "BTC")
	if err != nil || cat.CMCID != 1 {
		t.Fatalf("catalog=%+v err=%v", cat, err)
	}
	h, err := stubHoldersPort{}.GetHolders(context.Background(), "BTC")
	if err != nil || h.HolderCount != 1 {
		t.Fatalf("holders=%+v err=%v", h, err)
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
