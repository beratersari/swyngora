package portfolio

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
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
	// clientMu serializes mutations per clientId so concurrent market orders,
	// pending placements, and filler ticks cannot corrupt cash/positions.
	clientMu sync.Map // map[string]*sync.Mutex
}

// New constructs a portfolio service.
func New(store domain.PortfolioPort, market PriceFetcher) *Service {
	return &Service{store: store, market: market}
}

// lockClient returns an unlock function; always call via defer unlock().
func (s *Service) lockClient(clientID string) func() {
	v, _ := s.clientMu.LoadOrStore(clientID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
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
	unlock := s.lockClient(clientID)
	defer unlock()
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

// View returns cash, reservations, positions marked to market, and P&L summary.
func (s *Service) View(ctx context.Context, clientID string) (*domain.PortfolioView, error) {
	p, err := s.Get(ctx, clientID)
	if err != nil {
		return nil, err
	}
	reservedCash, err := s.store.SumReservedCash(ctx, p.ClientID)
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
		resQty, rerr := s.store.SumReservedQuantity(ctx, p.ClientID, pos.Exchange, pos.Symbol)
		if rerr != nil {
			return nil, rerr
		}
		mv := pos.Quantity * mark
		u := domain.UnrealizedPnL(pos.Quantity, pos.AvgCost, mark)
		views = append(views, domain.PositionView{
			Exchange:          pos.Exchange,
			Symbol:            pos.Symbol,
			Quantity:          pos.Quantity,
			ReservedQuantity:  resQty,
			AvailableQuantity: domain.AvailablePosition(pos.Quantity, resQty),
			AvgCost:           pos.AvgCost,
			MarkPrice:         mark,
			MarketValue:       mv,
			UnrealizedPnL:     u,
			CostBasis:         pos.Quantity * pos.AvgCost,
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
		ReservedCash:     reservedCash,
		AvailableCash:    domain.AvailableCash(p.CashBalance, reservedCash),
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
	// Fetch price outside the client lock so market I/O does not serialize all clients.
	price, err := s.lastPrice(ctx, string(ex), sym)
	if err != nil {
		return nil, nil, err
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	p, err := s.store.GetPortfolio(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	reservedCash, err := s.store.SumReservedCash(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	availCash := domain.AvailableCash(p.CashBalance, reservedCash)

	var posQty, avg float64
	pos, perr := s.store.GetPosition(ctx, clientID, ex, sym)
	if perr == nil && pos != nil {
		posQty, avg = pos.Quantity, pos.AvgCost
	} else if perr != nil && perr != domain.ErrNotFound {
		return nil, nil, perr
	}
	reservedQty, err := s.store.SumReservedQuantity(ctx, clientID, ex, sym)
	if err != nil {
		return nil, nil, err
	}
	availPos := domain.AvailablePosition(posQty, reservedQty)

	now := time.Now().UTC()
	var (
		newCash, newQty, newAvg, realized float64
	)
	switch side {
	case domain.TradeSideBuy:
		// Spend only unreserved cash; update absolute cash from full balance.
		_, newQty, newAvg, err = domain.ApplyBuy(availCash, in.Quantity, price, posQty, avg)
		if err == nil {
			newCash = p.CashBalance - in.Quantity*price
		}
		realized = 0
	case domain.TradeSideSell:
		// Sell only unreserved quantity.
		_, _, realized, err = domain.ApplySell(0, in.Quantity, price, availPos, avg)
		if err == nil {
			newCash = p.CashBalance + in.Quantity*price
			newQty = posQty - in.Quantity
			if newQty < domain.PositionEpsilon {
				newQty = 0
			}
			newAvg = avg
		}
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
	// Snapshot while still holding the client lock (marks may use market I/O).
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

// PendingOrderInput creates a limit or stop resting order.
type PendingOrderInput struct {
	ClientID     string
	Exchange     string
	Symbol       string
	Type         string // limit_buy | limit_sell | stop_loss
	Quantity     float64
	TriggerPrice float64
	TimeInForce  string     // gtc (default) | ioc | fok
	ExpiresAt    *time.Time // optional; GTC only
}

// PlacePendingOrder creates an open resting order and reserves cash or position.
func (s *Service) PlacePendingOrder(ctx context.Context, in PendingOrderInput) (*domain.PendingOrder, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	typ := domain.PendingOrderType(strings.ToLower(strings.TrimSpace(in.Type)))
	if !domain.IsValidPendingOrderType(string(typ)) {
		return nil, fmt.Errorf("%w: type must be limit_buy, limit_sell, or stop_loss", domain.ErrInvalidArgument)
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	if in.TriggerPrice < domain.MinTriggerPrice || in.TriggerPrice > domain.MaxTriggerPrice ||
		math.IsNaN(in.TriggerPrice) || math.IsInf(in.TriggerPrice, 0) {
		return nil, fmt.Errorf("%w: triggerPrice out of range", domain.ErrInvalidArgument)
	}
	tif, err := domain.NormalizeTimeInForce(in.TimeInForce)
	if err != nil {
		return nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	p, err := s.store.GetPortfolio(ctx, clientID)
	if err != nil {
		return nil, err
	}
	n, err := s.store.CountOpenPendingOrders(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxOpenPendingOrders {
		return nil, fmt.Errorf("%w: max open pending orders (%d) reached", domain.ErrInvalidArgument, domain.MaxOpenPendingOrders)
	}

	now := time.Now().UTC()
	var expiresAt *time.Time
	if in.ExpiresAt != nil && !in.ExpiresAt.IsZero() {
		if tif != domain.TimeInForceGTC {
			return nil, fmt.Errorf("%w: expiresAt is only valid for gtc orders", domain.ErrInvalidArgument)
		}
		exp := in.ExpiresAt.UTC()
		if !exp.After(now) {
			return nil, fmt.Errorf("%w: expiresAt must be in the future", domain.ErrInvalidArgument)
		}
		expiresAt = &exp
	}

	side := domain.SideForPendingType(typ)
	var reservedCash, reservedQty float64
	switch side {
	case domain.TradeSideBuy:
		need := domain.BuyReserveCash(in.Quantity, in.TriggerPrice)
		resCash, rerr := s.store.SumReservedCash(ctx, clientID)
		if rerr != nil {
			return nil, rerr
		}
		avail := domain.AvailableCash(p.CashBalance, resCash)
		if avail+1e-9 < need {
			return nil, fmt.Errorf("%w: insufficient available cash to reserve (need %g, available %g)", domain.ErrInvalidArgument, need, avail)
		}
		reservedCash = need
	case domain.TradeSideSell:
		var held float64
		pos, perr := s.store.GetPosition(ctx, clientID, ex, sym)
		if perr == nil && pos != nil {
			held = pos.Quantity
		} else if perr != nil && perr != domain.ErrNotFound {
			return nil, perr
		}
		resQty, rerr := s.store.SumReservedQuantity(ctx, clientID, ex, sym)
		if rerr != nil {
			return nil, rerr
		}
		avail := domain.AvailablePosition(held, resQty)
		if avail+domain.PositionEpsilon < in.Quantity {
			return nil, fmt.Errorf("%w: insufficient available position to reserve (need %g, available %g)", domain.ErrInvalidArgument, in.Quantity, avail)
		}
		reservedQty = in.Quantity
	}

	o := domain.PendingOrder{
		ID:                uuid.NewString(),
		ClientID:          clientID,
		Exchange:          ex,
		Symbol:            sym,
		Type:              typ,
		Side:              side,
		Quantity:          in.Quantity,
		FilledQuantity:    0,
		RemainingQuantity: in.Quantity,
		TriggerPrice:      in.TriggerPrice,
		ReservedCash:      reservedCash,
		ReservedQuantity:  reservedQty,
		TimeInForce:       tif,
		ExpiresAt:         expiresAt,
		Status:            domain.PendingStatusOpen,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return s.store.CreatePendingOrder(ctx, o)
}

// ListPendingOrders returns pending orders for a client (default: open only).
func (s *Service) ListPendingOrders(ctx context.Context, clientID string, status string, limit, offset int) ([]domain.PendingOrder, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.GetPortfolio(ctx, clientID); err != nil {
		return nil, err
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
	st := domain.PendingOrderStatus(strings.ToLower(strings.TrimSpace(status)))
	if st == "" {
		st = domain.PendingStatusOpen
	}
	if st == "all" {
		st = ""
	} else {
		switch st {
		case domain.PendingStatusOpen, domain.PendingStatusFilled, domain.PendingStatusCanceled, domain.PendingStatusRejected:
		default:
			return nil, fmt.Errorf("%w: status must be open, filled, canceled, rejected, or all", domain.ErrInvalidArgument)
		}
	}
	return s.store.ListPendingOrders(ctx, clientID, st, limit, offset)
}

// CancelPendingOrder cancels an open order; filled/canceled/rejected cannot be canceled.
func (s *Service) CancelPendingOrder(ctx context.Context, clientID, id string) (*domain.PendingOrder, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	// Ensure portfolio exists for clearer 404 vs order 404
	if _, err := s.store.GetPortfolio(ctx, clientID); err != nil {
		return nil, err
	}
	return s.store.CancelPendingOrder(ctx, clientID, id, time.Now().UTC(), domain.CancelReasonUser)
}

// ListAllOpenPendingOrders is used by the background order filler.
func (s *Service) ListAllOpenPendingOrders(ctx context.Context) ([]domain.PendingOrder, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	return s.store.ListAllOpenPendingOrders(ctx)
}

// ProcessOpenOrder runs expiry checks and one fill attempt according to time-in-force.
// Returns the order when something changed (fill and/or cancel), ok=true; otherwise (nil,false,nil).
func (s *Service) ProcessOpenOrder(ctx context.Context, o domain.PendingOrder, lastPrice float64, now time.Time, maxFillQty float64) (*domain.PendingOrder, bool, error) {
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	if o.Status != domain.PendingStatusOpen {
		return nil, false, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	// Serialize all mutations for this client (filler runs multi-symbol in parallel).
	unlock := s.lockClient(o.ClientID)
	defer unlock()
	return s.processOpenOrderLocked(ctx, o, lastPrice, now, maxFillQty)
}

func (s *Service) processOpenOrderLocked(ctx context.Context, o domain.PendingOrder, lastPrice float64, now time.Time, maxFillQty float64) (*domain.PendingOrder, bool, error) {
	// Re-load order under lock to avoid acting on a stale snapshot.
	cur, err := s.store.GetPendingOrder(ctx, o.ClientID, o.ID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if cur == nil || cur.Status != domain.PendingStatusOpen {
		return nil, false, nil
	}
	o = *cur

	tif := o.TimeInForce
	if tif == "" {
		tif = domain.TimeInForceGTC
	}

	// GTC expiration: cancel and release reservation.
	if tif == domain.TimeInForceGTC && domain.PendingOrderExpired(o, now) {
		canceled, err := s.store.CancelPendingOrder(ctx, o.ClientID, o.ID, now, domain.CancelReasonExpired)
		if err != nil {
			if err == domain.ErrNotFound {
				return nil, false, nil
			}
			return nil, false, err
		}
		return canceled, true, nil
	}

	triggered := domain.PendingOrderTriggered(o.Type, o.TriggerPrice, lastPrice)
	if !triggered {
		// IOC/FOK: one immediate attempt — if not marketable, cancel with no fill.
		switch tif {
		case domain.TimeInForceIOC:
			canceled, err := s.store.CancelPendingOrder(ctx, o.ClientID, o.ID, now, domain.CancelReasonIOCNoFill)
			if err != nil {
				if err == domain.ErrNotFound {
					return nil, false, nil
				}
				return nil, false, err
			}
			return canceled, true, nil
		case domain.TimeInForceFOK:
			canceled, err := s.store.CancelPendingOrder(ctx, o.ClientID, o.ID, now, domain.CancelReasonFOKUnfilled)
			if err != nil {
				if err == domain.ErrNotFound {
					return nil, false, nil
				}
				return nil, false, err
			}
			return canceled, true, nil
		default:
			return nil, false, nil
		}
	}

	// FOK: require full remaining size in one shot.
	if tif == domain.TimeInForceFOK {
		remaining := o.RemainingQuantity
		if remaining <= domain.PositionEpsilon {
			remaining = o.Quantity - o.FilledQuantity
		}
		can := s.maxFillableQty(o, remaining, lastPrice)
		if can+domain.PositionEpsilon < remaining {
			canceled, err := s.store.CancelPendingOrder(ctx, o.ClientID, o.ID, now, domain.CancelReasonFOKUnfilled)
			if err != nil {
				if err == domain.ErrNotFound {
					return nil, false, nil
				}
				return nil, false, err
			}
			return canceled, true, nil
		}
		// Force full fill (no partial cap).
		maxFillQty = 0
	}

	filled, ok, err := s.tryFillPendingOrderLocked(ctx, o, lastPrice, maxFillQty)
	if err != nil {
		return nil, false, err
	}

	// IOC: after first try, cancel any remainder and release reservation.
	if tif == domain.TimeInForceIOC {
		if ok && filled != nil && filled.Status == domain.PendingStatusOpen && filled.RemainingQuantity > domain.PositionEpsilon {
			canceled, cerr := s.store.CancelPendingOrder(ctx, filled.ClientID, filled.ID, now, domain.CancelReasonIOCRemainder)
			if cerr != nil {
				if cerr == domain.ErrNotFound {
					return filled, true, nil
				}
				return nil, false, cerr
			}
			return canceled, true, nil
		}
		if !ok {
			canceled, cerr := s.store.CancelPendingOrder(ctx, o.ClientID, o.ID, now, domain.CancelReasonIOCNoFill)
			if cerr != nil {
				if cerr == domain.ErrNotFound {
					return nil, false, nil
				}
				return nil, false, cerr
			}
			return canceled, true, nil
		}
	}
	return filled, ok, nil
}

func (s *Service) maxFillableQty(o domain.PendingOrder, remaining, lastPrice float64) float64 {
	switch o.Side {
	case domain.TradeSideBuy:
		return domain.MaxBuyFillQty(remaining, o.ReservedCash, lastPrice)
	case domain.TradeSideSell:
		maxByRes := o.ReservedQuantity
		if maxByRes <= 0 {
			maxByRes = remaining
		}
		if maxByRes > remaining {
			maxByRes = remaining
		}
		return maxByRes
	default:
		return 0
	}
}

// TryFillPendingOrder evaluates one open order against lastPrice and applies a fill.
// maxFillQty > 0 caps this execution (partial fill); <= 0 fills as much remaining as possible.
// Returns (order, true, nil) when any quantity was filled; (nil, false, nil) when not triggered / no-op.
// Prefer ProcessOpenOrder for TIF/expiry handling.
func (s *Service) TryFillPendingOrder(ctx context.Context, o domain.PendingOrder, lastPrice, maxFillQty float64) (*domain.PendingOrder, bool, error) {
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	if o.Status != domain.PendingStatusOpen {
		return nil, false, nil
	}
	unlock := s.lockClient(o.ClientID)
	defer unlock()
	// Prefer fresh row under the lock.
	if cur, err := s.store.GetPendingOrder(ctx, o.ClientID, o.ID); err == nil && cur != nil {
		o = *cur
	} else if err != nil && err != domain.ErrNotFound {
		return nil, false, err
	}
	return s.tryFillPendingOrderLocked(ctx, o, lastPrice, maxFillQty)
}

func (s *Service) tryFillPendingOrderLocked(ctx context.Context, o domain.PendingOrder, lastPrice, maxFillQty float64) (*domain.PendingOrder, bool, error) {
	if o.Status != domain.PendingStatusOpen {
		return nil, false, nil
	}
	remaining := o.RemainingQuantity
	if remaining <= domain.PositionEpsilon {
		remaining = o.Quantity - o.FilledQuantity
	}
	if remaining <= domain.PositionEpsilon {
		return nil, false, nil
	}
	if !domain.PendingOrderTriggered(o.Type, o.TriggerPrice, lastPrice) {
		return nil, false, nil
	}
	now := time.Now().UTC()
	p, err := s.store.GetPortfolio(ctx, o.ClientID)
	if err != nil {
		return nil, false, err
	}
	var posQty, avg float64
	pos, perr := s.store.GetPosition(ctx, o.ClientID, o.Exchange, o.Symbol)
	if perr == nil && pos != nil {
		posQty, avg = pos.Quantity, pos.AvgCost
	} else if perr != nil && perr != domain.ErrNotFound {
		return nil, false, perr
	}

	// How much can we fill this pass?
	fillQty := domain.ClampFillQty(remaining, 0, maxFillQty)
	switch o.Side {
	case domain.TradeSideBuy:
		// Bound by remaining reserved cash at fill price.
		maxByRes := domain.MaxBuyFillQty(remaining, o.ReservedCash, lastPrice)
		if maxFillQty > 0 && maxFillQty < maxByRes {
			fillQty = maxFillQty
		} else {
			fillQty = maxByRes
		}
		fillQty = domain.ClampFillQty(remaining, fillQty, 0)
	case domain.TradeSideSell:
		// Bound by reserved quantity remaining.
		maxByRes := o.ReservedQuantity
		if maxByRes <= 0 {
			maxByRes = remaining
		}
		if maxByRes > remaining {
			maxByRes = remaining
		}
		fillQty = domain.ClampFillQty(maxByRes, 0, maxFillQty)
	default:
		return nil, false, fmt.Errorf("%w: invalid order side", domain.ErrInvalidArgument)
	}
	if fillQty < domain.MinTradeQuantity {
		// Cannot fill any this pass — leave open for GTC; IOC/FOK handled by ProcessOpenOrder.
		return nil, false, nil
	}

	var newCash, newQty, newAvg, realized float64
	switch o.Side {
	case domain.TradeSideBuy:
		newCash, newQty, newAvg, err = domain.ApplyBuy(p.CashBalance, fillQty, lastPrice, posQty, avg)
		realized = 0
	case domain.TradeSideSell:
		newCash, newQty, realized, err = domain.ApplySell(p.CashBalance, fillQty, lastPrice, posQty, avg)
		newAvg = avg
	}
	if err != nil {
		// Reservation should prevent this; fail closed and release remainder.
		reason := err.Error()
		if rerr := s.store.RejectPendingOrder(ctx, o.ID, reason, now); rerr != nil {
			if rerr == domain.ErrNotFound {
				return nil, false, nil
			}
			return nil, false, rerr
		}
		return nil, false, nil
	}

	remainingAfter := remaining - fillQty
	if remainingAfter < domain.PositionEpsilon {
		remainingAfter = 0
	}
	filledAfter := o.FilledQuantity + fillQty
	updated := o
	updated.FilledQuantity = filledAfter
	updated.RemainingQuantity = remainingAfter
	updated.FillTradeID = "" // set after trade id known
	updated.FillPrice = lastPrice
	updated.UpdatedAt = now
	if remainingAfter <= domain.PositionEpsilon {
		updated.Status = domain.PendingStatusFilled
		updated.FilledAt = &now
		updated.ReservedCash = 0
		updated.ReservedQuantity = 0
		updated.RemainingQuantity = 0
	} else {
		updated.Status = domain.PendingStatusOpen
		if o.Side == domain.TradeSideBuy {
			updated.ReservedCash = domain.AfterBuyFillReservation(remainingAfter, o.TriggerPrice)
			updated.ReservedQuantity = 0
		} else {
			updated.ReservedCash = 0
			updated.ReservedQuantity = domain.AfterSellFillReservation(remainingAfter)
		}
	}

	p.CashBalance = newCash
	p.RealizedPnLTotal += realized
	p.UpdatedAt = now
	posOut := &domain.Position{
		ClientID: o.ClientID, Exchange: o.Exchange, Symbol: o.Symbol,
		Quantity: newQty, AvgCost: newAvg, UpdatedAt: now,
	}
	if newQty <= domain.PositionEpsilon {
		posOut.Quantity = 0
		posOut.AvgCost = 0
	}
	tr := domain.Trade{
		ID: uuid.NewString(), ClientID: o.ClientID, Exchange: o.Exchange, Symbol: o.Symbol,
		Side: o.Side, Quantity: fillQty, Price: lastPrice, Notional: fillQty * lastPrice,
		RealizedPnL: realized, PendingOrderID: o.ID, CreatedAt: now,
	}
	updated.FillTradeID = tr.ID
	if err := s.store.ExecutePendingFill(ctx, &updated, p, posOut, tr, now); err != nil {
		if err == domain.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	got, gerr := s.store.GetPendingOrder(ctx, o.ClientID, o.ID)
	if gerr != nil {
		return &updated, true, nil
	}
	return got, true, nil
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