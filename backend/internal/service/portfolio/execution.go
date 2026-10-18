package portfolio

import (
	"context"
	"fmt"
	"math"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ExecutionCompareInput prices three ways to work one spot size. Nothing is written.
// Exchange is the single venue. Best cost and the time split read Binance, Coinbase, and Bybit.
type ExecutionCompareInput struct {
	ClientID       string
	PortfolioID    string
	OwnerClientID  string
	Exchange       string
	Symbol         string
	Side           string
	Quantity       float64
	MaxSlippagePct float64
	SpreadMinutes  int
	SliceCount     int
}

// CompareExecutions prices a single-venue market order, best cost, and a time split.
// The time split assumes each slice finds the current book again. A bad schedule still returns the other two prices.
func (s *Service) CompareExecutions(ctx context.Context, in ExecutionCompareInput) (*domain.ExecutionCompare, error) {
	if s.store == nil || s.market == nil {
		return nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleViewer, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, err
	}
	if err := s.rejectClosedOwner(ctx, p); err != nil {
		return nil, err
	}
	side := domain.TradeSide(strings.ToLower(strings.TrimSpace(in.Side)))
	if !domain.IsValidTradeSide(string(side)) {
		return nil, fmt.Errorf("%w: side must be buy or sell", domain.ErrInvalidArgument)
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	ex := domain.ParseExchange(in.Exchange)
	if !isRouteVenue(ex) {
		return nil, fmt.Errorf("%w: compare binance, coinbase, or bybit", domain.ErrInvalidArgument)
	}
	canon, err := domain.CanonicalRouteSymbol(in.Symbol)
	if err != nil {
		return nil, err
	}
	slip, err := domain.ParseMaxSlippagePct(in.MaxSlippagePct)
	if err != nil {
		return nil, err
	}
	plan, schedule, detail := spreadPlan(in.SpreadMinutes, in.SliceCount)
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	fresh, ferr := s.store.GetPortfolio(ctx, clientID)
	if ferr != nil {
		return nil, ferr
	}
	p = fresh
	if err := domain.RouteSymbolMatchesBook(canon, p.Currency); err != nil {
		return nil, err
	}
	books, err := s.loadRouteBooks(ctx, clientID, canon, p.Currency, side)
	if err != nil {
		return nil, err
	}
	got := domain.CompareExecutions(side, in.Quantity, p.Currency, canon, ex, slip, plan, schedule, detail, in.SpreadMinutes, in.SliceCount, books)
	return &got, nil
}

func isRouteVenue(ex domain.Exchange) bool {
	for _, v := range domain.PaperRouteVenues() {
		if v == ex {
			return true
		}
	}
	return false
}

func spreadPlan(minutes, slices int) (*domain.SpreadSchedule, string, string) {
	got, err := domain.NormalizeSpreadSchedule(minutes, slices)
	if err != nil {
		code := domain.ExecutionScheduleRange
		if strings.Contains(err.Error(), "closer than") {
			code = domain.ExecutionScheduleInterval
		}
		return nil, code, err.Error()
	}
	return &got, "", ""
}
