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
	markets map[domain.Exchange]domain.MarketDataPort
	supply  domain.SupplyPort
}

// New constructs a market application service with a single market data port
// (typically Binance). Prefer NewMulti when wiring multiple exchanges.
func New(market domain.MarketDataPort, supply domain.SupplyPort) *Service {
	return NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.DefaultExchange: market,
	}, supply)
}

// NewMulti constructs a service that routes market calls by exchange id.
func NewMulti(markets map[domain.Exchange]domain.MarketDataPort, supply domain.SupplyPort) *Service {
	cp := make(map[domain.Exchange]domain.MarketDataPort, len(markets))
	for k, v := range markets {
		if v != nil {
			cp[k] = v
		}
	}
	return &Service{markets: cp, supply: supply}
}

func (s *Service) port(ex domain.Exchange) (domain.MarketDataPort, error) {
	if ex == "" {
		ex = domain.DefaultExchange
	}
	p, ok := s.markets[ex]
	if !ok || p == nil {
		return nil, fmt.Errorf("%w: exchange %q is not configured", domain.ErrInvalidArgument, ex)
	}
	return p, nil
}

// ResolveExchange parses and validates an exchange query value.
func (s *Service) ResolveExchange(raw string) (domain.Exchange, error) {
	ex := domain.ParseExchange(raw)
	if ex == "" {
		return "", fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
	}
	if _, err := s.port(ex); err != nil {
		return "", err
	}
	return ex, nil
}

// ListExchanges returns configured venue ids in stable order.
func (s *Service) ListExchanges() []domain.Exchange {
	out := make([]domain.Exchange, 0, len(domain.SupportedExchanges))
	for _, e := range domain.SupportedExchanges {
		if _, ok := s.markets[e]; ok {
			out = append(out, e)
		}
	}
	return out
}

// GetCandles validates and fetches OHLCV candles for a trading pair on the given exchange.
func (s *Service) GetCandles(ctx context.Context, exchange, symbol, interval string, limit int, start, end *time.Time) ([]domain.Candle, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	// Coinbase uses hyphenated product ids (BTC-USD); preserve hyphen, upper case.
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if !domain.IsValidIntervalFor(ex, interval) {
		return nil, fmt.Errorf("%w: interval must be one of %v for %s", domain.ErrInvalidArgument, domain.SupportedIntervalsFor(ex), ex)
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

	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	return p.GetCandles(ctx, q)
}

// GetTicker24h returns rolling 24h volume and price stats for a symbol.
func (s *Service) GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	return p.GetTicker24h(ctx, symbol)
}

// GetSupply returns circulating / total / max supply for a base asset (or pair).
// Supply is always served from the Binance daily snapshot (asset-level, exchange-agnostic).
func (s *Service) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if s.supply == nil {
		return nil, fmt.Errorf("%w: supply port not configured", domain.ErrUpstream)
	}
	return s.supply.GetSupply(ctx, asset)
}

// ListIntervals returns supported candle intervals for an exchange.
func (s *Service) ListIntervals(exchange string) ([]domain.CandleInterval, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	src := domain.SupportedIntervalsFor(ex)
	out := make([]domain.CandleInterval, len(src))
	copy(out, src)
	return out, nil
}

// ListProductTags returns unique product tags for filter UI (Binance catalog only).
func (s *Service) ListProductTags(ctx context.Context, exchange string) ([]string, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	return p.ListProductTags(ctx)
}

// normalizeSymbolForExchange uppercases symbols; Coinbase keeps a single hyphen (BTC-USD).
func normalizeSymbolForExchange(ex domain.Exchange, symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	if ex == domain.ExchangeCoinbase {
		symbol = strings.ToUpper(symbol)
		// Accept BTCUSD → BTC-USD when clearly a USD pair without hyphen.
		if !strings.Contains(symbol, "-") {
			for _, q := range []string{"USDT", "USDC", "USD", "EUR", "GBP", "BTC", "ETH"} {
				if strings.HasSuffix(symbol, q) && len(symbol) > len(q) {
					base := strings.TrimSuffix(symbol, q)
					if base != "" {
						return base + "-" + q
					}
				}
			}
		}
		return symbol
	}
	return strings.ToUpper(strings.ReplaceAll(symbol, "-", ""))
}

const (
	defaultSpotLimit = 50
	maxSpotLimit     = 500
)

