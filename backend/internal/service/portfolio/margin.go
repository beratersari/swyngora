package portfolio

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// MarginOrderInput opens a leveraged long/short via market or limit.
type MarginOrderInput struct {
	ClientID   string
	Exchange   string
	Symbol     string
	Side       string // long | short
	Type       string // market | limit
	Quantity   float64
	Leverage   int
	LimitPrice float64  // required for limit
	StopLoss   *float64 // optional
	TakeProfit *float64 // optional
}

// MarginCloseInput closes all or part of a margin position at market.
type MarginCloseInput struct {
	ClientID   string
	PositionID string
	Quantity   float64 // 0 = full close
}

// MarginBracketsInput sets or clears stop-loss / take-profit on an open position.
// Omit pointer (nil) to leave unchanged; use ClearStopLoss / ClearTakeProfit to remove.
type MarginBracketsInput struct {
	ClientID        string
	PositionID      string
	StopLoss        *float64
	TakeProfit      *float64
	ClearStopLoss   bool
	ClearTakeProfit bool
}

// availableCashForTrading returns cash free of spot pending + margin limit reservations.
func (s *Service) availableCashForTrading(ctx context.Context, clientID string, cashBalance float64) (float64, error) {
	reservedSpot, err := s.store.SumReservedCash(ctx, clientID)
	if err != nil {
		return 0, err
	}
	reservedMargin, err := s.store.SumReservedMargin(ctx, clientID)
	if err != nil {
		return 0, err
	}
	return domain.AvailableCash(cashBalance, reservedSpot+reservedMargin), nil
}

