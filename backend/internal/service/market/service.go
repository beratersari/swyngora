package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Service orchestrates market-data use cases. Handlers call this layer only.
type Service struct {
	market domain.MarketDataPort
	supply domain.SupplyPort
}

// New constructs a market application service.
func New(market domain.MarketDataPort, supply domain.SupplyPort) *Service {
	return &Service{market: market, supply: supply}
}

// GetCandles validates and fetches OHLCV candles for a Binance-style symbol.
func (s *Service) GetCandles(ctx context.Context, symbol, interval string, limit int, start, end *time.Time) ([]domain.Candle, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if !domain.IsValidInterval(interval) {
		return nil, fmt.Errorf("%w: interval must be one of %v", domain.ErrInvalidArgument, domain.SupportedIntervals)
	}
	if limit < 0 {
		return nil, fmt.Errorf("%w: limit must be >= 0", domain.ErrInvalidArgument)
	}
	if limit == 0 {
		limit = 100
	}
	if limit > 1000 {
		return nil, fmt.Errorf("%w: limit must be <= 1000", domain.ErrInvalidArgument)
	}

	q := domain.CandleQuery{
		Symbol:   symbol,
		Interval: domain.CandleInterval(interval),
		Limit:    limit,
	}
	if start != nil {
		q.StartTime = start.UTC()
	}
	if end != nil {
		q.EndTime = end.UTC()
	}
	if !q.StartTime.IsZero() && !q.EndTime.IsZero() && q.EndTime.Before(q.StartTime) {
		return nil, fmt.Errorf("%w: endTime must be >= startTime", domain.ErrInvalidArgument)
	}

	return s.market.GetCandles(ctx, q)
}

// GetTicker24h returns rolling 24h volume and price stats for a symbol.
func (s *Service) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return s.market.GetTicker24h(ctx, symbol)
}

// GetSupply returns circulating / total / max supply for a base asset (or pair).
func (s *Service) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	return s.supply.GetSupply(ctx, asset)
}

// ListIntervals returns supported candle intervals.
func (s *Service) ListIntervals() []domain.CandleInterval {
	out := make([]domain.CandleInterval, len(domain.SupportedIntervals))
	copy(out, domain.SupportedIntervals)
	return out
}

const (
	defaultSpotLimit = 50
	maxSpotLimit     = 500
)

// ListSpotMarkets lists Binance spot pairs with search, metric sort, and pagination.
// Market-cap fields are enriched from CoinGecko supply (best-effort; missing assets stay null).
func (s *Service) ListSpotMarkets(ctx context.Context, q domain.SpotListQuery) (*domain.SpotListResult, error) {
	q, err := normalizeSpotListQuery(q)
	if err != nil {
		return nil, err
	}

	all, err := s.market.ListSpotMarkets(ctx)
	if err != nil {
		return nil, err
	}

	filtered := filterSpotMarkets(all, q)

	// Mcap sorts need supply on the full filtered set before ordering.
	if q.SortBy.NeedsSupplyEnrichment() {
		s.enrichSpotMarkets(ctx, filtered)
		sortSpotMarkets(filtered, q.SortBy, q.Order)
		total := len(filtered)
		page := pageSpotMarkets(filtered, q.Offset, q.Limit)
		return &domain.SpotListResult{
			Items: page, Total: total, Limit: q.Limit, Offset: q.Offset,
			SortBy: q.SortBy, Order: q.Order, Query: q.Query,
		}, nil
	}

	sortSpotMarkets(filtered, q.SortBy, q.Order)
	total := len(filtered)
	page := pageSpotMarkets(filtered, q.Offset, q.Limit)
	s.enrichSpotMarkets(ctx, page)

	return &domain.SpotListResult{
		Items:  page,
		Total:  total,
		Limit:  q.Limit,
		Offset: q.Offset,
		SortBy: q.SortBy,
		Order:  q.Order,
		Query:  q.Query,
	}, nil
}

