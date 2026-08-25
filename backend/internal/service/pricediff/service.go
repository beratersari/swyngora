package pricediff

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// TickerFetcher loads last prices (usually *market.Service).
type TickerFetcher interface {
	GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error)
}

// BookFetcher loads a raw spot book for one venue (usually *market.Service).
type BookFetcher interface {
	GetRawOrderBook(ctx context.Context, exchange, symbol string) (*domain.RawOrderBook, error)
}

// QuoteInput is a standalone two-venue executable quote (no stored opportunity).
type QuoteInput struct {
	Symbol        string
	BuyExchange   string
	SellExchange  string
	BuyFeePct     float64
	SellFeePct    float64
	Notional      float64
	Quantity      float64
	MinNetDiffPct float64
}

// CreateInput creates a cross-exchange price difference watch.
type CreateInput struct {
	ClientID       string
	Symbol         string
	MinNetDiffPct  float64
	FeeBinancePct  float64
	FeeCoinbasePct float64
	FeeBybitPct    float64
}

// AccountChecker reports whether a tenant is closed so workers can skip them.
type AccountChecker interface {
	IsClosed(ctx context.Context, clientID string) (bool, *domain.Account, error)
}

// Service orchestrates price-diff watches and opportunity evaluation.
type Service struct {
	store   domain.PriceDiffPort
	market  TickerFetcher
	books   BookFetcher
	account AccountChecker
	// MaxPriceAge rejects tickers older than this (default 2m).
	MaxPriceAge time.Duration
}

// New constructs a price-diff service.
func New(store domain.PriceDiffPort, market TickerFetcher) *Service {
	s := &Service{
		store:       store,
		market:      market,
		MaxPriceAge: domain.DefaultPriceDiffMaxAge,
	}
	if b, ok := market.(BookFetcher); ok {
		s.books = b
	}
	return s
}

// WithBooks attaches a raw-book source for executable quotes.
func (s *Service) WithBooks(b BookFetcher) *Service {
	if s != nil {
		s.books = b
	}
	return s
}

// SetAccountChecker wires account-closed skips for ProcessActiveWatches.
func (s *Service) SetAccountChecker(a AccountChecker) {
	if s != nil {
		s.account = a
	}
}

func tenantClosed(ctx context.Context, accounts AccountChecker, clientID string) bool {
	if accounts == nil || clientID == "" {
		return false
	}
	closed, _, err := accounts.IsClosed(ctx, clientID)
	return err == nil && closed
}

// PurgeClient deletes watches (and their opportunities via store FK/delete) for a tenant.
func (s *Service) PurgeClient(ctx context.Context, clientID string) error {
	if s == nil || s.store == nil {
		return nil
	}
	list, err := s.store.ListWatches(ctx, clientID)
	if err != nil {
		return err
	}
	for i := range list {
		_ = s.store.DeleteWatch(ctx, clientID, list[i].ID)
	}
	return nil
}

// CreateWatch validates and stores an active watch.
func (s *Service) CreateWatch(ctx context.Context, in CreateInput) (*domain.PriceDiffWatch, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: price-diff store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	sym := domain.NormalizeSymbol(domain.ExchangeBinance, in.Symbol)
	if sym == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if in.MinNetDiffPct < domain.MinPriceDiffNetPct || in.MinNetDiffPct > domain.MaxPriceDiffNetPct ||
		math.IsNaN(in.MinNetDiffPct) || math.IsInf(in.MinNetDiffPct, 0) {
		return nil, fmt.Errorf("%w: minNetDiffPct must be between %g and %g", domain.ErrInvalidArgument,
			domain.MinPriceDiffNetPct, domain.MaxPriceDiffNetPct)
	}
	for _, f := range []float64{in.FeeBinancePct, in.FeeCoinbasePct, in.FeeBybitPct} {
		if f < 0 || f > domain.MaxPriceDiffFeePct || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("%w: fees must be between 0 and %g percent", domain.ErrInvalidArgument, domain.MaxPriceDiffFeePct)
		}
	}
	n, err := s.store.CountWatches(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxPriceDiffWatchesPerClient {
		return nil, fmt.Errorf("%w: max %d price-diff watches per client", domain.ErrInvalidArgument, domain.MaxPriceDiffWatchesPerClient)
	}
	now := time.Now().UTC()
	w := domain.PriceDiffWatch{
		ID: uuid.NewString(), ClientID: clientID, Symbol: sym,
		MinNetDiffPct: in.MinNetDiffPct,
		FeeBinancePct: in.FeeBinancePct, FeeCoinbasePct: in.FeeCoinbasePct, FeeBybitPct: in.FeeBybitPct,
		Status: domain.PriceDiffWatchActive, CreatedAt: now, UpdatedAt: now,
	}
	return s.store.CreateWatch(ctx, w)
}

