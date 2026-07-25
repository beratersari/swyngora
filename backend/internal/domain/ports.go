package domain

import "context"

// MarketDataPort fetches exchange market data (candles, 24h stats).
// Implemented by the Binance adapter.
type MarketDataPort interface {
	GetCandles(ctx context.Context, q CandleQuery) ([]Candle, error)
	GetTicker24h(ctx context.Context, symbol string) (*Ticker24h, error)
}

// SupplyPort fetches asset supply metadata from a free public source.
type SupplyPort interface {
	GetSupply(ctx context.Context, asset string) (*AssetSupply, error)
}