// enrichSpotMarkets fills supply + market-cap from the daily supply cache only (no live fetch).
func (s *Service) enrichSpotMarkets(ctx context.Context, items []domain.SpotMarket) {
	if len(items) == 0 || s.supply == nil {
		return
	}
	byAsset := map[string]*domain.AssetSupply{}
	for i := range items {
		b := strings.ToUpper(strings.TrimSpace(items[i].BaseAsset))
		if b == "" {
			continue
		}
		if _, ok := byAsset[b]; ok {
			continue
		}
		sup, err := s.supply.GetSupply(ctx, b)
		if err != nil || sup == nil {
			byAsset[b] = nil
			continue
		}
		byAsset[b] = sup
	}
	for i := range items {
		applySupplyAndMcap(&items[i], byAsset[strings.ToUpper(items[i].BaseAsset)])
	}
}

func applySupplyAndMcap(m *domain.SpotMarket, sup *domain.AssetSupply) {
	if m == nil || sup == nil {
		// Unknown max supply still shows infinity when we have no metadata? User asked
		// for infinity when max is not defined — only after we know supply is missing max.
		return
	}
	m.CirculatingSupply = sup.CirculatingSupply
	m.TotalSupply = sup.TotalSupply
	m.MaxSupply = sup.MaxSupply

	price := usdPriceForMcap(*m, sup)
	if price == nil {
		// Still surface max infinity if max supply is undefined.
		if sup.MaxSupply == nil {
			m.MarketCapMaxInfinite = true
		}
		return
	}
	p := *price
	if sup.CirculatingSupply != nil {
		v := p * *sup.CirculatingSupply
		m.MarketCapCirculating = &v
	}
	if sup.TotalSupply != nil {
		v := p * *sup.TotalSupply
		m.MarketCapTotal = &v
	}
	if sup.MaxSupply == nil {
		m.MarketCapMax = nil
		m.MarketCapMaxInfinite = true
	} else {
		v := p * *sup.MaxSupply
		m.MarketCapMax = &v
		m.MarketCapMaxInfinite = false
	}
}

// usdPriceForMcap prefers Binance last price when the quote is a USD stablecoin;
// otherwise uses CoinGecko USD price when available.
func usdPriceForMcap(m domain.SpotMarket, sup *domain.AssetSupply) *float64 {
	q := strings.ToUpper(m.QuoteAsset)
	switch q {
	case "USDT", "USDC", "BUSD", "FDUSD", "TUSD", "USD", "DAI":
		if px, err := parseFloat(m.LastPrice); err == nil && px > 0 {
			return &px
		}
	}
	if sup != nil && sup.CurrentPriceUSD != nil && *sup.CurrentPriceUSD > 0 {
		v := *sup.CurrentPriceUSD
		return &v
	}
	return nil
}

func normalizeSpotListQuery(q domain.SpotListQuery) (domain.SpotListQuery, error) {
	q.Query = strings.TrimSpace(q.Query)
	q.QuoteAsset = strings.ToUpper(strings.TrimSpace(q.QuoteAsset))
	q.BaseAsset = strings.ToUpper(strings.TrimSpace(q.BaseAsset))
	q.Status = strings.ToUpper(strings.TrimSpace(q.Status))

	if q.SortBy == "" {
		q.SortBy = domain.SpotSortQuoteVolume
	}
	if !domain.IsValidSpotSortField(string(q.SortBy)) {
		return q, fmt.Errorf("%w: sort must be one of %v", domain.ErrInvalidArgument, domain.SupportedSpotSortFields)
	}

	if q.Order == "" {
		switch q.SortBy {
		case domain.SpotSortSymbol, domain.SpotSortBaseAsset:
			q.Order = domain.SortAsc
		default:
			q.Order = domain.SortDesc
		}
	}
	order := strings.ToLower(string(q.Order))
	if !domain.IsValidSortOrder(order) {
		return q, fmt.Errorf("%w: order must be asc or desc", domain.ErrInvalidArgument)
	}
	q.Order = domain.SortOrder(order)

	if q.Limit < 0 {
		return q, fmt.Errorf("%w: limit must be >= 0", domain.ErrInvalidArgument)
	}
	if q.Limit == 0 {
		q.Limit = defaultSpotLimit
	}
	if q.Limit > maxSpotLimit {
		return q, fmt.Errorf("%w: limit must be <= %d", domain.ErrInvalidArgument, maxSpotLimit)
	}
	if q.Offset < 0 {
		return q, fmt.Errorf("%w: offset must be >= 0", domain.ErrInvalidArgument)
	}
	return q, nil
}