// ListWatches lists watches for a client.
func (s *Service) ListWatches(ctx context.Context, clientID string) ([]domain.PriceDiffWatch, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: price-diff store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListWatches(ctx, clientID)
}

// GetWatch returns one watch.
func (s *Service) GetWatch(ctx context.Context, clientID, id string) (*domain.PriceDiffWatch, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: price-diff store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: watch id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetWatch(ctx, clientID, id)
}

// DeleteWatch removes a watch and its opportunities.
func (s *Service) DeleteWatch(ctx context.Context, clientID, id string) error {
	if s.store == nil {
		return fmt.Errorf("%w: price-diff store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: watch id is required", domain.ErrInvalidArgument)
	}
	return s.store.DeleteWatch(ctx, clientID, id)
}

// ListOpportunities lists opportunities for a client (status: open|closed|"" for all).
func (s *Service) ListOpportunities(ctx context.Context, clientID, status string, limit, offset int) ([]domain.PriceDiffOpportunity, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: price-diff store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	var st domain.PriceDiffOppStatus
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", string(domain.PriceDiffOppOpen):
		st = domain.PriceDiffOppOpen
	case "all":
		st = ""
	case string(domain.PriceDiffOppClosed):
		st = domain.PriceDiffOppClosed
	default:
		return nil, fmt.Errorf("%w: status must be open, closed, or all", domain.ErrInvalidArgument)
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListOpportunities(ctx, clientID, st, limit, offset)
}

// GetOpportunity returns one opportunity.
func (s *Service) GetOpportunity(ctx context.Context, clientID, id string) (*domain.PriceDiffOpportunity, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: price-diff store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: opportunity id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetOpportunity(ctx, clientID, id)
}

