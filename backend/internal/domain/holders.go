package domain

import (
	"math"
	"time"
)

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
	// ResolvedBalance prefers share × circulating when the reported CMC
	// balance is dust-scale compared with that estimate.
	ResolvedBalance *float64
	// UsdValue is share × circulating × USD price when those inputs exist.
	UsdValue *float64
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

// HoldersUseful is true when a snapshot has a count or at least one wallet.
func HoldersUseful(h *AssetHolders) bool {
	if h == nil {
		return false
	}
	return h.HolderCount > 0 || len(h.TopHolders) > 0
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

// ResolveHolderBalance prefers share × circulating when the reported wallet
// size is missing, zero, or at least 100× smaller than that estimate
// (common on high-supply tokens where CMC prints dust).
func ResolveHolderBalance(reported, sharePct float64, circulating *float64) *float64 {
	var raw *float64
	if !math.IsNaN(reported) && !math.IsInf(reported, 0) {
		v := reported
		raw = &v
	}
	var estimated *float64
	if circulating != nil && *circulating > 0 && !math.IsNaN(sharePct) && !math.IsInf(sharePct, 0) {
		v := (sharePct / 100) * *circulating
		estimated = &v
	}
	if estimated != nil && *estimated > 0 {
		if raw == nil || *raw == 0 {
			return estimated
		}
		den := *raw
		if den < 0 {
			den = -den
		}
		if den < 1e-18 {
			den = 1e-18
		}
		num := *estimated
		if num < 0 {
			num = -num
		}
		if num/den >= 100 {
			return estimated
		}
	}
	return raw
}

// HolderUsdValue is share of circulating market cap in USD.
func HolderUsdValue(sharePct float64, circulating, priceUSD *float64) *float64 {
	if circulating == nil || priceUSD == nil {
		return nil
	}
	if *circulating <= 0 || *priceUSD <= 0 || math.IsNaN(sharePct) || math.IsInf(sharePct, 0) {
		return nil
	}
	v := (sharePct / 100) * *circulating * *priceUSD
	return &v
}

// EnrichHolderRows fills ResolvedBalance and UsdValue on each wallet.
func EnrichHolderRows(h *AssetHolders, circulating, priceUSD *float64) {
	if h == nil {
		return
	}
	for i := range h.TopHolders {
		row := &h.TopHolders[i]
		row.ResolvedBalance = ResolveHolderBalance(row.Balance, row.SharePct, circulating)
		row.UsdValue = HolderUsdValue(row.SharePct, circulating, priceUSD)
	}
}
