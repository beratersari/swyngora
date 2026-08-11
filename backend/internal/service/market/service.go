package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	wallSampleEvery = 3 * time.Second
	// Hold a little past WallPersistentMin so the last 3s tick can land
	// after the 2-minute mark instead of deleting the watch just short.
	wallWatchSlack = 15 * time.Second
)

// wallWatchIdle is how long the sampler keeps a book after the last user/AI
// look. It must outlast persistent-min plus a sample tick, otherwise the last
// check happens before the wall can be labeled persistent.
func wallWatchIdle() time.Duration {
	return domain.WallPersistentMin + wallSampleEvery + wallWatchSlack
}

// Service orchestrates market-data use cases. Handlers call this layer only.
type Service struct {
	markets       map[domain.Exchange]domain.MarketDataPort
	supply        domain.SupplyPort
	delist        domain.SpotDelistStore
	delistEnabled bool
	walls         *domain.WallMemory
	watchMu       sync.Mutex
	wallWatch     map[string]wallWatch
}

type wallWatch struct {
	exchange domain.Exchange
	symbol   string
	seen     time.Time
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
	return &Service{markets: cp, supply: supply, walls: domain.NewWallMemory(), wallWatch: map[string]wallWatch{}}
}

// WithDelistStore attaches optional delist schedule for spot enrichment.
func (s *Service) WithDelistStore(store domain.SpotDelistStore) *Service {
	if s != nil {
		s.delist = store
	}
	return s
}

// WithDelistEnabled marks whether hourly delist refresh is configured.
func (s *Service) WithDelistEnabled(enabled bool) *Service {
	if s != nil {
		s.delistEnabled = enabled
	}
	return s
}

// DelistEnabled reports whether delist data can be populated.
func (s *Service) DelistEnabled() bool {
	return s != nil && s.delistEnabled
}

// ListDelistSchedule returns cached schedule for an exchange.
func (s *Service) ListDelistSchedule(exchange string) ([]domain.SpotDelistEntry, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	if s.delist == nil {
		return []domain.SpotDelistEntry{}, nil
	}
	return s.delist.List(ex), nil
}

// DelistTime looks up scheduled delist for one symbol.
func (s *Service) DelistTime(exchange, symbol string) (time.Time, bool) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil || s.delist == nil {
		return time.Time{}, false
	}
	return s.delist.DelistTime(ex, symbol)
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

// GetSpotOrderBook returns a grouped spot order book plus ±rangePct analysis.
func (s *Service) GetSpotOrderBook(ctx context.Context, exchange, symbol, group string, levels int, rangePct float64) (*domain.OrderBook, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	groupSize, err := domain.ParseGroupSize(group)
	if err != nil {
		return nil, err
	}
	levels = domain.ClampOrderBookLevels(levels)
	rangePct = domain.ClampRangePct(rangePct)
	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	// Pull a deep snapshot so analysis can use ±% of price, not only the ladder rows.
	raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: symbol, Limit: domain.MaxOrderBookRawLimit})
	if err != nil {
		return nil, err
	}
	if raw == nil || (len(raw.Bids) == 0 && len(raw.Asks) == 0) {
		return nil, fmt.Errorf("%w: empty order book", domain.ErrNotFound)
	}
	book := domain.GroupOrderBook(*raw, groupSize, levels)
	book.Analysis = domain.AnalyzeOrderBook(*raw, rangePct)
	book.Exchange = ex
	if book.Symbol == "" {
		book.Symbol = symbol
	}
	s.recordWalls(ex, book.Symbol, book.Analysis.Walls)
	return &book, nil
}

// GetCombinedOrderBookAnalysis sums live depth from every configured venue
// inside the same ±rangePct band around a shared mid.
func (s *Service) GetCombinedOrderBookAnalysis(ctx context.Context, symbol string, rangePct float64) (*domain.CombinedOrderBookAnalysis, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	rangePct = domain.ClampRangePct(rangePct)
	books, err := s.fetchVenueBooks(ctx, symbol, nil)
	if err != nil {
		return nil, err
	}
	mid := domain.SharedBookMid(books)
	if mid <= 0 {
		return nil, fmt.Errorf("%w: could not determine shared mid for %s", domain.ErrUpstream, symbol)
	}
	display := domain.CrossVenueSymbol(domain.ExchangeBinance, symbol)
	if display == "" {
		display = strings.ToUpper(symbol)
	}
	combined := domain.CombineOrderBooks(display, mid, rangePct, books)
	now := time.Now().UTC()
	for _, vb := range books {
		if vb.Err != "" {
			continue
		}
		an := domain.AnalyzeOrderBookAt(vb.Book, mid, combined.UsedRangePct)
		s.recordWalls(vb.Exchange, vb.Symbol, an.Walls)
	}
	if s.walls != nil {
		s.walls.ApplyCombined(now, combined.Walls)
	}
	return &combined, nil
}