// ListSpotMarkets lists spot pairs for an exchange with search, metric sort, and pagination.
// Market-cap fields are enriched from Binance circulating supply (best-effort; missing assets stay null).
// When sorting by market-cap fields, rows are collapsed to one preferred quote per base asset
// so multi-pair clones do not dominate the leaderboard with the same asset mcap.
func (s *Service) ListSpotMarkets(ctx context.Context, exchange string, q domain.SpotListQuery) (*domain.SpotListResult, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	q, err = normalizeSpotListQuery(q)
	if err != nil {
		return nil, err
	}

	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	all, err := p.ListSpotMarkets(ctx)
	if err != nil {
		return nil, err
	}

	filtered := filterSpotMarkets(all, q)

	// Mcap sorts need supply on the full filtered set before ordering.
	if q.SortBy.NeedsSupplyEnrichment() {
		enriched := s.enrichSpotMarkets(ctx, filtered)
		if enriched == 0 {
			return nil, fmt.Errorf("%w: supply snapshot unavailable for market-cap sort", domain.ErrUpstream)
		}
		// One row per base so full-asset mcap is not repeated across every quote pair.
		filtered = preferPrimaryQuotePerBase(filtered)
		sortSpotMarkets(filtered, q.SortBy, q.Order)
		total := len(filtered)
		page := pageSpotMarkets(filtered, q.Offset, q.Limit)
		return &domain.SpotListResult{
			Items: page, Total: total, Limit: q.Limit, Offset: q.Offset,
			SortBy: q.SortBy, Order: q.Order, Query: q.Query, Tags: q.Tags, Exchange: ex,
		}, nil
	}

	sortSpotMarkets(filtered, q.SortBy, q.Order)
	total := len(filtered)
	page := pageSpotMarkets(filtered, q.Offset, q.Limit)
	_ = s.enrichSpotMarkets(ctx, page)

	return &domain.SpotListResult{
		Items:    page,
		Total:    total,
		Limit:    q.Limit,
		Offset:   q.Offset,
		SortBy:   q.SortBy,
		Order:    q.Order,
		Query:    q.Query,
		Tags:     q.Tags,
		Exchange: ex,
	}, nil
}

// enrichSpotMarkets fills supply + market-cap from the daily supply cache only (no live fetch).
// Returns the number of items that received a non-nil supply snapshot.
func (s *Service) enrichSpotMarkets(ctx context.Context, items []domain.SpotMarket) int {
	if len(items) == 0 || s.supply == nil {
		return 0
	}
	byAsset := map[string]*domain.AssetSupply{}
	hits := 0
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
		hits++
	}
	for i := range items {
		applySupplyAndMcap(&items[i], byAsset[strings.ToUpper(items[i].BaseAsset)])
	}
	return hits
}

func applySupplyAndMcap(m *domain.SpotMarket, sup *domain.AssetSupply) {
	if m == nil || sup == nil {
		return
	}
	m.CirculatingSupply = domain.CloneFloatPtr(sup.CirculatingSupply)
	m.TotalSupply = domain.CloneFloatPtr(sup.TotalSupply)
	m.MaxSupply = domain.CloneFloatPtr(sup.MaxSupply)

	price := usdPriceForMcap(*m, sup)
	if price == nil {
		// Unknown price: never treat as ranked infinite max mcap.
		m.MarketCapMaxInfinite = false
		m.MarketCapCirculating = nil
		m.MarketCapTotal = nil
		m.MarketCapMax = nil
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
		// Uncapped max supply with a known price → infinite max mcap for display/sort.
		m.MarketCapMax = nil
		m.MarketCapMaxInfinite = true
	} else {
		v := p * *sup.MaxSupply
		m.MarketCapMax = &v
		m.MarketCapMaxInfinite = false
	}
}

// usdPriceForMcap prefers Binance last price when the quote is a USD stablecoin;
// otherwise uses the supply snapshot USD price when available (from a USDT-class pair).
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

// quotePreference ranks USD-stable quotes highest for primary-pair selection.
func quotePreference(quote string) int {
	switch strings.ToUpper(quote) {
	case "USDT":
		return 100
	case "USDC":
		return 90
	case "FDUSD":
		return 80
	case "BUSD":
		return 70
	case "TUSD":
		return 60
	case "DAI":
		return 50
	case "USD":
		return 40
	default:
		return 0
	}
}

// preferPrimaryQuotePerBase keeps one market row per base asset, preferring USDT-class quotes
// then higher quote volume. Used before market-cap sorts so asset mcap is not duplicated.
func preferPrimaryQuotePerBase(items []domain.SpotMarket) []domain.SpotMarket {
	if len(items) == 0 {
		return items
	}
	best := make(map[string]domain.SpotMarket, len(items))
	for _, m := range items {
		base := strings.ToUpper(strings.TrimSpace(m.BaseAsset))
		if base == "" {
			base = strings.ToUpper(m.Symbol)
		}
		prev, ok := best[base]
		if !ok || betterPrimaryPair(m, prev) {
			best[base] = m
		}
	}
	out := make([]domain.SpotMarket, 0, len(best))
	for _, m := range best {
		out = append(out, m)
	}
	return out
}

func betterPrimaryPair(a, b domain.SpotMarket) bool {
	pa, pb := quotePreference(a.QuoteAsset), quotePreference(b.QuoteAsset)
	if pa != pb {
		return pa > pb
	}
	// Prefer higher quote volume as a liquidity signal among equal quote class.
	return cmpFloatString(a.QuoteVolume, b.QuoteVolume) > 0
}

