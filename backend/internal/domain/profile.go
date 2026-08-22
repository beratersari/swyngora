package domain

import (
	"strconv"
	"time"
)

// AssetContract is a published on-chain token address for an asset.
type AssetContract struct {
	Chain   string
	Address string
}

// AssetProfile is identity metadata (logo, listing date, contracts).
// Coverage follows the Binance marketing CMC id, not every venue listing.
type AssetProfile struct {
	Asset       string
	Name        string
	Slug        string
	ProviderID  string
	LogoURL     string
	ListingDate *time.Time
	Contracts   []AssetContract
	AsOf        time.Time
	Source      string
	Stale       bool
}

// CMCLogoURL is CoinMarketCap's public static coin icon for a numeric id.
func CMCLogoURL(cmcID int64) string {
	if cmcID <= 0 {
		return ""
	}
	return "https://s2.coinmarketcap.com/static/img/coins/64x64/" + strconv.FormatInt(cmcID, 10) + ".png"
}
