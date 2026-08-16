package pricediff

import (
	"context"
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

// CreateInput creates a cross-exchange price difference watch.
type CreateInput struct {
	ClientID       string
	Symbol         string
	MinNetDiffPct  float64
	FeeBinancePct  float64
	FeeCoinbasePct float64
	FeeBybitPct    float64
}

// Service orchestrates price-diff watches and opportunity evaluation.
type Service struct {
	store  domain.PriceDiffPort
	market TickerFetcher
	// MaxPriceAge rejects tickers older than this (default 2m).
	MaxPriceAge time.Duration
}

// New constructs a price-diff service.
func New(store domain.PriceDiffPort, market TickerFetcher) *Service {
	return &Service{
		store:       store,
		market:      market,
		MaxPriceAge: domain.DefaultPriceDiffMaxAge,
	}
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

func routeKey(buy, sell domain.Exchange) string {
	return string(buy) + "->" + string(sell)
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
}