// PlaceMarginOrder opens a market position immediately or rests a limit open order.
func (s *Service) PlaceMarginOrder(ctx context.Context, in MarginOrderInput) (*domain.MarginPosition, *domain.MarginOrder, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, nil, err
	}
	side, err := domain.NormalizeMarginSide(in.Side)
	if err != nil {
		return nil, nil, err
	}
	typ, err := domain.NormalizeMarginOrderType(in.Type)
	if err != nil {
		return nil, nil, err
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	if !domain.IsValidMarginLeverage(in.Leverage) {
		return nil, nil, fmt.Errorf("%w: leverage must be between %d and %d", domain.ErrInvalidArgument,
			domain.MinMarginLeverage, domain.MaxMarginLeverage)
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.store.GetPortfolio(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	nPos, err := s.store.CountOpenMarginPositions(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	if nPos >= domain.MaxOpenMarginPositions {
		return nil, nil, fmt.Errorf("%w: max open margin positions (%d) reached", domain.ErrInvalidArgument, domain.MaxOpenMarginPositions)
	}

	now := time.Now().UTC()

	if typ == domain.MarginOrderLimit {
		if in.LimitPrice < domain.MinTriggerPrice || in.LimitPrice > domain.MaxTriggerPrice ||
			math.IsNaN(in.LimitPrice) || math.IsInf(in.LimitPrice, 0) {
			return nil, nil, fmt.Errorf("%w: limitPrice out of range", domain.ErrInvalidArgument)
		}
		if err := domain.ValidateMarginBrackets(side, in.LimitPrice, in.StopLoss, in.TakeProfit); err != nil {
			return nil, nil, err
		}
		nOrd, err := s.store.CountOpenMarginOrders(ctx, clientID)
		if err != nil {
			return nil, nil, err
		}
		if nOrd >= domain.MaxOpenMarginOrders {
			return nil, nil, fmt.Errorf("%w: max open margin orders (%d) reached", domain.ErrInvalidArgument, domain.MaxOpenMarginOrders)
		}
		need, err := domain.InitialMargin(in.Quantity, in.LimitPrice, in.Leverage)
		if err != nil {
			return nil, nil, err
		}
		avail, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
		if err != nil {
			return nil, nil, err
		}
		if avail+1e-9 < need {
			return nil, nil, fmt.Errorf("%w: insufficient available cash for margin (need %g, available %g)", domain.ErrInvalidArgument, need, avail)
		}
		o := domain.MarginOrder{
			ID: uuid.NewString(), ClientID: clientID, Exchange: ex, Symbol: sym,
			Side: side, Type: domain.MarginOrderLimit, Quantity: in.Quantity, Leverage: in.Leverage,
			LimitPrice: in.LimitPrice, ReservedMargin: need, StopLoss: in.StopLoss, TakeProfit: in.TakeProfit,
			Status: domain.MarginOrderOpen, CreatedAt: now, UpdatedAt: now,
		}
		out, err := s.store.CreateMarginOrder(ctx, o)
		return nil, out, err
	}

	// Market open
	price, err := s.lastPrice(ctx, string(ex), sym)
	if err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateMarginBrackets(side, price, in.StopLoss, in.TakeProfit); err != nil {
		return nil, nil, err
	}
	margin, err := domain.InitialMargin(in.Quantity, price, in.Leverage)
	if err != nil {
		return nil, nil, err
	}
	avail, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
	if err != nil {
		return nil, nil, err
	}
	if avail+1e-9 < margin {
		return nil, nil, fmt.Errorf("%w: insufficient available cash for margin (need %g, available %g)", domain.ErrInvalidArgument, margin, avail)
	}
	liq, err := domain.LiquidationPriceIsolated(side, price, in.Leverage, domain.DefaultMaintenanceMarginRate)
	if err != nil {
		return nil, nil, err
	}
	pos := domain.MarginPosition{
		ID: uuid.NewString(), ClientID: clientID, Exchange: ex, Symbol: sym, Side: side,
		Quantity: in.Quantity, EntryPrice: price, Leverage: in.Leverage, Margin: margin,
		LiquidationPrice: liq, StopLoss: in.StopLoss, TakeProfit: in.TakeProfit,
		Status: domain.MarginPositionOpen, OpenedAt: now, UpdatedAt: now,
	}
	tr := domain.MarginTrade{
		ID: uuid.NewString(), ClientID: clientID, PositionID: pos.ID, Exchange: ex, Symbol: sym,
		Side: side, Action: "open", Quantity: in.Quantity, Price: price, Notional: in.Quantity * price,
		MarginDelta: -margin, Leverage: in.Leverage, CreatedAt: now,
	}
	p.CashBalance -= margin
	p.UpdatedAt = now
	if err := s.store.ApplyMarginOpen(ctx, p, pos, tr); err != nil {
		return nil, nil, err
	}
	out, err := s.store.GetMarginPosition(ctx, clientID, pos.ID)
	return out, nil, err
}

// ListMarginPositions lists open margin positions with marks.
func (s *Service) ListMarginPositions(ctx context.Context, clientID string) ([]domain.MarginPosition, error) {
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
	list, err := s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		s.markMarginPosition(ctx, &list[i])
	}
	return list, nil
}

// GetMarginPosition returns one open or closed position with mark if open.
func (s *Service) GetMarginPosition(ctx context.Context, clientID, id string) (*domain.MarginPosition, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: position id is required", domain.ErrInvalidArgument)
	}
	pos, err := s.store.GetMarginPosition(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	if pos.Status == domain.MarginPositionOpen {
		s.markMarginPosition(ctx, pos)
	}
	return pos, nil
}

func (s *Service) markMarginPosition(ctx context.Context, pos *domain.MarginPosition) {
	if s.market == nil || pos == nil {
		return
	}
	mark, err := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
	if err != nil || mark <= 0 {
		mark = pos.EntryPrice
	}
	pos.MarkPrice = mark
	pos.UnrealizedPnL = domain.MarginUnrealizedPnL(pos.Side, pos.Quantity, pos.EntryPrice, mark)
}

// CloseMarginPosition closes full or partial size at market.
func (s *Service) CloseMarginPosition(ctx context.Context, in MarginCloseInput) (*domain.MarginPosition, *domain.MarginTrade, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, nil, err
	}
	id := strings.TrimSpace(in.PositionID)
	if id == "" {
		return nil, nil, fmt.Errorf("%w: position id is required", domain.ErrInvalidArgument)
	}
	pos, err := s.store.GetMarginPosition(ctx, clientID, id)
	if err != nil {
		return nil, nil, err
	}
	if pos.Status != domain.MarginPositionOpen {
		return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
	}
	closeQty := in.Quantity
	if closeQty <= 0 {
		closeQty = pos.Quantity
	}
	if closeQty < domain.MinTradeQuantity || closeQty > pos.Quantity+domain.PositionEpsilon ||
		math.IsNaN(closeQty) || math.IsInf(closeQty, 0) {
		return nil, nil, fmt.Errorf("%w: close quantity out of range", domain.ErrInvalidArgument)
	}
	if closeQty > pos.Quantity {
		closeQty = pos.Quantity
	}
	price, err := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
	if err != nil {
		return nil, nil, err
	}
	reason := domain.MarginCloseUser
	if closeQty+domain.PositionEpsilon < pos.Quantity {
		reason = domain.MarginClosePartialUser
	}
	return s.closeMarginAt(ctx, pos, closeQty, price, reason)
}

// SetMarginBrackets updates SL/TP on an open position.
func (s *Service) SetMarginBrackets(ctx context.Context, in MarginBracketsInput) (*domain.MarginPosition, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	pos, err := s.store.GetMarginPosition(ctx, clientID, strings.TrimSpace(in.PositionID))
	if err != nil {
		return nil, err
	}
	if pos.Status != domain.MarginPositionOpen {
		return nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
	}
	sl, tp := pos.StopLoss, pos.TakeProfit
	if in.ClearStopLoss {
		sl = nil
	} else if in.StopLoss != nil {
		sl = in.StopLoss
	}
	if in.ClearTakeProfit {
		tp = nil
	} else if in.TakeProfit != nil {
		tp = in.TakeProfit
	}
	if err := domain.ValidateMarginBrackets(pos.Side, pos.EntryPrice, sl, tp); err != nil {
		return nil, err
	}
	pos.StopLoss, pos.TakeProfit = sl, tp
	pos.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMarginPosition(ctx, *pos); err != nil {
		return nil, err
	}
	return s.GetMarginPosition(ctx, clientID, pos.ID)
}

// ListMarginOrders lists margin open orders.
func (s *Service) ListMarginOrders(ctx context.Context, clientID, status string, limit, offset int) ([]domain.MarginOrder, error) {
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
	var st domain.MarginOrderStatus
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "open":
		st = domain.MarginOrderOpen
	case "all":
		st = ""
	case "filled":
		st = domain.MarginOrderFilled
	case "canceled":
		st = domain.MarginOrderCanceled
	case "rejected":
		st = domain.MarginOrderRejected
	default:
		return nil, fmt.Errorf("%w: status must be open, filled, canceled, rejected, or all", domain.ErrInvalidArgument)
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListMarginOrders(ctx, clientID, st, limit, offset)
}

// CancelMarginOrder cancels an open limit margin order.
func (s *Service) CancelMarginOrder(ctx context.Context, clientID, id string) (*domain.MarginOrder, error) {
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
	return s.store.CancelMarginOrder(ctx, clientID, id, time.Now().UTC(), domain.CancelReasonUser)
}

// ListMarginTrades returns margin trade history.
func (s *Service) ListMarginTrades(ctx context.Context, clientID string, limit, offset int) ([]domain.MarginTrade, error) {
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
	if offset < 0 {
		offset = 0
	}
	return s.store.ListMarginTrades(ctx, clientID, limit, offset)
}

// ProcessMarginMaintenance fills limits, liquidates, and hits SL/TP (worker).
func (s *Service) ProcessMarginMaintenance(ctx context.Context, now time.Time) (filled, liquidated, stopped int, err error) {
	if s.store == nil || s.market == nil {
		return 0, 0, 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	// 1) Limit opens
	orders, err := s.store.ListAllOpenMarginOrders(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	for i := range orders {
		if s.tryFillMarginLimit(ctx, &orders[i], now) {
			filled++
		}
	}
	// 2) Liquidation + SL/TP on open positions
	positions, err := s.store.ListAllOpenMarginPositions(ctx)
	if err != nil {
		return filled, 0, 0, err
	}
	for i := range positions {
		pos := positions[i]
		mark, merr := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
		if merr != nil || mark <= 0 {
			continue
		}
		// Liquidation first
		if domain.ShouldLiquidate(pos.Side, mark, pos.LiquidationPrice) {
			// Close at liquidation price (or mark if worse for realism use mark)
			exit := pos.LiquidationPrice
			if pos.Side == domain.MarginLong && mark < exit {
				exit = mark
			}
			if pos.Side == domain.MarginShort && mark > exit {
				exit = mark
			}
			if _, _, e := s.closeMarginAt(ctx, &pos, pos.Quantity, exit, domain.MarginCloseLiquidation); e == nil {
				liquidated++
			}
			continue
		}
		// Stop loss
		if domain.ShouldTriggerStopLoss(pos.Side, mark, pos.StopLoss) {
			if _, _, e := s.closeMarginAt(ctx, &pos, pos.Quantity, mark, domain.MarginCloseStopLoss); e == nil {
				stopped++
			}
			continue
		}
		// Take profit
		if domain.ShouldTriggerTakeProfit(pos.Side, mark, pos.TakeProfit) {
			if _, _, e := s.closeMarginAt(ctx, &pos, pos.Quantity, mark, domain.MarginCloseTakeProfit); e == nil {
				stopped++
			}
		}
	}
	return filled, liquidated, stopped, nil
}

func (s *Service) tryFillMarginLimit(ctx context.Context, o *domain.MarginOrder, now time.Time) bool {
	if o == nil || o.Status != domain.MarginOrderOpen {
		return false
	}
	last, err := s.lastPrice(ctx, string(o.Exchange), o.Symbol)
	if err != nil || last <= 0 {
		return false
	}
	if !domain.MarginLimitTriggered(o.Side, o.LimitPrice, last) {
		return false
	}
	// Fill at limit price (maker-style)
	price := o.LimitPrice
	p, err := s.store.GetPortfolio(ctx, o.ClientID)
	if err != nil {
		return false
	}
	// Available cash must cover margin; reserved margin already set aside conceptually —
	// reservation is not deducted from cashBalance, only from available — so we debit now
	// and clear reservation in the same tx.
	margin, err := domain.InitialMargin(o.Quantity, price, o.Leverage)
	if err != nil {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "invalid margin", now)
		return false
	}
	// Cash check: full balance minus other spot reserves and OTHER margin reserves (exclude this order)
	reservedSpot, _ := s.store.SumReservedCash(ctx, o.ClientID)
	allMarginRes, _ := s.store.SumReservedMargin(ctx, o.ClientID)
	otherMarginRes := allMarginRes - o.ReservedMargin
	if otherMarginRes < 0 {
		otherMarginRes = 0
	}
	avail := domain.AvailableCash(p.CashBalance, reservedSpot+otherMarginRes)
	if avail+1e-9 < margin {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "insufficient cash at fill", now)
		return false
	}
	nPos, err := s.store.CountOpenMarginPositions(ctx, o.ClientID)
	if err != nil || nPos >= domain.MaxOpenMarginPositions {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "max positions", now)
		return false
	}
	liq, err := domain.LiquidationPriceIsolated(o.Side, price, o.Leverage, domain.DefaultMaintenanceMarginRate)
	if err != nil {
		return false
	}
	pos := domain.MarginPosition{
		ID: uuid.NewString(), ClientID: o.ClientID, Exchange: o.Exchange, Symbol: o.Symbol, Side: o.Side,
		Quantity: o.Quantity, EntryPrice: price, Leverage: o.Leverage, Margin: margin,
		LiquidationPrice: liq, StopLoss: o.StopLoss, TakeProfit: o.TakeProfit,
		Status: domain.MarginPositionOpen, OpenedAt: now, UpdatedAt: now,
	}
	tr := domain.MarginTrade{
		ID: uuid.NewString(), ClientID: o.ClientID, PositionID: pos.ID, Exchange: o.Exchange, Symbol: o.Symbol,
		Side: o.Side, Action: "open", Quantity: o.Quantity, Price: price, Notional: o.Quantity * price,
		MarginDelta: -margin, Leverage: o.Leverage, CreatedAt: now,
	}
	p.CashBalance -= margin
	p.UpdatedAt = now
	if err := s.store.ApplyMarginOpenFromOrder(ctx, p, o.ID, pos, tr, now); err != nil {
		return false
	}
	return true
}

