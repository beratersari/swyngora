package domain

import "time"

// SpotSortField is a metric or identity field used to order spot markets.
type SpotSortField string

const (
	SpotSortQuoteVolume          SpotSortField = "quoteVolume"
	SpotSortVolume               SpotSortField = "volume"
	SpotSortPriceChangePercent   SpotSortField = "priceChangePercent"
	SpotSortLastPrice            SpotSortField = "lastPrice"
	SpotSortTradeCount           SpotSortField = "tradeCount"
	SpotSortSymbol               SpotSortField = "symbol"
	SpotSortBaseAsset            SpotSortField = "baseAsset"
	SpotSortMarketCapCirculating SpotSortField = "marketCapCirculating"
	SpotSortMarketCapTotal       SpotSortField = "marketCapTotal"
	SpotSortMarketCapMax         SpotSortField = "marketCapMax"
	// SpotSortTags orders by primary tag (lexicographic join of tags); empty tags last.
	SpotSortTags SpotSortField = "tags"
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
	SpotSortTags,
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

// SpotMarket is a venue-agnostic spot trading pair with optional 24h metrics and mcap.
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

	// Tags are product-catalog labels for the base asset
	// (e.g. "Meme", "Layer1_Layer2", "defi"), sourced from Binance catalog
	// and applied cross-venue by base when other venues lack tags.
	// Empty when unknown.
	Tags []string

	// Supply snapshot (from Binance product catalog circulating supply).
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

	// DelistTime is set when the pair is on the venue's scheduled delist list.
	DelistTime *time.Time
	// DelistAnnouncedAt is when the venue published that delist notice.
	DelistAnnouncedAt *time.Time
}

// ApplyTickerToSpot copies tape fields onto a spot row. Empty dest fields only
// (does not overwrite a live book print).
func ApplyTickerToSpot(m *SpotMarket, t Ticker24h) {
	if m == nil {
		return
	}
	if m.LastPrice == "" && t.LastPrice != "" {
		m.LastPrice = t.LastPrice
	}
	if m.PriceChange == "" && t.PriceChange != "" {
		m.PriceChange = t.PriceChange
	}
	if m.PriceChangePercent == "" && t.PriceChangePercent != "" {
		m.PriceChangePercent = t.PriceChangePercent
	}
	if m.HighPrice == "" && t.HighPrice != "" {
		m.HighPrice = t.HighPrice
	}
	if m.LowPrice == "" && t.LowPrice != "" {
		m.LowPrice = t.LowPrice
	}
	if m.Volume == "" && t.Volume != "" {
		m.Volume = t.Volume
	}
	if m.QuoteVolume == "" && t.QuoteVolume != "" {
		m.QuoteVolume = t.QuoteVolume
	}
	if m.TradeCount == 0 && t.TradeCount != 0 {
		m.TradeCount = t.TradeCount
	}
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
	// Tags filters markets that have at least one of these product tags
	// (case-insensitive; OR semantics). Empty = no tag filter.
	Tags []string
	// SortBy defaults to quoteVolume.
	SortBy SpotSortField
	// Order defaults to desc for metrics, asc for symbol/baseAsset/tags when unset by service.
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
	// Tags is the normalized tag filter applied (may be empty).
	Tags     []string
	Exchange Exchange
}

// NeedsSupplyEnrichment is true when sort order depends on mcap fields before pagination
// (full filtered set must be enriched). Display enrichment of a single page is separate.
func (f SpotSortField) NeedsSupplyEnrichment() bool {
	switch f {
	case SpotSortMarketCapCirculating, SpotSortMarketCapTotal, SpotSortMarketCapMax:
		return true
	default:
		return false
	}
}