func (s *Service) fetchVenueBooks(ctx context.Context, symbol string, only []domain.Exchange) ([]domain.VenueRawBook, error) {
	exchanges := only
	if len(exchanges) == 0 {
		exchanges = s.ListExchanges()
	}
	if len(exchanges) == 0 {
		return nil, fmt.Errorf("%w: no exchanges configured", domain.ErrUpstream)
	}
	type result struct {
		vb domain.VenueRawBook
	}
	outCh := make(chan result, len(exchanges))
	for _, ex := range exchanges {
		ex := ex
		go func() {
			pair := domain.CrossVenueSymbol(ex, symbol)
			vb := domain.VenueRawBook{Exchange: ex, Symbol: pair}
			if pair == "" {
				vb.Err = "symbol not mapped"
				outCh <- result{vb: vb}
				return
			}
			p, err := s.port(ex)
			if err != nil {
				vb.Err = err.Error()
				outCh <- result{vb: vb}
				return
			}
			raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: pair, Limit: domain.MaxOrderBookRawLimit})
			if err != nil || raw == nil || (len(raw.Bids) == 0 && len(raw.Asks) == 0) {
				if err != nil {
					vb.Err = err.Error()
				} else {
					vb.Err = "empty order book"
				}
				outCh <- result{vb: vb}
				return
			}
			vb.Book = *raw
			outCh <- result{vb: vb}
		}()
	}
	books := make([]domain.VenueRawBook, 0, len(exchanges))
	ok := 0
	for i := 0; i < len(exchanges); i++ {
		r := <-outCh
		books = append(books, r.vb)
		if r.vb.Err == "" {
			ok++
		}
	}
	if ok == 0 {
		return nil, fmt.Errorf("%w: no live order books for %s", domain.ErrUpstream, symbol)
	}
	return books, nil
}