func (s *Service) closeMarginAt(ctx context.Context, pos *domain.MarginPosition, closeQty, price float64, reason string) (*domain.MarginPosition, *domain.MarginTrade, error) {
	p, err := s.store.GetPortfolio(ctx, pos.ClientID)
	if err != nil {
		return nil, nil, err
	}
	// Re-read open state
	cur, err := s.store.GetMarginPosition(ctx, pos.ClientID, pos.ID)
	if err != nil {
		return nil, nil, err
	}
	if cur.Status != domain.MarginPositionOpen {
		return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
	}
	if closeQty > cur.Quantity {
		closeQty = cur.Quantity
	}
	if closeQty <= domain.PositionEpsilon {
		return nil, nil, fmt.Errorf("%w: nothing to close", domain.ErrInvalidArgument)
	}
	full := closeQty+domain.PositionEpsilon >= cur.Quantity
	frac := closeQty / cur.Quantity
	marginRelease := cur.Margin * frac
	realized := domain.MarginRealizedPnL(cur.Side, closeQty, cur.EntryPrice, price)
	now := time.Now().UTC()

	p.CashBalance += marginRelease + realized
	p.RealizedPnLTotal += realized
	p.UpdatedAt = now

	action := "close"
	switch reason {
	case domain.MarginCloseLiquidation:
		action = "liquidation"
	case domain.MarginCloseStopLoss:
		action = "stop_loss"
	case domain.MarginCloseTakeProfit:
		action = "take_profit"
	case domain.MarginClosePartialUser:
		action = "partial_close"
	}

	tr := domain.MarginTrade{
		ID: uuid.NewString(), ClientID: cur.ClientID, PositionID: cur.ID, Exchange: cur.Exchange, Symbol: cur.Symbol,
		Side: cur.Side, Action: action, Quantity: closeQty, Price: price, Notional: closeQty * price,
		RealizedPnL: realized, MarginDelta: marginRelease, Leverage: cur.Leverage, CreatedAt: now,
	}

	updated := *cur
	updated.RealizedPnL += realized
	updated.UpdatedAt = now
	if full {
		updated.Quantity = 0
		updated.Margin = 0
		updated.Status = domain.MarginPositionClosed
		updated.CloseReason = reason
		updated.ClosedAt = &now
		updated.StopLoss = nil
		updated.TakeProfit = nil
	} else {
		updated.Quantity = cur.Quantity - closeQty
		updated.Margin = cur.Margin - marginRelease
		// Recalculate liquidation on remaining (entry/leverage unchanged)
		liq, lerr := domain.LiquidationPriceIsolated(cur.Side, cur.EntryPrice, cur.Leverage, domain.DefaultMaintenanceMarginRate)
		if lerr == nil {
			updated.LiquidationPrice = liq
		}
		if reason == "" {
			reason = domain.MarginClosePartialUser
		}
	}
	if err := s.store.ApplyMarginClose(ctx, p, updated, tr, full); err != nil {
		return nil, nil, err
	}
	out, err := s.store.GetMarginPosition(ctx, cur.ClientID, cur.ID)
	if err != nil {
		return &updated, &tr, nil
	}
	if out.Status == domain.MarginPositionOpen {
		s.markMarginPosition(ctx, out)
	}
	return out, &tr, nil
}