func filterSpotMarkets(all []domain.SpotMarket, q domain.SpotListQuery) []domain.SpotMarket {
	out := make([]domain.SpotMarket, 0, len(all))
	needle := strings.ToUpper(q.Query)
	for _, m := range all {
		if q.QuoteAsset != "" && !strings.EqualFold(m.QuoteAsset, q.QuoteAsset) {
			continue
		}
		if q.BaseAsset != "" && !strings.EqualFold(m.BaseAsset, q.BaseAsset) {
			continue
		}
		if q.Status != "" && !strings.EqualFold(m.Status, q.Status) {
			continue
		}
		if needle != "" {
			if !strings.Contains(strings.ToUpper(m.Symbol), needle) &&
				!strings.Contains(strings.ToUpper(m.BaseAsset), needle) &&
				!strings.Contains(strings.ToUpper(m.QuoteAsset), needle) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

func sortSpotMarkets(items []domain.SpotMarket, by domain.SpotSortField, order domain.SortOrder) {
	desc := order == domain.SortDesc
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		var cmp int
		switch by {
		case domain.SpotSortSymbol:
			cmp = strings.Compare(a.Symbol, b.Symbol)
		case domain.SpotSortBaseAsset:
			cmp = strings.Compare(a.BaseAsset, b.BaseAsset)
			if cmp == 0 {
				cmp = strings.Compare(a.Symbol, b.Symbol)
			}
		case domain.SpotSortTradeCount:
			cmp = cmpInt64(a.TradeCount, b.TradeCount)
		case domain.SpotSortVolume:
			cmp = cmpFloatString(a.Volume, b.Volume)
		case domain.SpotSortPriceChangePercent:
			cmp = cmpFloatString(a.PriceChangePercent, b.PriceChangePercent)
		case domain.SpotSortLastPrice:
			cmp = cmpFloatString(a.LastPrice, b.LastPrice)
		case domain.SpotSortMarketCapCirculating:
			cmp = cmpOptionalFloat(a.MarketCapCirculating, b.MarketCapCirculating)
		case domain.SpotSortMarketCapTotal:
			cmp = cmpOptionalFloat(a.MarketCapTotal, b.MarketCapTotal)
		case domain.SpotSortMarketCapMax:
			cmp = cmpMcapMax(a, b)
		default: // quoteVolume
			cmp = cmpFloatString(a.QuoteVolume, b.QuoteVolume)
		}
		if cmp == 0 {
			// Always break ties by symbol ascending for stable pages.
			return a.Symbol < b.Symbol
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func pageSpotMarkets(items []domain.SpotMarket, offset, limit int) []domain.SpotMarket {
	if offset >= len(items) {
		return []domain.SpotMarket{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]domain.SpotMarket(nil), items[offset:end]...)
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpFloatString(a, b string) int {
	fa, _ := parseFloat(a)
	fb, _ := parseFloat(b)
	switch {
	case fa < fb:
		return -1
	case fa > fb:
		return 1
	default:
		return 0
	}
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

func cmpOptionalFloat(a, b *float64) int {
	fa, fb := 0.0, 0.0
	if a != nil {
		fa = *a
	}
	if b != nil {
		fb = *b
	}
	switch {
	case fa < fb:
		return -1
	case fa > fb:
		return 1
	default:
		return 0
	}
}

// cmpMcapMax treats infinite max mcap as larger than any finite value.
func cmpMcapMax(a, b domain.SpotMarket) int {
	rank := func(m domain.SpotMarket) float64 {
		if m.MarketCapMaxInfinite {
			return 1e300 // sort as top when desc
		}
		if m.MarketCapMax != nil {
			return *m.MarketCapMax
		}
		return 0
	}
	fa, fb := rank(a), rank(b)
	switch {
	case fa < fb:
		return -1
	case fa > fb:
		return 1
	default:
		return 0
	}
}
