package portfolio

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

const (
	maxClientIDLen = 128
	paperNote      = "Paper trading only — simulated fills at last market price. Not financial advice. No real money."
)

// PriceFetcher loads last prices for paper fills and marks.
type PriceFetcher interface {
	GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error)
}

// Service orchestrates paper-trading portfolios.
type Service struct {
	store  domain.PortfolioPort
	market PriceFetcher
}

// New constructs a portfolio service.
func New(store domain.PortfolioPort, market PriceFetcher) *Service {
	return &Service{store: store, market: market}
}

// CreateInput creates a paper portfolio.
type CreateInput struct {
	ClientID        string
	StartingBalance float64
	Currency        string
}

// OrderInput is a market buy/sell.
type OrderInput struct {
	ClientID string
	Exchange string
	Symbol   string
	Side     string // buy | sell
	Quantity float64
}

// Create opens a new paper portfolio with starting cash.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Portfolio, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	if in.StartingBalance < domain.MinStartingBalance || in.StartingBalance > domain.MaxStartingBalance ||
		math.IsNaN(in.StartingBalance) || math.IsInf(in.StartingBalance, 0) {
		return nil, fmt.Errorf("%w: startingBalance must be between %g and %g", domain.ErrInvalidArgument, domain.MinStartingBalance, domain.MaxStartingBalance)
	}
	cur := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cur == "" {
		cur = domain.DefaultPaperCurrency
	}
	if _, err := s.store.GetPortfolio(ctx, clientID); err == nil {
		return nil, fmt.Errorf("%w: portfolio already exists for this clientId", domain.ErrInvalidArgument)
	} else if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	now := time.Now().UTC()
	return s.store.CreatePortfolio(ctx, domain.Portfolio{
		ClientID:         clientID,
		Currency:         cur,
		StartingBalance:  in.StartingBalance,
		CashBalance:      in.StartingBalance,
		RealizedPnLTotal: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
}

// Get returns portfolio row or ErrNotFound.
func (s *Service) Get(ctx context.Context, clientID string) (*domain.Portfolio, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.GetPortfolio(ctx, clientID)
}

// View returns cash, positions marked to market, and P&L summary.
func (s *Service) View(ctx context.Context, clientID string) (*domain.PortfolioView, error) {
	p, err := s.Get(ctx, clientID)
	if err != nil {
		return nil, err
	}
	positions, err := s.store.ListPositions(ctx, p.ClientID)
	if err != nil {
		return nil, err
	}
	views := make([]domain.PositionView, 0, len(positions))
	var posValue, unreal float64
	for _, pos := range positions {
		mark, merr := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
		if merr != nil {
			// Skip mark on error — still show cost basis; mark=avg for display safety
			mark = pos.AvgCost
		}
		mv := pos.Quantity * mark
		u := domain.UnrealizedPnL(pos.Quantity, pos.AvgCost, mark)
		views = append(views, domain.PositionView{
			Exchange:      pos.Exchange,
			Symbol:        pos.Symbol,
			Quantity:      pos.Quantity,
			AvgCost:       pos.AvgCost,
			MarkPrice:     mark,
			MarketValue:   mv,
			UnrealizedPnL: u,
			CostBasis:     pos.Quantity * pos.AvgCost,
		})
		posValue += mv
		unreal += u
	}
	equity := p.CashBalance + posValue
	return &domain.PortfolioView{
		ClientID:         p.ClientID,
		Currency:         p.Currency,
		StartingBalance:  p.StartingBalance,
		CashBalance:      p.CashBalance,
		PositionsValue:   posValue,
		Equity:           equity,
		UnrealizedPnL:    unreal,
		RealizedPnLTotal: p.RealizedPnLTotal,
		TotalPnL:         equity - p.StartingBalance,
		Positions:        views,
		Note:             paperNote,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}, nil
}

// PlaceOrder executes a paper market order at last trade price.
func (s *Service) PlaceOrder(ctx context.Context, in OrderInput) (*domain.Trade, *domain.PortfolioView, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, nil, err
	}
	side := domain.TradeSide(strings.ToLower(strings.TrimSpace(in.Side)))
	if !domain.IsValidTradeSide(string(side)) {
		return nil, nil, fmt.Errorf("%w: side must be buy or sell", domain.ErrInvalidArgument)
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.store.GetPortfolio(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	price, err := s.lastPrice(ctx, string(ex), sym)
	if err != nil {
		return nil, nil, err
	}

	var posQty, avg float64
	pos, perr := s.store.GetPosition(ctx, clientID, ex, sym)
	if perr == nil && pos != nil {
		posQty, avg = pos.Quantity, pos.AvgCost
	} else if perr != nil && perr != domain.ErrNotFound {
		return nil, nil, perr
	}

	now := time.Now().UTC()
	var (
		newCash, newQty, newAvg, realized float64
	)
	switch side {
	case domain.TradeSideBuy:
		newCash, newQty, newAvg, err = domain.ApplyBuy(p.CashBalance, in.Quantity, price, posQty, avg)
		realized = 0
	case domain.TradeSideSell:
		newCash, newQty, realized, err = domain.ApplySell(p.CashBalance, in.Quantity, price, posQty, avg)
		newAvg = avg
	}
	if err != nil {
		return nil, nil, err
	}

	p.CashBalance = newCash
	p.RealizedPnLTotal += realized
	p.UpdatedAt = now

	posOut := &domain.Position{
		ClientID:  clientID,
		Exchange:  ex,
		Symbol:    sym,
		Quantity:  newQty,
		AvgCost:   newAvg,
		UpdatedAt: now,
	}
	if newQty <= domain.PositionEpsilon {
		posOut.Quantity = 0
		posOut.AvgCost = 0
	}

	tr := domain.Trade{
		ID:          uuid.NewString(),
		ClientID:    clientID,
		Exchange:    ex,
		Symbol:      sym,
		Side:        side,
		Quantity:    in.Quantity,
		Price:       price,
		Notional:    in.Quantity * price,
		RealizedPnL: realized,
		CreatedAt:   now,
	}
	if err := s.store.ExecuteTrade(ctx, p, posOut, tr); err != nil {
		return nil, nil, err
	}
	view, err := s.View(ctx, clientID)
	if err != nil {
		return &tr, nil, err
	}
	return &tr, view, nil
}

// ListTrades returns trade history.
func (s *Service) ListTrades(ctx context.Context, clientID string, limit, offset int) ([]domain.Trade, int, error) {
	if s.store == nil {
		return nil, 0, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	// Ensure portfolio exists
	if _, err := s.store.GetPortfolio(ctx, clientID); err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountTrades(ctx, clientID)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListTrades(ctx, clientID, limit, offset)
	return list, total, err
}

func (s *Service) lastPrice(ctx context.Context, exchange, symbol string) (float64, error) {
	tkr, err := s.market.GetTicker24h(ctx, exchange, symbol)
	if err != nil {
		return 0, err
	}
	if tkr == nil || tkr.LastPrice == "" {
		return 0, fmt.Errorf("%w: last price unavailable", domain.ErrUpstream)
	}
	p, err := strconv.ParseFloat(tkr.LastPrice, 64)
	if err != nil || p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
		return 0, fmt.Errorf("%w: invalid last price", domain.ErrUpstream)
	}
	return p, nil
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	if strings.EqualFold(id, "default") {
		return "", fmt.Errorf("%w: clientId must not be the shared name \"default\"", domain.ErrInvalidArgument)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: clientId has invalid characters", domain.ErrInvalidArgument)
	}
	return id, nil
}

func normalizeExchangeSymbol(exchange, symbol string) (domain.Exchange, string, error) {
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return "", "", fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return "", "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return ex, sym, nil
}