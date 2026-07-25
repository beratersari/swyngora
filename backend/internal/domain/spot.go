package domain

// SpotSortField is a metric or identity field used to order spot markets.
type SpotSortField string

const (
	SpotSortQuoteVolume           SpotSortField = "quoteVolume"
	SpotSortVolume                SpotSortField = "volume"
	SpotSortPriceChangePercent    SpotSortField = "priceChangePercent"
	SpotSortLastPrice             SpotSortField = "lastPrice"
	SpotSortTradeCount            SpotSortField = "tradeCount"
	SpotSortSymbol                SpotSortField = "symbol"
	SpotSortBaseAsset             SpotSortField = "baseAsset"
	SpotSortMarketCapCirculating  SpotSortField = "marketCapCirculating"
	SpotSortMarketCapTotal        SpotSortField = "marketCapTotal"
	SpotSortMarketCapMax          SpotSortField = "marketCapMax"
)

// SupportedSpotSortFields lists valid sort query values.
var SupportedSpotSortFields = []SpotSortField{
	SpotSortQuoteVolume,
	SpotSortVolume,
	SpotSortPriceChangePercent,
	SpotSortLastPrice,
	SpotSortTradeCount,
	SpotSortSymbol,
	SpotSortBaseAsset,
	SpotSortMarketCapCirculating,
	SpotSortMarketCapTotal,
	SpotSortMarketCapMax,
}

// IsValidSpotSortField reports whether s is a supported sort field.
func IsValidSpotSortField(s string) bool {
	for _, f := range SupportedSpotSortFields {
		if string(f) == s {
			return true
		}
	}
	return false
}

// SortOrder is ascending or descending.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// IsValidSortOrder reports whether s is asc or desc.
func IsValidSortOrder(s string) bool {
	return s == string(SortAsc) || s == string(SortDesc)
}

// SpotMarket is a Binance spot trading pair with optional 24h metrics and mcap.
//
// BaseAsset / QuoteAsset / Status are retained for server-side filtering and
// enrichment; they are not always exposed on the public list DTO.
type SpotMarket struct {
	Symbol             string
	BaseAsset          string
	QuoteAsset         string
	Status             string
	LastPrice          string
	PriceChange        string
	PriceChangePercent string
	HighPrice          string
	LowPrice           string
	// Volume is base-asset 24h volume.
	Volume string
	// QuoteVolume is quote-asset 24h volume (primary liquidity metric).
	QuoteVolume string
	TradeCount  int64

	// Supply snapshot (from free CoinGecko metadata).
	CirculatingSupply *float64
	TotalSupply       *float64
	MaxSupply         *float64

	// Market caps = USD price × supply (see service pricing rules).
	MarketCapCirculating *float64
	MarketCapTotal       *float64
	// MarketCapMax is set when MaxSupply is known. When MaxSupply is unknown,
	// MarketCapMaxInfinite is true and MarketCapMax is nil.
	MarketCapMax         *float64
	MarketCapMaxInfinite bool
}

// SpotListQuery filters, sorts, and pages the spot market list.
type SpotListQuery struct {
	// Query is a case-insensitive substring match on symbol, base, or quote.
	Query string
	// QuoteAsset filters by quote (e.g. USDT). Empty = all quotes.
	QuoteAsset string
	// BaseAsset filters by base (e.g. BTC). Empty = all bases.
	BaseAsset string
	// Status filters exchange status (e.g. TRADING). Empty = all statuses.
	Status string
	// SortBy defaults to quoteVolume.
	SortBy SpotSortField
	// Order defaults to desc for metrics, asc for symbol/baseAsset when unset by service.
	Order SortOrder
	// Limit is page size (service applies defaults/max).
	Limit int
	// Offset is the number of matching rows to skip after filter+sort.
	Offset int
}

// SpotListResult is a page of spot markets plus total match count.
type SpotListResult struct {
	Items  []SpotMarket
	Total  int
	Limit  int
	Offset int
	SortBy SpotSortField
	Order  SortOrder
	Query  string
}

// NeedsSupplyEnrichment reports whether sort requires market-cap fields.
func (f SpotSortField) NeedsSupplyEnrichment() bool {
	switch f {
	case SpotSortMarketCapCirculating, SpotSortMarketCapTotal, SpotSortMarketCapMax:
		return true
	default:
		return false
	}
}