// ProcessActiveWatches evaluates all active watches once (worker).
func (s *Service) ProcessActiveWatches(ctx context.Context, now time.Time) (created, closed, touched int, err error) {
	if s.store == nil || s.market == nil {
		return 0, 0, 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	watches, err := s.store.ListActiveWatches(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	for i := range watches {
		if tenantClosed(ctx, s.account, watches[i].ClientID) {
			continue
		}
		c, cl, t, e := s.processWatch(ctx, &watches[i], now)
		if e != nil {
			continue
		}
		created += c
		closed += cl
		touched += t
	}
	return created, closed, touched, nil
}

func (s *Service) processWatch(ctx context.Context, w *domain.PriceDiffWatch, now time.Time) (created, closed, touched int, err error) {
	maxAge := s.MaxPriceAge
	if maxAge <= 0 {
		maxAge = domain.DefaultPriceDiffMaxAge
	}
	prices := map[domain.Exchange]float64{}
	for _, ex := range domain.SupportedExchanges {
		if domain.IsEquityExchange(ex) {
			continue
		}
		sym := domain.PriceDiffSymbolForExchange(ex, w.Symbol)
		tkr, err := s.market.GetTicker24h(ctx, string(ex), sym)
		if err != nil || tkr == nil {
			continue
		}
		if !domain.IsFreshTicker(tkr, now, maxAge) {
			continue
		}
		last, err := strconv.ParseFloat(tkr.LastPrice, 64)
		if err != nil || last <= 0 {
			continue
		}
		prices[ex] = last
	}
	// Need at least two fresh venues to compare.
	if len(prices) < 2 {
		return 0, 0, 0, nil
	}

	fees := map[domain.Exchange]float64{
		domain.ExchangeBinance:  w.FeeBinancePct,
		domain.ExchangeCoinbase: w.FeeCoinbasePct,
		domain.ExchangeBybit:    w.FeeBybitPct,
	}
	routes := domain.BestPriceDiffRoutes(prices, fees, w.MinNetDiffPct)

	// Routes currently above the min net threshold.
	activeKeys := map[string]domain.PriceDiffRoute{}
	for _, r := range routes {
		activeKeys[routeKey(r.BuyExchange, r.SellExchange)] = r
	}

	open, err := s.store.ListOpenOpportunitiesForWatch(ctx, w.ID)
	if err != nil {
		return 0, 0, 0, err
	}
	openByKey := map[string]*domain.PriceDiffOpportunity{}
	for i := range open {
		k := routeKey(open[i].BuyExchange, open[i].SellExchange)
		openByKey[k] = &open[i]

		buyP, buyOK := prices[open[i].BuyExchange]
		sellP, sellOK := prices[open[i].SellExchange]
		if !buyOK || !sellOK {
			// Missing/stale price: do not open or close based on incomplete data.
			continue
		}
		gross, net, err := domain.NetDiffPctAfterFees(buyP, sellP, fees[open[i].BuyExchange], fees[open[i].SellExchange])
		if err != nil {
			continue
		}
		if net+1e-12 < w.MinNetDiffPct {
			if _, e := s.store.CloseOpportunity(ctx, open[i].ID, now); e == nil {
				closed++
			}
			continue
		}
		if _, e := s.store.TouchOpportunity(ctx, open[i].ID, buyP, sellP, gross, net, now); e == nil {
			touched++
		}
	}

	// Create opportunities for routes above threshold with no open row.
	for k, r := range activeKeys {
		if _, exists := openByKey[k]; exists {
			continue
		}
		opp := domain.PriceDiffOpportunity{
			ID: uuid.NewString(), WatchID: w.ID, ClientID: w.ClientID, Symbol: w.Symbol,
			BuyExchange: r.BuyExchange, SellExchange: r.SellExchange,
			BuyPrice: r.BuyPrice, SellPrice: r.SellPrice,
			GrossDiffPct: r.GrossDiffPct, NetDiffPct: r.NetDiffPct,
			MinNetDiffPct: w.MinNetDiffPct, Status: domain.PriceDiffOppOpen,
			OpenedAt: now, LastSeenAt: now,
		}
		if _, e := s.store.CreateOpportunity(ctx, opp); e == nil {
			created++
		}
	}
	return created, closed, touched, nil
}

// Quote walks live books for a buy/sell venue pair.
func (s *Service) Quote(ctx context.Context, in QuoteInput) (*domain.PriceDiffQuote, error) {
	if s == nil || s.books == nil {
		return nil, fmt.Errorf("%w: order books not configured", domain.ErrUpstream)
	}
	buy := domain.ParseExchange(in.BuyExchange)
	sell := domain.ParseExchange(in.SellExchange)
	if buy == "" || sell == "" {
		return nil, fmt.Errorf("%w: buyExchange and sellExchange must be known venues", domain.ErrInvalidArgument)
	}
	sym := domain.NormalizeSymbol(domain.ExchangeBinance, in.Symbol)
	if sym == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	q := domain.PriceDiffQuoteQuery{
		Symbol:        sym,
		BuyExchange:   buy,
		SellExchange:  sell,
		BuyFeePct:     in.BuyFeePct,
		SellFeePct:    in.SellFeePct,
		Notional:      in.Notional,
		Quantity:      in.Quantity,
		MinNetDiffPct: in.MinNetDiffPct,
	}
	if err := domain.ValidatePriceDiffQuoteRoute(buy, sell, in.BuyFeePct, in.SellFeePct); err != nil {
		return nil, err
	}
	if err := domain.ValidatePriceDiffQuoteSize(in.Notional, in.Quantity); err != nil {
		return nil, err
	}
	buyBook, sellBook, err := s.fetchQuoteBooks(ctx, buy, sell, sym)
	if err != nil {
		return nil, err
	}
	return domain.QuotePriceDiffRoute(q, buyBook, sellBook)
}

// ScanInput quotes every crypto venue pair at one size.
type ScanInput struct {
	Symbol          string
	Notional        float64
	Quantity        float64
	FeeBinancePct   float64
	FeeCoinbasePct  float64
	FeeBybitPct     float64
	MinNetDiffPct   float64
	MinProfitPct    float64
	MinProfitAmount float64
}

// QuoteScan walks Binance, Coinbase, and Bybit books and ranks every buy/sell route.
func (s *Service) QuoteScan(ctx context.Context, in ScanInput) (*domain.PriceDiffQuoteScan, error) {
	if s == nil || s.books == nil {
		return nil, fmt.Errorf("%w: order books not configured", domain.ErrUpstream)
	}
	sym := domain.NormalizeSymbol(domain.ExchangeBinance, in.Symbol)
	if sym == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	for _, f := range []float64{in.FeeBinancePct, in.FeeCoinbasePct, in.FeeBybitPct} {
		if f < 0 || f > domain.MaxPriceDiffFeePct || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("%w: fees must be between 0 and %g percent", domain.ErrInvalidArgument, domain.MaxPriceDiffFeePct)
		}
	}
	books, unavailable, err := s.fetchAllQuoteBooks(ctx, sym)
	if err != nil {
		return nil, err
	}
	return domain.ScanPriceDiffQuotes(domain.PriceDiffScanQuery{
		Symbol:   sym,
		Notional: in.Notional,
		Quantity: in.Quantity,
		Fees: map[domain.Exchange]float64{
			domain.ExchangeBinance:  in.FeeBinancePct,
			domain.ExchangeCoinbase: in.FeeCoinbasePct,
			domain.ExchangeBybit:    in.FeeBybitPct,
		},
		MinNetDiffPct:   in.MinNetDiffPct,
		MinProfitPct:    in.MinProfitPct,
		MinProfitAmount: in.MinProfitAmount,
		Books:           books,
		Unavailable:     unavailable,
	})
}

// QuoteWatch scans all venue pairs using a stored watch's symbol and fees.
func (s *Service) QuoteWatch(ctx context.Context, clientID, watchID string, in ScanInput) (*domain.PriceDiffQuoteScan, error) {
	w, err := s.GetWatch(ctx, clientID, watchID)
	if err != nil {
		return nil, err
	}
	in.Symbol = w.Symbol
	in.FeeBinancePct = w.FeeBinancePct
	in.FeeCoinbasePct = w.FeeCoinbasePct
	in.FeeBybitPct = w.FeeBybitPct
	if in.MinNetDiffPct == 0 {
		in.MinNetDiffPct = w.MinNetDiffPct
	}
	return s.QuoteScan(ctx, in)
}

func (s *Service) fetchAllQuoteBooks(ctx context.Context, symbol string) (map[domain.Exchange]*domain.RawOrderBook, []domain.PriceDiffUnavailable, error) {
	type result struct {
		ex   domain.Exchange
		book *domain.RawOrderBook
		err  error
	}
	var venues []domain.Exchange
	for _, ex := range domain.SupportedExchanges {
		if !domain.IsEquityExchange(ex) {
			venues = append(venues, ex)
		}
	}
	ch := make(chan result, len(venues))
	for _, ex := range venues {
		ex := ex
		go func() {
			b, err := s.books.GetRawOrderBook(ctx, string(ex), domain.PriceDiffSymbolForExchange(ex, symbol))
			ch <- result{ex: ex, book: b, err: err}
		}()
	}
	out := make(map[domain.Exchange]*domain.RawOrderBook, len(venues))
	var unavailable []domain.PriceDiffUnavailable
	for range venues {
		r := <-ch
		book, err := bookOrEmpty(r.book, r.err)
		if err != nil || book == nil || (len(book.Bids) == 0 && len(book.Asks) == 0) {
			unavailable = append(unavailable, domain.PriceDiffUnavailable{
				Exchange: string(r.ex),
				Reason:   domain.PriceDiffUnavailableBook,
				Message:  "order book could not be loaded",
			})
			continue
		}
		out[r.ex] = book
	}
	return out, unavailable, nil
}

// QuoteOpportunity loads a stored opportunity (and its watch fees) and quotes a size.
func (s *Service) QuoteOpportunity(ctx context.Context, clientID, id string, notional, quantity float64) (*domain.PriceDiffQuote, error) {
	opp, err := s.GetOpportunity(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	watch, err := s.store.GetWatch(ctx, clientID, opp.WatchID)
	if err != nil {
		return nil, err
	}
	quote, err := s.Quote(ctx, QuoteInput{
		Symbol:        opp.Symbol,
		BuyExchange:   string(opp.BuyExchange),
		SellExchange:  string(opp.SellExchange),
		BuyFeePct:     watch.FeePctFor(opp.BuyExchange),
		SellFeePct:    watch.FeePctFor(opp.SellExchange),
		Notional:      notional,
		Quantity:      quantity,
		MinNetDiffPct: watch.MinNetDiffPct,
	})
	if err != nil {
		return nil, err
	}
	return quote, nil
}

func (s *Service) fetchQuoteBooks(ctx context.Context, buy, sell domain.Exchange, symbol string) (*domain.RawOrderBook, *domain.RawOrderBook, error) {
	type result struct {
		book *domain.RawOrderBook
		err  error
	}
	buyCh := make(chan result, 1)
	sellCh := make(chan result, 1)
	go func() {
		b, err := s.books.GetRawOrderBook(ctx, string(buy), domain.PriceDiffSymbolForExchange(buy, symbol))
		buyCh <- result{book: b, err: err}
	}()
	go func() {
		b, err := s.books.GetRawOrderBook(ctx, string(sell), domain.PriceDiffSymbolForExchange(sell, symbol))
		sellCh <- result{book: b, err: err}
	}()
	br := <-buyCh
	sr := <-sellCh
	buyBook, err := bookOrEmpty(br.book, br.err)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: buy book %s: %v", domain.ErrUpstream, buy, err)
	}
	sellBook, err := bookOrEmpty(sr.book, sr.err)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: sell book %s: %v", domain.ErrUpstream, sell, err)
	}
	return buyBook, sellBook, nil
}

func bookOrEmpty(book *domain.RawOrderBook, err error) (*domain.RawOrderBook, error) {
	if err == nil && book != nil {
		return book, nil
	}
	if err != nil && errors.Is(err, domain.ErrNotFound) {
		if book != nil {
			return book, nil
		}
		return &domain.RawOrderBook{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.RawOrderBook{}, nil
}

func routeKey(buy, sell domain.Exchange) string {
	return string(buy) + "->" + string(sell)
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
}
