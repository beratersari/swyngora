package domain

import (
	"strconv"
	"strings"
	"time"
)

// AssetContract is a published on-chain token address for an asset.
type AssetContract struct {
	Chain   string
	Address string
}

// InferContractChain fills a missing chain from the address shape
// (0x… → ethereum, long base58 → solana).
func InferContractChain(chain, address string) string {
	c := CanonicalChain(chain)
	if c != "" {
		return c
	}
	addr := strings.TrimSpace(address)
	if len(addr) >= 40 && strings.HasPrefix(strings.ToLower(addr), "0x") {
		return "ethereum"
	}
	// TRC-20 is base58 starting with T (≈34 chars). Do this before Solana.
	if strings.HasPrefix(addr, "T") && len(addr) >= 30 && len(addr) <= 36 {
		return "tron"
	}
	if len(addr) >= 32 && !strings.Contains(addr, "0x") {
		return "solana"
	}
	return ""
}

// CanonicalChain maps CMC/Gecko platform labels to a short chain id.
func CanonicalChain(chain string) string {
	s := strings.ToLower(strings.TrimSpace(chain))
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	switch {
	case s == "":
		return ""
	case strings.Contains(s, "chiliz"):
		return "chiliz"
	case strings.Contains(s, "tron"):
		return "tron"
	case strings.Contains(s, "zksync"):
		return "zksync"
	case strings.Contains(s, "scroll"):
		return "scroll"
	case strings.Contains(s, "kaia") || strings.Contains(s, "klay"):
		return "kaia"
	case strings.Contains(s, "manta"):
		return "manta"
	case s == "sonic" || strings.HasPrefix(s, "sonic "):
		return "sonic"
	case strings.Contains(s, "ronin"):
		return "ronin"
	case strings.Contains(s, "celo"):
		return "celo"
	case strings.Contains(s, "bep20") || strings.Contains(s, "bnb smart") || strings.Contains(s, "binance smart") || s == "bsc" || s == "bnb":
		return "bsc"
	case s == "base" || strings.HasPrefix(s, "base "):
		return "base"
	case strings.Contains(s, "optimism") || strings.Contains(s, "optimistic"):
		return "optimism"
	case strings.Contains(s, "arbitrum"):
		return "arbitrum"
	case strings.Contains(s, "polygon"):
		return "polygon"
	case strings.Contains(s, "solana"):
		return "solana"
	case strings.Contains(s, "avalanche") && !strings.Contains(s, "x chain"):
		return "avalanche"
	case s == "eth" || s == "ethereum" || strings.Contains(s, "erc20"):
		return "ethereum"
	default:
		return strings.Join(strings.Fields(s), "-")
	}
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
