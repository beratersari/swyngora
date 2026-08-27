package domain

import (
	"context"
	"time"
)

// MarketDataPort fetches exchange market data (candles, 24h stats, spot listings).
// Implemented by venue adapters (Binance, Coinbase, Bybit).
type MarketDataPort interface {
	GetCandles(ctx context.Context, q CandleQuery) ([]Candle, error)
	GetTicker24h(ctx context.Context, symbol string) (*Ticker24h, error)
	// GetOrderBook returns a raw spot bid/ask snapshot (ungrouped).
	GetOrderBook(ctx context.Context, q OrderBookQuery) (*RawOrderBook, error)
	// ListSpotMarkets returns all spot-tradable pairs joined with 24h metrics.
	ListSpotMarkets(ctx context.Context) ([]SpotMarket, error)
	// ListProductTags returns unique product-catalog tags for crypto spot bases
	// (sorted), used for UI filters. Empty when the venue has no catalog.
	ListProductTags(ctx context.Context) ([]string, error)
	// TagsByBase returns product-catalog tags keyed by uppercased base asset.
	// Used to enrich Coinbase/Bybit spot rows with Binance marketing tags.
	// Empty map when the venue has no catalog (never an error for "unsupported").
	TagsByBase(ctx context.Context) (map[string][]string, error)
}

// SupplyPort serves asset supply / mcap inputs for user requests.
// Implementations MUST serve GetSupply from cache only (no live upstream on request path).
// Refresh populates the cache (e.g. daily Binance product-catalog snapshot).
type SupplyPort interface {
	GetSupply(ctx context.Context, asset string) (*AssetSupply, error)
	// Refresh loads a bulk snapshot into cache. Returns number of assets stored.
	Refresh(ctx context.Context) (int, error)
}

// SymbolSupplyFallback looks up circulating/total/max supply by ticker when the
// Binance marketing snapshot has no row (typical for already-delisted alts).
type SymbolSupplyFallback interface {
	SupplyBySymbols(ctx context.Context, symbols []string) (map[string]*AssetSupply, error)
}

// AssetCatalogPort resolves a base ticker (or pair) to a CoinMarketCap id.
// Implementations MUST serve from the Binance marketing snapshot (cache-only).
type AssetCatalogPort interface {
	LookupAsset(ctx context.Context, asset string) (*AssetCatalogEntry, error)
}

// HoldersPort serves on-chain holder snapshots for crypto assets.
// Implementations may fetch upstream on a cache miss; user requests still go
// through this port (never from the HTTP handler).
type HoldersPort interface {
	GetHolders(ctx context.Context, asset string) (*AssetHolders, error)
}

// TokenContractPort resolves published token addresses for a base ticker.
// Used by holder fallbacks when CoinMarketCap has no table.
type TokenContractPort interface {
	LookupContracts(ctx context.Context, asset string) ([]AssetContract, error)
}

// AssetProfilePort serves logo, listing date, and published contracts.
type AssetProfilePort interface {
	GetAssetProfile(ctx context.Context, asset string) (*AssetProfile, error)
}

// OpenInterestPort loads current + historical futures open interest for one venue.
// Implemented by Binance USD-M and Bybit linear adapters.
type OpenInterestPort interface {
	GetOpenInterestSeries(ctx context.Context, symbol string) (*OpenInterestSeries, error)
}

// FundingRatePort loads the predicted next funding rate plus settled history.
// Implemented by Binance USD-M and Bybit linear adapters.
type FundingRatePort interface {
	GetFundingSeries(ctx context.Context, symbol string, limit int) (*FundingSeries, error)
	// ListFundingHistory returns settled funding prints in [from, to] (UTC), oldest first.
	ListFundingHistory(ctx context.Context, symbol string, from, to time.Time) ([]FundingPoint, error)
}

// LongShortRatioPort loads the latest account long/short ratio plus recent history.
// Implemented by Binance USD-M and Bybit linear adapters.
type LongShortRatioPort interface {
	GetLongShortSeries(ctx context.Context, symbol string, limit int) (*LongShortSeries, error)
}
