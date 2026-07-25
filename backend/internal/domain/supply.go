package domain

import "time"

// AssetSupply holds circulating / total / max supply for an asset.
//
// Loaded from Binance's public marketing symbol list (circulatingSupply, totalSupply,
// maxSupply). Max remains nil when Binance does not define a hard cap.
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
	// Source identifies the data provider (e.g. "binance").
	Source string
}

// CloneFloatPtr returns a copy of the pointed-to float64 (or nil). This is used
// to ensure that values returned from caches are not shared pointers, preventing
// accidental mutation of cached data from affecting other callers or the cache itself.
func CloneFloatPtr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
