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
	Notional       float64
	MinProfit      float64
	MinNetDiffPct  float64
	MinDurationSec float64
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
	if err := validateWatchNotional(in.Notional); err != nil {
		return nil, err
	}
	if err := validateWatchMinProfit(in.MinProfit); err != nil {
		return nil, err
	}
	if err := validateWatchMinNet(in.MinNetDiffPct); err != nil {
		return nil, err
	}
	minDur, err := domain.ResolvePriceDiffMinDuration(in.MinDurationSec)
	if err != nil {
		return nil, err
	}
	if err := validateWatchFees(in.FeeBinancePct, in.FeeCoinbasePct, in.FeeBybitPct); err != nil {
		return nil, err
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
		Notional: in.Notional, MinProfit: in.MinProfit, MinNetDiffPct: in.MinNetDiffPct,
		MinDurationSec: minDur.Seconds(),
		FeeBinancePct:  in.FeeBinancePct, FeeCoinbasePct: in.FeeCoinbasePct, FeeBybitPct: in.FeeBybitPct,
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

// UpdateInput patches watch settings. Nil fields stay unchanged.
type UpdateInput struct {
	ClientID       string
	ID             string
	Notional       *float64
	MinProfit      *float64
	MinNetDiffPct  *float64
	MinDurationSec *float64
	FeeBinancePct  *float64
	FeeCoinbasePct *float64
	FeeBybitPct    *float64
}

// UpdateWatch changes notional, min profit, duration, and fees without deleting.
// A settings change clears the duration timer so the next hold starts from zero.
func (s *Service) UpdateWatch(ctx context.Context, in UpdateInput) (*domain.PriceDiffWatch, error) {
	w, err := s.GetWatch(ctx, in.ClientID, in.ID)
	if err != nil {
		return nil, err
	}
	changed := false
	if in.Notional != nil {
		if err := validateWatchNotional(*in.Notional); err != nil {
			return nil, err
		}
		w.Notional = *in.Notional
		changed = true
	}
	if in.MinProfit != nil {
		if err := validateWatchMinProfit(*in.MinProfit); err != nil {
			return nil, err
		}
		w.MinProfit = *in.MinProfit
		changed = true
	}
	if in.MinNetDiffPct != nil {
		if err := validateWatchMinNet(*in.MinNetDiffPct); err != nil {
			return nil, err
		}
		w.MinNetDiffPct = *in.MinNetDiffPct
		changed = true
	}
	if in.MinDurationSec != nil {
		minDur, err := domain.ResolvePriceDiffMinDuration(*in.MinDurationSec)
		if err != nil {
			return nil, err
		}
		w.MinDurationSec = minDur.Seconds()
		changed = true
	}
	if in.FeeBinancePct != nil || in.FeeCoinbasePct != nil || in.FeeBybitPct != nil {
		fb, fc, fy := w.FeeBinancePct, w.FeeCoinbasePct, w.FeeBybitPct
		if in.FeeBinancePct != nil {
			fb = *in.FeeBinancePct
		}
		if in.FeeCoinbasePct != nil {
			fc = *in.FeeCoinbasePct
		}
		if in.FeeBybitPct != nil {
			fy = *in.FeeBybitPct
		}
		if err := validateWatchFees(fb, fc, fy); err != nil {
			return nil, err
		}
		w.FeeBinancePct, w.FeeCoinbasePct, w.FeeBybitPct = fb, fc, fy
		changed = true
	}
	if !changed {
		return w, nil
	}
	if err := s.store.ClearRouteArms(ctx, w.ID); err != nil {
		return nil, err
	}
	w.UpdatedAt = time.Now().UTC()
	return s.store.UpdateWatch(ctx, *w)
}

// PauseWatch stops evaluation, closes open opportunities, and drops the duration timer.
func (s *Service) PauseWatch(ctx context.Context, clientID, id string) (*domain.PriceDiffWatch, error) {
	w, err := s.GetWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.closeOpenAndClearArms(ctx, w.ID, now); err != nil {
		return nil, err
	}
	if w.Status == domain.PriceDiffWatchPaused {
		return w, nil
	}
	w.Status = domain.PriceDiffWatchPaused
	w.UpdatedAt = now
	return s.store.UpdateWatch(ctx, *w)
}

// ResumeWatch starts evaluating again. The duration timer starts from zero
// (it does not continue any wait that was in progress before pause).
func (s *Service) ResumeWatch(ctx context.Context, clientID, id string) (*domain.PriceDiffWatch, error) {
	w, err := s.GetWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	if w.Status == domain.PriceDiffWatchActive {
		return w, nil
	}
	if err := s.store.ClearRouteArms(ctx, w.ID); err != nil {
		return nil, err
	}
	w.Status = domain.PriceDiffWatchActive
	w.UpdatedAt = time.Now().UTC()
	return s.store.UpdateWatch(ctx, *w)
}

func (s *Service) closeOpenAndClearArms(ctx context.Context, watchID string, now time.Time) error {
	open, err := s.store.ListOpenOpportunitiesForWatch(ctx, watchID)
	if err != nil {
		return err
	}
	for i := range open {
		if _, e := s.store.CloseOpportunity(ctx, open[i].ID, now); e != nil && !errors.Is(e, domain.ErrNotFound) {
			return e
		}
	}
	return s.store.ClearRouteArms(ctx, watchID)
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
	if s.books == nil || w.Notional < domain.MinPriceDiffNotional {
		return 0, 0, 0, nil
	}
	maxAge := s.MaxPriceAge
	if maxAge <= 0 {
		maxAge = domain.DefaultPriceDiffMaxAge
	}
	minDur, err := domain.ResolvePriceDiffMinDuration(w.MinDurationSec)
	if err != nil {
		return 0, 0, 0, err
	}
	books, _, err := s.fetchAllQuoteBooks(ctx, w.Symbol)
	if err != nil {
		return 0, 0, 0, err
	}
	if len(books) < 2 {
		return 0, 0, 0, nil
	}

	fees := map[domain.Exchange]float64{
		domain.ExchangeBinance:  w.FeeBinancePct,
		domain.ExchangeCoinbase: w.FeeCoinbasePct,
		domain.ExchangeBybit:    w.FeeBybitPct,
	}
	var venues []domain.Exchange
	for _, ex := range domain.SupportedExchanges {
		if domain.IsEquityExchange(ex) {
			continue
		}
		b := books[ex]
		if b == nil || (len(b.Bids) == 0 && len(b.Asks) == 0) {
			continue
		}
		if !domain.IsFreshOrderBook(b, now, maxAge) {
			continue
		}
		venues = append(venues, ex)
	}
	if len(venues) < 2 {
		return 0, 0, 0, nil
	}

	quoted := map[string]*domain.PriceDiffQuote{}
	active := map[string]*domain.PriceDiffQuote{}
	for i := range venues {
		for j := range venues {
			if i == j {
				continue
			}
			buy, sell := venues[i], venues[j]
			q, qerr := domain.QuotePriceDiffRoute(domain.PriceDiffQuoteQuery{
				Symbol: w.Symbol, BuyExchange: buy, SellExchange: sell,
				BuyFeePct: fees[buy], SellFeePct: fees[sell],
				Notional: w.Notional, MinNetDiffPct: w.MinNetDiffPct,
			}, books[buy], books[sell])
			if qerr != nil || q == nil {
				continue
			}
			k := routeKey(buy, sell)
			quoted[k] = q
			if bookRouteQualifies(q, w) {
				active[k] = q
			}
		}
	}

	armByKey := map[string]time.Time{}
	if minDur > 0 {
		arms, aerr := s.store.ListRouteArms(ctx, w.ID)
		if aerr != nil {
			return 0, 0, 0, aerr
		}
		for _, a := range arms {
			armByKey[routeKey(a.BuyExchange, a.SellExchange)] = a.ArmedAt
		}
	}

	open, err := s.store.ListOpenOpportunitiesForWatch(ctx, w.ID)
	if err != nil {
		return 0, 0, 0, err
	}
	openByKey := map[string]*domain.PriceDiffOpportunity{}
	for i := range open {
		k := routeKey(open[i].BuyExchange, open[i].SellExchange)
		openByKey[k] = &open[i]
		q, haveQuote := quoted[k]
		if !haveQuote {
			continue
		}
		if _, ok := active[k]; !ok {
			_ = s.store.ClearRouteArm(ctx, w.ID, open[i].BuyExchange, open[i].SellExchange)
			if _, e := s.store.CloseOpportunity(ctx, open[i].ID, now); e == nil {
				closed++
			}
			continue
		}
		buyP, sellP := quoteFillPrices(q)
		if _, e := s.store.TouchOpportunity(ctx, open[i].ID, buyP, sellP, q.GrossPct, q.ProfitPct, now); e == nil {
			touched++
		}
	}

	for k, q := range active {
		if _, exists := openByKey[k]; exists {
			continue
		}
		if minDur > 0 {
			since, ok := armByKey[k]
			if !ok {
				_ = s.store.SetRouteArm(ctx, domain.PriceDiffRouteArm{
					WatchID: w.ID, BuyExchange: q.BuyExchange, SellExchange: q.SellExchange, ArmedAt: now,
				})
				continue
			}
			if !domain.PriceDiffHeldLongEnough(since, now, minDur) {
				continue
			}
		}
		buyP, sellP := quoteFillPrices(q)
		opp := domain.PriceDiffOpportunity{
			ID: uuid.NewString(), WatchID: w.ID, ClientID: w.ClientID, Symbol: w.Symbol,
			BuyExchange: q.BuyExchange, SellExchange: q.SellExchange,
			BuyPrice: buyP, SellPrice: sellP,
			GrossDiffPct: q.GrossPct, NetDiffPct: q.ProfitPct,
			MinNetDiffPct: w.MinNetDiffPct, Status: domain.PriceDiffOppOpen,
			OpenedAt: now, LastSeenAt: now,
		}
		if _, e := s.store.CreateOpportunity(ctx, opp); e == nil {
			created++
			_ = s.store.ClearRouteArm(ctx, w.ID, q.BuyExchange, q.SellExchange)
		}
	}
	for k, q := range quoted {
		if _, ok := active[k]; ok {
			continue
		}
		_ = s.store.ClearRouteArm(ctx, w.ID, q.BuyExchange, q.SellExchange)
	}
	return created, closed, touched, nil
}

func bookRouteQualifies(q *domain.PriceDiffQuote, w *domain.PriceDiffWatch) bool {
	if q == nil || w == nil || !q.Executable {
		return false
	}
	if q.AfterFeeProfit()+1e-12 < w.MinProfit {
		return false
	}
	if w.MinNetDiffPct > 0 && q.ProfitPct+1e-12 < w.MinNetDiffPct {
		return false
	}
	return true
}

func quoteFillPrices(q *domain.PriceDiffQuote) (buy, sell float64) {
	if q == nil {
		return 0, 0
	}
	buy, _ = strconv.ParseFloat(q.AverageBuyPrice, 64)
	sell, _ = strconv.ParseFloat(q.AverageSellPrice, 64)
	return buy, sell
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
	if in.Notional == 0 && in.Quantity == 0 {
		in.Notional = w.Notional
	}
	if in.MinProfitAmount == 0 {
		in.MinProfitAmount = w.MinProfit
	}
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

func validateWatchNotional(v float64) error {
	if v < domain.MinPriceDiffNotional || v > domain.MaxPriceDiffNotional ||
		math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%w: notional must be between %g and %g", domain.ErrInvalidArgument,
			domain.MinPriceDiffNotional, domain.MaxPriceDiffNotional)
	}
	return nil
}

func validateWatchMinProfit(v float64) error {
	if v < 0 || v > domain.MaxPriceDiffMinProfit || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%w: minProfit must be between 0 and %g", domain.ErrInvalidArgument, domain.MaxPriceDiffMinProfit)
	}
	return nil
}

func validateWatchMinNet(v float64) error {
	if v < 0 || v > domain.MaxPriceDiffNetPct || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%w: minNetDiffPct must be between 0 and %g", domain.ErrInvalidArgument, domain.MaxPriceDiffNetPct)
	}
	return nil
}

func validateWatchFees(binance, coinbase, bybit float64) error {
	for _, f := range []float64{binance, coinbase, bybit} {
		if f < 0 || f > domain.MaxPriceDiffFeePct || math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("%w: fees must be between 0 and %g percent", domain.ErrInvalidArgument, domain.MaxPriceDiffFeePct)
		}
	}
	return nil
}
