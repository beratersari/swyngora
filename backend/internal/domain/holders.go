package domain

import "time"

// MaxHolderList is the maximum number of top wallets returned to clients.
const MaxHolderList = 20

// AssetCatalogEntry maps a base ticker to a CoinMarketCap id from the Binance
// marketing symbol list. Used to fetch holder snapshots without a live CMC search.
type AssetCatalogEntry struct {
	Asset string
	Name  string
	// CMCID is CoinMarketCap's numeric id (e.g. 1 for Bitcoin).
	CMCID int64
	Slug  string
}

// AssetHolder is one wallet in a top-holder snapshot.
type AssetHolder struct {
	Address string
	// Label is a public attribution when the address is widely published
	// (exchange cold wallet, genesis, seized hack). Empty when unknown.
	// Not identity proof.
	Label    string
	Balance  float64
	SharePct float64
}

// AssetHolders is an on-chain holder snapshot for a crypto base asset.
// Pair forms (BTCUSDT) resolve to the base (BTC). Equities have no mapping.
type AssetHolders struct {
	Asset              string
	Name               string
	ProviderID         string
	HolderCount        int64
	DailyActive        *int64
	TopTenSharePct     *float64
	TopTwentySharePct  *float64
	TopFiftySharePct   *float64
	TopHundredSharePct *float64
	TopHolders         []AssetHolder
	AsOf               time.Time
	Source             string
	// Stale is true when serving last-good after a CMC 429/upstream/empty blip.
	Stale bool
}

// CloneHolders copies a snapshot so cached values are not mutated by callers.
func CloneHolders(in *AssetHolders) *AssetHolders {
	if in == nil {
		return nil
	}
	out := *in
	out.DailyActive = CloneInt64Ptr(in.DailyActive)
	out.TopTenSharePct = CloneFloatPtr(in.TopTenSharePct)
	out.TopTwentySharePct = CloneFloatPtr(in.TopTwentySharePct)
	out.TopFiftySharePct = CloneFloatPtr(in.TopFiftySharePct)
	out.TopHundredSharePct = CloneFloatPtr(in.TopHundredSharePct)
	if in.TopHolders != nil {
		out.TopHolders = make([]AssetHolder, len(in.TopHolders))
		copy(out.TopHolders, in.TopHolders)
	}
	return &out
}

// CloneInt64Ptr returns a copy of the pointed-to int64 (or nil).
func CloneInt64Ptr(p *int64) *int64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// CapHolderList returns at most n wallets (n<=0 uses MaxHolderList).
func CapHolderList(list []AssetHolder, n int) []AssetHolder {
	if n <= 0 {
		n = MaxHolderList
	}
	if len(list) <= n {
		return list
	}
	return list[:n]
}