// GetMarketLiquidity scores buy/sell depth in ±0.1 / ±0.5 / ±1% for each
// venue and a market-wide sum. exchange empty or "all" includes every venue.
func (s *Service) GetMarketLiquidity(ctx context.Context, exchange, symbol string) (*domain.MarketLiquidity, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	var only []domain.Exchange
	rawEx := strings.ToLower(strings.TrimSpace(exchange))
	if rawEx != "" && rawEx != "all" && rawEx != "combined" {
		ex, err := s.ResolveExchange(exchange)
		if err != nil {
			return nil, err
		}
		only = []domain.Exchange{ex}
	}
	books, err := s.fetchVenueBooks(ctx, symbol, only)
	if err != nil {
		return nil, err
	}
	display := domain.CrossVenueSymbol(domain.ExchangeBinance, symbol)
	if display == "" {
		display = strings.ToUpper(symbol)
	}
	out := &domain.MarketLiquidity{
		Symbol: display,
		Venues: []domain.VenueLiquidity{},
	}
	var parts []domain.LiquidityScore
	for _, vb := range books {
		row := domain.VenueLiquidity{Exchange: vb.Exchange, Symbol: vb.Symbol, Error: vb.Err}
		if vb.Err == "" {
			row.Live = vb.Book.Live
			row.LiquidityScore = domain.ScoreBookLiquidity(vb.Book, 0)
			parts = append(parts, row.LiquidityScore)
			out.VenueCount++
		}
		out.Venues = append(out.Venues, row)
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	out.Market = domain.MergeLiquidityScores(parts)
	return out, nil
}

// EstimateOrderBookImpact walks live asks (buy) or bids (sell) for a market size.
// exchange empty or "all" walks Binance+Coinbase+Bybit cheapest-first.
func (s *Service) EstimateOrderBookImpact(ctx context.Context, exchange, symbol, side string, quantity, notional float64) (*domain.OrderBookImpact, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	side, err := domain.ParseImpactSide(side)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateImpactSize(quantity, notional); err != nil {
		return nil, err
	}
	var only []domain.Exchange
	scope := domain.ImpactScopeCombined
	rawEx := strings.ToLower(strings.TrimSpace(exchange))
	if rawEx != "" && rawEx != "all" && rawEx != "combined" {
		ex, err := s.ResolveExchange(exchange)
		if err != nil {
			return nil, err
		}
		only = []domain.Exchange{ex}
		scope = string(ex)
	}
	books, err := s.fetchVenueBooks(ctx, symbol, only)
	if err != nil {
		return nil, err
	}
	mid := domain.ImpactBookMid(books)
	levels := domain.CollectImpactLevels(side, books)
	display := domain.CrossVenueSymbol(domain.ExchangeBinance, symbol)
	if display == "" {
		display = strings.ToUpper(symbol)
	}
	imp := domain.SimulateMarketImpact(display, scope, side, mid, levels, quantity, notional)
	imp.VenueCount = 0
	live := false
	for _, b := range books {
		if b.Err == "" {
			imp.VenueCount++
			if b.Book.Live {
				live = true
			}
		}
	}
	imp.Live = live
	return &imp, nil
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

// ListProductTags returns unique product tags for filter UI.
// Coinbase/Bybit have no native catalog — fall back to Binance tags so filters stay useful.
func (s *Service) ListProductTags(ctx context.Context, exchange string) ([]string, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	tags, err := p.ListProductTags(ctx)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 && ex != domain.ExchangeBinance {
		if bp, berr := s.port(domain.ExchangeBinance); berr == nil {
			tags, err = bp.ListProductTags(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	return s.withDelistFilterTag(ex, tags), nil
}

func (s *Service) withDelistFilterTag(ex domain.Exchange, tags []string) []string {
	if s.delist == nil || len(s.delist.List(ex)) == 0 {
		return tags
	}
	return ensureTag(tags, domain.TagDelist)
}

// normalizeSymbolForExchange delegates to domain.NormalizeSymbol (shared with watchlist).
func normalizeSymbolForExchange(ex domain.Exchange, symbol string) string {
	return domain.NormalizeSymbol(ex, symbol)
}

func (s *Service) recordWalls(ex domain.Exchange, symbol string, walls []domain.OrderBookWall) {
	s.noteWallWatch(ex, symbol)
	s.observeWalls(ex, symbol, walls)
}

func (s *Service) observeWalls(ex domain.Exchange, symbol string, walls []domain.OrderBookWall) {
	if s == nil {
		return
	}
	if s.walls == nil {
		s.walls = domain.NewWallMemory()
	}
	s.walls.Observe(time.Now().UTC(), string(ex), symbol, walls)
}

func (s *Service) noteWallWatch(ex domain.Exchange, symbol string) {
	if s == nil || ex == "" || symbol == "" {
		return
	}
	key := string(ex) + "|" + symbol
	s.watchMu.Lock()
	if s.wallWatch == nil {
		s.wallWatch = map[string]wallWatch{}
	}
	s.wallWatch[key] = wallWatch{exchange: ex, symbol: symbol, seen: time.Now().UTC()}
	s.watchMu.Unlock()
}

// StartWallSampler keeps sampling recently requested books so wall persistence
// and flicker can be measured while the live local book is attached.
func (s *Service) StartWallSampler(ctx context.Context) {
	if s == nil {
		return
	}
	tick := time.NewTicker(wallSampleEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s.sampleWatchedWalls(ctx)
		}
	}
}

func (s *Service) sampleWatchedWalls(ctx context.Context) {
	now := time.Now().UTC()
	s.watchMu.Lock()
	watches := make([]wallWatch, 0, len(s.wallWatch))
	for k, w := range s.wallWatch {
		if now.Sub(w.seen) > wallWatchIdle() {
			delete(s.wallWatch, k)
			continue
		}
		watches = append(watches, w)
	}
	s.watchMu.Unlock()
	for _, w := range watches {
		if ctx.Err() != nil {
			return
		}
		p, err := s.port(w.exchange)
		if err != nil {
			continue
		}
		raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: w.symbol, Limit: domain.MaxOrderBookRawLimit})
		if err != nil || raw == nil {
			continue
		}
		an := domain.AnalyzeOrderBook(*raw, domain.DefaultOrderBookRangePct)
		s.observeWalls(w.exchange, w.symbol, an.Walls)
	}
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

	// Attach Binance product tags by base when the venue has none (Coinbase/Bybit).
	// Must run before tag filters so ?tag=Meme works on every exchange.
	s.enrichProductTags(ctx, all)
	s.enrichDelistTimes(ex, all)

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

// enrichProductTags fills empty Tags from the Binance product catalog by base asset.
// No-op when Binance is not configured or the catalog is unavailable.
func (s *Service) enrichProductTags(ctx context.Context, items []domain.SpotMarket) {
	if len(items) == 0 {
		return
	}
	need := false
	for i := range items {
		if len(items[i].Tags) == 0 {
			need = true
			break
		}
	}
	if !need {
		return
	}
	bp, err := s.port(domain.ExchangeBinance)
	if err != nil {
		return
	}
	byBase, err := bp.TagsByBase(ctx)
	if err != nil || len(byBase) == 0 {
		return
	}
	for i := range items {
		if len(items[i].Tags) > 0 {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(items[i].BaseAsset))
		if base == "" {
			continue
		}
		if tags, ok := byBase[base]; ok && len(tags) > 0 {
			items[i].Tags = append([]string(nil), tags...)
		}
	}
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
	// Missing volume loses to a known positive volume.
	fa, erra := parseFloat(a.QuoteVolume)
	fb, errb := parseFloat(b.QuoteVolume)
	if erra != nil {
		return false
	}
	if errb != nil {
		return true
	}
	return fa > fb
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
			cmp = cmpFloatStringNullsLast(a.Volume, b.Volume, desc)
		case domain.SpotSortPriceChangePercent:
			cmp = cmpFloatStringNullsLast(a.PriceChangePercent, b.PriceChangePercent, desc)
		case domain.SpotSortLastPrice:
			cmp = cmpFloatStringNullsLast(a.LastPrice, b.LastPrice, desc)
		case domain.SpotSortMarketCapCirculating:
			cmp = cmpOptionalFloatNullsLast(a.MarketCapCirculating, b.MarketCapCirculating, desc)
		case domain.SpotSortMarketCapTotal:
			cmp = cmpOptionalFloatNullsLast(a.MarketCapTotal, b.MarketCapTotal, desc)
		case domain.SpotSortMarketCapMax:
			cmp = cmpMcapMax(a, b, desc)
		case domain.SpotSortTags:
			cmp = cmpTags(a.Tags, b.Tags, desc)
		default: // quoteVolume
			cmp = cmpFloatStringNullsLast(a.QuoteVolume, b.QuoteVolume, desc)
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

// cmpFloatStringNullsLast compares decimal strings. Unparseable/empty values sort
// last for both asc and desc (never equated to zero).
func cmpFloatStringNullsLast(a, b string, desc bool) int {
	fa, erra := parseFloat(a)
	fb, errb := parseFloat(b)
	if erra != nil && errb != nil {
		return 0
	}
	if erra != nil {
		if desc {
			return -1
		}
		return 1
	}
	if errb != nil {
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

// cmpFloatString is kept for tests / callers that do not need nulls-last.
func cmpFloatString(a, b string) int {
	return cmpFloatStringNullsLast(a, b, false)
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
	sa := strings.ToLower(strings.Join(sortedCopyTags(a), ","))
	sb := strings.ToLower(strings.Join(sortedCopyTags(b), ","))
	return strings.Compare(sa, sb)
}

func sortedCopyTags(in []string) []string {
	out := append([]string(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
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

func (s *Service) enrichDelistTimes(ex domain.Exchange, items []domain.SpotMarket) {
	if s.delist == nil || len(items) == 0 {
		return
	}
	for i := range items {
		if t, ok := s.delist.DelistTime(ex, items[i].Symbol); ok {
			tt := t
			items[i].DelistTime = &tt
			items[i].Tags = ensureTag(items[i].Tags, domain.TagDelist)
		}
	}
}

func ensureTag(tags []string, tag string) []string {
	if tag == "" {
		return tags
	}
	want := strings.ToLower(tag)
	for _, t := range tags {
		if strings.ToLower(strings.TrimSpace(t)) == want {
			return tags
		}
	}
	out := make([]string, 0, len(tags)+1)
	out = append(out, tag)
	out = append(out, tags...)
	return out
}