func normalizeSpotListQuery(q domain.SpotListQuery) (domain.SpotListQuery, error) {
	q.Query = strings.TrimSpace(q.Query)
	q.QuoteAsset = strings.ToUpper(strings.TrimSpace(q.QuoteAsset))
	q.BaseAsset = strings.ToUpper(strings.TrimSpace(q.BaseAsset))
	q.Status = strings.ToUpper(strings.TrimSpace(q.Status))
	q.Tags = normalizeTagFilters(q.Tags)

	if q.SortBy == "" {
		q.SortBy = domain.SpotSortQuoteVolume
	}
	if !domain.IsValidSpotSortField(string(q.SortBy)) {
		return q, fmt.Errorf("%w: sort must be one of %v", domain.ErrInvalidArgument, domain.SupportedSpotSortFields)
	}

	if q.Order == "" {
		switch q.SortBy {
		case domain.SpotSortSymbol, domain.SpotSortBaseAsset, domain.SpotSortTags:
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

func normalizeTagFilters(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range tags {
		// Allow comma-separated values inside a single entry.
		for _, part := range strings.Split(raw, ",") {
			t := strings.TrimSpace(part)
			if t == "" {
				continue
			}
			key := strings.ToLower(t)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, t)
		}
	}
	return out
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
		if len(q.Tags) > 0 && !marketHasAnyTag(m, q.Tags) {
			continue
		}
		if needle != "" {
			if !strings.Contains(strings.ToUpper(m.Symbol), needle) &&
				!strings.Contains(strings.ToUpper(m.BaseAsset), needle) &&
				!strings.Contains(strings.ToUpper(m.QuoteAsset), needle) &&
				!marketTagsContainNeedle(m, needle) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// marketHasAnyTag reports whether m has at least one of the filter tags (OR, case-insensitive).
func marketHasAnyTag(m domain.SpotMarket, want []string) bool {
	if len(m.Tags) == 0 {
		return false
	}
	have := map[string]struct{}{}
	for _, t := range m.Tags {
		have[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	for _, w := range want {
		if _, ok := have[strings.ToLower(strings.TrimSpace(w))]; ok {
			return true
		}
	}
	return false
}

func marketTagsContainNeedle(m domain.SpotMarket, needleUpper string) bool {
	for _, t := range m.Tags {
		if strings.Contains(strings.ToUpper(t), needleUpper) {
			return true
		}
	}
	return false
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
			cmp = cmpOptionalFloatNullsLast(a.MarketCapCirculating, b.MarketCapCirculating, desc)
		case domain.SpotSortMarketCapTotal:
			cmp = cmpOptionalFloatNullsLast(a.MarketCapTotal, b.MarketCapTotal, desc)
		case domain.SpotSortMarketCapMax:
			cmp = cmpMcapMax(a, b, desc)
		case domain.SpotSortTags:
			cmp = cmpTags(a.Tags, b.Tags, desc)
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
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseFloat(s, 64)
}

// cmpOptionalFloatNullsLast compares optional floats. Missing values sort last
// for both asc and desc (never equated to zero). When both missing, equal.
// The returned cmp is used with the caller's desc flip for defined values only;
// for null placement we pre-adjust so that after the desc flip, nulls still last.
func cmpOptionalFloatNullsLast(a, b *float64, desc bool) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		// Want a after b always → under asc cmp>0; under desc need cmp<0 so desc flip still places a later.
		if desc {
			return -1
		}
		return 1
	}
	if b == nil {
		if desc {
			return 1
		}
		return -1
	}
	switch {
	case *a < *b:
		return -1
	case *a > *b:
		return 1
	default:
		return 0
	}
}

// cmpTags compares lexicographic join of sorted tags; empty tag lists sort last.
func cmpTags(a, b []string, desc bool) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		if desc {
			return -1
		}
		return 1
	}
	if len(b) == 0 {
		if desc {
			return 1
		}
		return -1
	}
	sa := strings.ToLower(strings.Join(a, ","))
	sb := strings.ToLower(strings.Join(b, ","))
	return strings.Compare(sa, sb)
}

// cmpMcapMax ranks: finite values by size; infinite (known uncapped with price) above all finite;
// unknown (no price / no supply) last.
func cmpMcapMax(a, b domain.SpotMarket, desc bool) int {
	rankKnown := func(m domain.SpotMarket) (float64, bool) {
		if m.MarketCapMaxInfinite {
			return 1e300, true
		}
		if m.MarketCapMax != nil {
			return *m.MarketCapMax, true
		}
		return 0, false
	}
	fa, oka := rankKnown(a)
	fb, okb := rankKnown(b)
	if !oka && !okb {
		return 0
	}
	if !oka {
		if desc {
			return -1
		}
		return 1
	}
	if !okb {
		if desc {
			return 1
		}
		return -1
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
