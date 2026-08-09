package portfolio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// AllocationBasketCreateInput creates a named target mix.
type AllocationBasketCreateInput struct {
	ClientID    string
	PortfolioID string
	Name        string
	Targets     []domain.AllocationTarget
}

// AllocationBasketUpdateInput replaces name and/or targets.
type AllocationBasketUpdateInput struct {
	ClientID    string
	PortfolioID string
	BasketID    string
	Name        *string
	Targets     []domain.AllocationTarget
}

// AllocationBasketView is a basket plus live drift (no trades until rebalance).
type AllocationBasketView struct {
	Basket domain.AllocationBasket
	Plan   domain.RebalancePlan
	Note   string
}

// CreateAllocationBasket stores a new named basket. Does not trade.
func (s *Service) CreateAllocationBasket(ctx context.Context, in AllocationBasketCreateInput) (*domain.AllocationBasket, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireBook(ctx, in.ClientID, in.PortfolioID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
	name, err := domain.NormalizeAllocationName(in.Name)
	if err != nil {
		return nil, err
	}
	targets, err := normalizeAllocationTargets(in.Targets, p.Currency)
	if err != nil {
		return nil, err
	}
	n, err := s.store.CountAllocationBaskets(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxAllocationBasketsPerClient {
		return nil, fmt.Errorf("%w: max %d allocation baskets per client", domain.ErrInvalidArgument, domain.MaxAllocationBasketsPerClient)
	}
	now := time.Now().UTC()
	b := domain.AllocationBasket{
		ID: uuid.NewString(), ClientID: clientID, Name: name, Targets: targets,
		CreatedAt: now, UpdatedAt: now,
	}
	return s.store.CreateAllocationBasket(ctx, b)
}

// UpdateAllocationBasket changes name and/or targets. Does not trade.
func (s *Service) UpdateAllocationBasket(ctx context.Context, in AllocationBasketUpdateInput) (*domain.AllocationBasket, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireBook(ctx, in.ClientID, in.PortfolioID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
	id := strings.TrimSpace(in.BasketID)
	if id == "" {
		return nil, fmt.Errorf("%w: basket id is required", domain.ErrInvalidArgument)
	}
	cur, err := s.store.GetAllocationBasket(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	name := cur.Name
	if in.Name != nil {
		name, err = domain.NormalizeAllocationName(*in.Name)
		if err != nil {
			return nil, err
		}
	}
	targets := cur.Targets
	if in.Targets != nil {
		targets, err = normalizeAllocationTargets(in.Targets, p.Currency)
		if err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	cur.Name = name
	cur.Targets = targets
	cur.UpdatedAt = now
	return s.store.UpdateAllocationBasket(ctx, clientID, id, *cur)
}

// ListAllocationBaskets lists saved baskets (no live marks).
func (s *Service) ListAllocationBaskets(ctx context.Context, clientID string, portfolioID ...string) ([]domain.AllocationBasket, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return nil, err
	}
	return s.store.ListAllocationBaskets(ctx, p.BookID())
}

// GetAllocationBasket returns one basket.
func (s *Service) GetAllocationBasket(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.AllocationBasket, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: basket id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetAllocationBasket(ctx, p.BookID(), id)
}

// DeleteAllocationBasket removes a saved mix. Does not trade.
func (s *Service) DeleteAllocationBasket(ctx context.Context, clientID, id string, portfolioID ...string) error {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return err
	}
	clientID = p.BookID()
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: basket id is required", domain.ErrInvalidArgument)
	}
	return s.store.DeleteAllocationBasket(ctx, clientID, id)
}

// PreviewAllocationRebalance shows drift and proposed market legs. No trades.
func (s *Service) PreviewAllocationRebalance(ctx context.Context, clientID, basketID string, portfolioID ...string) (*AllocationBasketView, error) {
	plan, basket, err := s.buildAllocationPlan(ctx, clientID, basketID, portfolioID...)
	if err != nil {
		return nil, err
	}
	return &AllocationBasketView{
		Basket: *basket, Plan: plan,
		Note: "Preview only — drift is allowed. Rebalance runs solely when you confirm. Paper trading; not real money.",
	}, nil
}

// ExecuteAllocationRebalance places market sells then buys to move toward targets.
// Manual only: nothing auto-rebalances in the background.
func (s *Service) ExecuteAllocationRebalance(ctx context.Context, clientID, basketID string, portfolioID ...string) (*AllocationBasketView, []domain.Trade, error) {
	plan, basket, err := s.buildAllocationPlan(ctx, clientID, basketID, portfolioID...)
	if err != nil {
		return nil, nil, err
	}
	pfID := ""
	if len(portfolioID) > 0 {
		pfID = portfolioID[0]
	}
	var trades []domain.Trade
	var failed []string
	for _, leg := range plan.Legs {
		tr, _, err := s.PlaceOrder(ctx, OrderInput{
			ClientID: clientID, PortfolioID: pfID, Exchange: string(leg.Exchange), Symbol: leg.Symbol,
			Side: string(leg.Side), Quantity: leg.Quantity,
		})
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s %s: %s", leg.Side, leg.Asset, err.Error()))
			continue
		}
		if tr != nil {
			trades = append(trades, *tr)
		}
	}
	view, err := s.PreviewAllocationRebalance(ctx, clientID, basketID, portfolioID...)
	if err != nil {
		return nil, trades, err
	}
	view.Note = "Rebalance executed at last prices. Drift is allowed until you run rebalance again. Paper trading; not real money."
	if len(failed) > 0 {
		view.Note += " Some legs skipped: " + strings.Join(failed, "; ")
	}
	if len(trades) == 0 && len(plan.Legs) == 0 {
		view.Note = "Already near targets (or deltas below min notional). No trades. Paper trading; not real money."
	}
	_ = basket
	return view, trades, nil
}

func (s *Service) buildAllocationPlan(ctx context.Context, clientID, basketID string, portfolioID ...string) (domain.RebalancePlan, *domain.AllocationBasket, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return domain.RebalancePlan{}, nil, err
	}
	clientID = p.BookID()
	basketID = strings.TrimSpace(basketID)
	if basketID == "" {
		return domain.RebalancePlan{}, nil, fmt.Errorf("%w: basket id is required", domain.ErrInvalidArgument)
	}
	b, err := s.store.GetAllocationBasket(ctx, clientID, basketID)
	if err != nil {
		return domain.RebalancePlan{}, nil, err
	}
	view, err := s.View(ctx, p.ClientID, p.ID)
	if err != nil {
		return domain.RebalancePlan{}, nil, err
	}
	quote := domain.NormalizeAllocationAsset(p.Currency)
	var holdings []domain.AllocationHolding
	var spotValue float64
	for _, pos := range view.Positions {
		base, q := domain.SplitBaseQuote(pos.Exchange, pos.Symbol)
		if q != "" && q != quote {
			continue // only quote-currency pairs count toward the mix
		}
		if base == "" {
			continue
		}
		holdings = append(holdings, domain.AllocationHolding{
			Asset: base, Exchange: pos.Exchange, Symbol: pos.Symbol,
			MarkPrice: pos.MarkPrice, Quantity: pos.Quantity, AvailableQty: pos.AvailableQuantity,
			MarketValue: pos.MarketValue,
		})
		spotValue += pos.MarketValue
	}
	// Fetch marks for target coins not currently held.
	have := map[string]struct{}{}
	for _, h := range holdings {
		have[h.Asset] = struct{}{}
	}
	for _, t := range b.Targets {
		a := domain.NormalizeAllocationAsset(t.Asset)
		if a == quote {
			continue
		}
		if _, ok := have[a]; ok {
			continue
		}
		ex := t.Exchange
		if ex == "" {
			ex = domain.DefaultExchange
		}
		sym := domain.PairSymbol(ex, a, quote)
		px, err := s.lastPrice(ctx, string(ex), sym)
		if err != nil || px <= 0 {
			return domain.RebalancePlan{}, nil, fmt.Errorf("%w: mark price unavailable for %s", domain.ErrUpstream, sym)
		}
		holdings = append(holdings, domain.AllocationHolding{
			Asset: a, Exchange: ex, Symbol: sym, MarkPrice: px,
		})
	}
	equity := view.CashBalance + spotValue
	plan, err := domain.PlanRebalance(quote, equity, view.CashBalance, view.AvailableCash, holdings, b.Targets)
	if err != nil {
		return domain.RebalancePlan{}, nil, err
	}
	return plan, b, nil
}

func normalizeAllocationTargets(in []domain.AllocationTarget, currency string) ([]domain.AllocationTarget, error) {
	out := make([]domain.AllocationTarget, 0, len(in))
	for _, t := range in {
		a := domain.NormalizeAllocationAsset(t.Asset)
		ex := t.Exchange
		if !domain.IsCashAllocation(a, currency) {
			if string(ex) == "" {
				ex = domain.DefaultExchange
			} else if !domain.IsValidExchange(string(ex)) {
				return nil, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
			} else {
				ex = domain.ParseExchange(string(ex))
			}
		} else {
			ex = ""
		}
		out = append(out, domain.AllocationTarget{Asset: a, Exchange: ex, WeightPct: t.WeightPct})
	}
	if err := domain.ValidateAllocationTargets(out, currency); err != nil {
		return nil, err
	}
	return out, nil
}
