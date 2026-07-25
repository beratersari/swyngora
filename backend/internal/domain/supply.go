package domain

import "time"

// AssetSupply holds circulating / total / max supply for an asset.
//
// Note: Binance public market APIs do not expose supply metrics. This data is
// sourced from a free public metadata provider (CoinGecko) and is informational.
type AssetSupply struct {
	// Asset is the base asset ticker, uppercased (e.g. "BTC").
	Asset string
	// Name is a human-readable name when available (e.g. "Bitcoin").
	Name string
	// ProviderID is the upstream identifier used to fetch the record.
	ProviderID string
	// CirculatingSupply is coins currently in circulation (nullable when unknown).
	CirculatingSupply *float64
	// TotalSupply is total existing supply including locked/unreleased if known.
	TotalSupply *float64
	// MaxSupply is the hard cap if one exists (nullable, e.g. ETH has no max).
	MaxSupply *float64
	// CurrentPriceUSD is the latest USD price when the provider returns it.
	CurrentPriceUSD *float64
	// AsOf is when this snapshot was retrieved or produced.
	AsOf time.Time
	// Source identifies the data provider (e.g. "coingecko").
	Source string
}
