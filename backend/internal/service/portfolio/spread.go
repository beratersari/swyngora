package portfolio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// previewSpread prices the first slice of a timed best-cost order.
// Nothing is written. Later slices are not priced, because each one reads the books again.
func (s *Service) previewSpread(ctx context.Context, p *domain.Portfolio, in OrderPreviewInput) (*domain.OrderPreview, error) {
	plan, err := domain.NormalizeSpreadSchedule(in.SpreadMinutes, in.SliceCount)
	if err != nil {
		return nil, err
	}
	side, canon, lotMethod, slip, err := spreadInputs(in.Side, in.Symbol, in.LotMethod, in.MaxSlippagePct, in.Quantity, plan.SliceCount)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
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
	target := domain.SpreadTarget(in.Quantity, 0, plan.SliceCount)
	books, err := s.loadRouteBooks(ctx, clientID, canon, p.Currency, side)
	if err != nil {
		return nil, err
	}
	held := routeHeld(books)
	var holds []domain.SpreadHold
	inventoryShort := false
	if side == domain.TradeSideSell {
		holds, inventoryShort = spreadHoldsFromBooks(in.Quantity, books)
		if inventoryShort {
			holds = nil
		} else {
			books = booksWithHolds(books, holds)
		}
	}
	route := domain.RoutePaperBooks(side, target, p.Currency, slip, books)
	if inventoryShort {
		route.InventoryShort = true
		route.Exhausted = true
		route.Legs = nil
		route.FilledQuantity = 0
	}
	var compare *domain.PaperRouteCompare
	if ex := domain.ParseRouteCompare(in.CompareExchange); ex != "" {
		got := domain.ComparePaperRoute(route, ex, side, target, p.Currency, slip, books)
		compare = &got
	}
	avail, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
	if err != nil {
		return nil, err
	}
	base, _ := domain.SplitBaseQuote(domain.ExchangeBinance, canon)
	snap, err := s.previewSnap(ctx, p, base)
	if err != nil {
		return nil, err
	}
	last := s.routeLastPrice(ctx, in.CompareExchange, canon, books)
	out := &domain.OrderPreview{
		Product:           "spot",
		Type:              "market",
		Side:              string(side),
		Exchange:          domain.Exchange(domain.PaperRouteAll),
		Symbol:            canon,
		Quantity:          in.Quantity,
		LastPrice:         last,
		AvailableCash:     avail,
		AvailableQuantity: held,
		CashAfter:         p.CashBalance,
		AvailableAfter:    avail,
		Note:              domain.ExplainSpread(plan.SliceCount, plan.Minutes),
		Spread:            true,
		SpreadMinutes:     plan.Minutes,
		SliceCount:        plan.SliceCount,
		SliceQuantity:     target,
		SliceIntervalSec:  plan.IntervalSec(),
	}
	s.stampSpreadPreview(out, route, target, inventoryShort, compare, p.CashBalance, avail, snap, slip)
	if side == domain.TradeSideSell && spreadSliceComplete(route, target) {
		if pnl, ok := s.routeSellPnL(ctx, clientID, route, lotMethod); ok {
			out.EstimatedRealizedPnL = &pnl
		}
	}
	return s.finishPreview(ctx, clientID, out, in.Reason)
}

// PlaceSpreadOrder starts a best-cost order that fills in the requested slices.
// The first slice is due immediately and is taken by the background worker.
// The last slice is due when the duration ends.
func (s *Service) PlaceSpreadOrder(ctx context.Context, in OrderInput) (*domain.SpreadOrder, *domain.PortfolioView, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	if in.RecurringPlanID != "" {
		return nil, nil, fmt.Errorf("%w: a recurring buy stays on one venue", domain.ErrInvalidArgument)
	}
	if !domain.IsPaperRouteExchange(in.Exchange) {
		return nil, nil, fmt.Errorf("%w: a spread order uses best cost", domain.ErrInvalidArgument)
	}
	p, _, err := s.requireAccess(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.rejectClosedOwner(ctx, p); err != nil {
		return nil, nil, err
	}
	plan, err := domain.NormalizeSpreadSchedule(in.SpreadMinutes, in.SliceCount)
	if err != nil {
		return nil, nil, err
	}
	side, canon, lotMethod, slip, err := spreadInputs(in.Side, in.Symbol, in.LotMethod, in.MaxSlippagePct, in.Quantity, plan.SliceCount)
	if err != nil {
		return nil, nil, err
	}
	clientID := p.BookID()
	idempKey, err := domain.NormalizeIdempotencyKey(in.IdempotencyKey)
	if err != nil {
		return nil, nil, err
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	fresh, ferr := s.store.GetPortfolio(ctx, clientID)
	if ferr != nil {
		return nil, nil, ferr
	}
	p = fresh
	if err := domain.RouteSymbolMatchesBook(canon, p.Currency); err != nil {
		return nil, nil, err
	}
	idempHash := hashParts("spread", canon, string(side), in.Quantity, string(lotMethod), slip, in.Reason.HashPart(), plan.Minutes, plan.SliceCount)
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, err
	} else if rec != nil {
		return s.replaySpread(ctx, rec, in.ClientID, p.ID)
	}
	target := domain.SpreadTarget(in.Quantity, 0, plan.SliceCount)
	books, err := s.loadRouteBooks(ctx, clientID, canon, p.Currency, side)
	if err != nil {
		return nil, nil, err
	}
	var holds []domain.SpreadHold
	if side == domain.TradeSideSell {
		var short bool
		holds, short = spreadHoldsFromBooks(in.Quantity, books)
		if short {
			return nil, nil, fmt.Errorf("%w: insufficient position quantity", domain.ErrInvalidArgument)
		}
		books = booksWithHolds(books, holds)
	}
	route := domain.RoutePaperBooks(side, target, p.Currency, slip, books)
	if err := rejectIncompleteSlice(route, target); err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	order := domain.SpreadOrder{
		ID: uuid.NewString(), ClientID: clientID, Symbol: canon, Side: side,
		Quantity: in.Quantity, RemainingQuantity: in.Quantity,
		SliceCount: plan.SliceCount, Duration: plan.Duration, NextSliceAt: now, EndsAt: now.Add(plan.Duration),
		Status: domain.SpreadStatusOpen, MaxSlippage: slip, LotMethod: lotMethod,
		CreatedAt: now, UpdatedAt: now,
	}
	if side == domain.TradeSideBuy {
		order.ReservedCash = spreadBuyLock(in.Quantity, route, slip)
		base, _ := domain.SplitBaseQuote(domain.ExchangeBinance, canon)
		if err := s.guardNewRisk(ctx, clientID, base, in.Quantity*domain.SpreadReservePrice(route.TouchPrice, slip)); err != nil {
			return nil, nil, err
		}
		avail, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
		if err != nil {
			return nil, nil, err
		}
		if avail+1e-9 < order.ReservedCash || p.CashBalance+1e-9 < order.ReservedCash {
			return nil, nil, fmt.Errorf("%w: insufficient cash balance", domain.ErrInvalidArgument)
		}
		if in.Reason.Set {
			order.ReasonSet = true
			order.Reason = append([]string(nil), in.Reason.Clauses...)
		}
	} else {
		order.Holds = holds
	}
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindSpread, idempIDs{OrderID: order.ID})
	created, err := s.store.CreateSpreadOrder(ctx, order)
	if err != nil {
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replaySpread(ctx, rec, in.ClientID, p.ID)
			}
		}
		return nil, nil, err
	}
	s.keepFillCost(ctx, s.takeFillQuote(ctx, clientID, in.ExecutionQuoteID, domain.ExecutionSpread, canon, side, in.Quantity, "", in.MaxSlippagePct, in.SpreadMinutes, in.SliceCount), nil, created.ID, false)
	view, verr := s.View(ctx, in.ClientID, p.ID)
	if verr == nil && view != nil {
		s.notifyChange(ctx, p.ID, domain.PortfolioChangeOrderPlaced, nil, nil, view)
	}
	return created, view, verr
}

// CancelSpreadOrder stops slices that have not run and releases the unused lock.
// Fills already taken stay on the book.
func (s *Service) CancelSpreadOrder(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.SpreadOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleTrader, portfolioID...)
	if err != nil {
		return nil, err
	}
	bookID := p.BookID()
	unlock := s.lockClient(bookID)
	defer unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	o, err := s.store.GetSpreadOrder(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.SpreadStatusOpen {
		return nil, fmt.Errorf("%w: order is not open", domain.ErrConflict)
	}
	now := time.Now().UTC()
	o.Status = domain.SpreadStatusCanceled
	o.CancelReason = domain.CancelReasonUser
	o.ReservedCash = 0
	o.Holds = nil
	o.UpdatedAt = now
	if err := s.store.SaveSpreadWorking(ctx, nil, *o, nil); err != nil {
		return nil, err
	}
	s.applySpreadFillCost(ctx, o, nil)
	s.notifyChange(ctx, bookID, domain.PortfolioChangeOrderCancelled, nil, nil, nil)
	return o, nil
}

// ListSpreadOrders returns spread orders for one book. An empty status means open.
func (s *Service) ListSpreadOrders(ctx context.Context, clientID, status string, limit, offset int, portfolioID ...string) ([]domain.SpreadOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
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
	st := domain.SpreadStatus(strings.ToLower(strings.TrimSpace(status)))
	if st == "" {
		st = domain.SpreadStatusOpen
	}
	if st == "all" {
		st = ""
	} else if st != domain.SpreadStatusOpen && st != domain.SpreadStatusFilled && st != domain.SpreadStatusCanceled {
		return nil, fmt.Errorf("%w: status must be open, filled, canceled, or all", domain.ErrInvalidArgument)
	}
	return s.store.ListSpreadOrders(ctx, p.BookID(), st, limit, offset)
}

// ProcessDueSpreadOrders fills at most one due slice per open order.
// A restart does not dump the remaining size in one fill.
func (s *Service) ProcessDueSpreadOrders(ctx context.Context, now time.Time) (int, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	open, err := s.store.ListOpenSpreadOrders(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range open {
		if open[i].NextSliceAt.After(now) {
			continue
		}
		ok, err := s.fillSpreadSlice(ctx, open[i].ClientID, open[i].ID, now)
		if err != nil {
			slog.Default().Debug("spread slice", "id", open[i].ID, "err", err)
			continue
		}
		if ok {
			n++
		}
	}
	return n, nil
}

func (s *Service) fillSpreadSlice(ctx context.Context, bookID, orderID string, now time.Time) (bool, error) {
	unlock := s.lockClient(bookID)
	defer unlock()
	o, err := s.store.GetSpreadOrder(ctx, bookID, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if o.Status != domain.SpreadStatusOpen || o.NextSliceAt.After(now) {
		return false, nil
	}
	p, err := s.store.GetPortfolio(ctx, bookID)
	if err != nil {
		return false, err
	}
	if o.RemainingQuantity < domain.MinTradeQuantity {
		o.RemainingQuantity = 0
		return true, s.finishSpread(ctx, p, o, nil, now)
	}
	target := domain.SpreadTarget(o.RemainingQuantity, o.SlicesDone, o.SliceCount)
	books, err := s.loadRouteBooks(ctx, bookID, o.Symbol, p.Currency, o.Side)
	if err != nil {
		return false, err
	}
	if o.Side == domain.TradeSideSell {
		books = booksWithHolds(books, o.Holds)
	}
	route := domain.RoutePaperBooks(o.Side, target, p.Currency, o.MaxSlippage, books)
	if o.Side == domain.TradeSideBuy {
		reserved := o.ReservedCash
		route = fitBuyRoute(route, reserved, func(qty float64) domain.PaperRoute {
			return domain.RoutePaperBooks(o.Side, qty, p.Currency, o.MaxSlippage, books)
		})
	}
	return true, s.finishSpread(ctx, p, o, &route, now)
}

func (s *Service) finishSpread(ctx context.Context, p *domain.Portfolio, o *domain.SpreadOrder, route *domain.PaperRoute, now time.Time) error {
	var fills []domain.RoutedFill
	filledQty := 0.0
	if route != nil && route.FilledQuantity > 0 && len(route.Legs) > 0 {
		filledQty = route.FilledQuantity
		reason := domain.SituationReason{}
		if o.ReasonSet {
			reason.Set = true
			reason.Clauses = o.Reason
		}
		sit := s.captureEntryRead(ctx, domain.ExchangeBinance, o.Symbol, o.Side, reason)
		avail, err := s.availableCashForTrading(ctx, o.ClientID, p.CashBalance)
		if err != nil {
			return err
		}
		if o.Side == domain.TradeSideBuy {
			avail += o.ReservedCash
		}
		built, cash, realized, err := s.buildRoutedFills(ctx, o.ClientID, p, o.Side, o.LotMethod, *route, sit, avail, now)
		if err != nil {
			return err
		}
		fills = built
		p.CashBalance = cash
		p.RealizedPnLTotal += realized
		p.UpdatedAt = now
		if o.Side == domain.TradeSideBuy {
			o.ReservedCash -= domain.SpreadBuyRelease(o.ReservedCash, o.RemainingQuantity, filledQty, route.CashDebit)
			if o.ReservedCash < 1e-8 {
				o.ReservedCash = 0
			}
		} else {
			used := make([]domain.SpreadHold, 0, len(route.Legs))
			for _, leg := range route.Legs {
				used = append(used, domain.SpreadHold{Exchange: leg.Exchange, Symbol: leg.Symbol, Quantity: leg.Quantity})
			}
			o.Holds = domain.ReduceSpreadHolds(o.Holds, used)
		}
		o.FilledNotional += route.Notional
		o.FilledFee += route.Fee
	}
	o.FilledQuantity += filledQty
	o.RemainingQuantity -= filledQty
	if o.RemainingQuantity < domain.MinTradeQuantity {
		o.RemainingQuantity = 0
	}
	if route != nil {
		o.SlicesDone++
	}
	o.UpdatedAt = now
	if o.RemainingQuantity == 0 {
		o.Status = domain.SpreadStatusFilled
		o.ReservedCash = 0
		o.Holds = nil
	} else if o.SlicesDone >= o.SliceCount {
		o.Status = domain.SpreadStatusCanceled
		o.CancelReason = domain.CancelReasonExpired
		o.ReservedCash = 0
		o.Holds = nil
	} else if route != nil {
		next := o.NextSliceAt.Add(o.SliceInterval())
		if !next.After(now) {
			next = now.Add(o.SliceInterval())
		}
		o.NextSliceAt = next
	}
	if err := s.store.SaveSpreadWorking(ctx, p, *o, fills); err != nil {
		return err
	}
	s.applySpreadFillCost(ctx, o, fills)
	fresh, err := s.store.GetPortfolio(ctx, o.ClientID)
	if err != nil || fresh == nil {
		return nil
	}
	view, err := s.buildView(ctx, fresh, domain.PortfolioRoleOwner)
	if err != nil {
		return nil
	}
	if len(fills) == 0 {
		s.notifyChange(ctx, o.ClientID, domain.PortfolioChangeOrderUpdated, nil, nil, view)
		return nil
	}
	for i := range fills {
		tr := fills[i].Trade
		s.notifyChange(ctx, o.ClientID, domain.PortfolioChangeOrderFilled, nil, &tr, view)
	}
	return nil
}

func (s *Service) replaySpread(ctx context.Context, rec *domain.IdempotencyRecord, actor, portfolioID string) (*domain.SpreadOrder, *domain.PortfolioView, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	o, err := s.store.GetSpreadOrder(ctx, rec.ClientID, ids.OrderID)
	if err != nil {
		return nil, nil, err
	}
	view, err := s.View(ctx, actor, portfolioID)
	if err != nil {
		return o, nil, err
	}
	return o, view, nil
}

func (s *Service) stampSpreadPreview(out *domain.OrderPreview, route domain.PaperRoute, target float64, inventoryShort bool, compare *domain.PaperRouteCompare, cash, avail float64, snap riskSnap, slip float64) {
	if out == nil {
		return
	}
	out.Routed = true
	out.BookSource = domain.PaperFillBook
	out.RouteLegs = route.Legs
	out.RouteMissing = route.Missing
	out.RouteSkipped = route.Skipped
	out.RouteCompare = compare
	if out.Note != "" {
		out.RouteNote = out.Note
	}
	complete := spreadSliceComplete(route, target)
	out.BookExhausted = !complete && !route.SlippageLimited && !route.InventoryShort && !inventoryShort
	out.BookVisibleQuantity = route.VisibleQuantity
	out.BookFilledQuantity = route.FilledQuantity
	out.BookLevels = route.LevelsUsed
	out.FillPrice = route.AveragePrice
	out.ReferencePrice = out.LastPrice
	out.MaxSlippagePct = slip * 100
	out.SlippageTouch = route.TouchPrice
	out.SlippageLimitPrice = route.LimitPrice
	out.SlippageInsideQuantity = route.InsideQuantity
	out.SlippageLimited = route.SlippageLimited
	out.FillsNow = false
	out.InsufficientQuantity = inventoryShort || route.InventoryShort
	price := domain.SpreadReservePrice(route.TouchPrice, slip)
	if price <= 0 && out.LastPrice > 0 {
		price = domain.SpreadReservePrice(out.LastPrice, slip)
	}
	if out.Side == string(domain.TradeSideBuy) {
		lock := domain.SpreadBuyReserve(out.Quantity, route.TouchPrice, slip, domain.CoinbaseFeeRate)
		if lock <= 0 && out.LastPrice > 0 {
			lock = domain.SpreadBuyReserve(out.Quantity, out.LastPrice, slip, domain.CoinbaseFeeRate)
		}
		if complete && route.FilledQuantity > 0 {
			projected := route.CashDebit * (out.Quantity / route.FilledQuantity)
			if projected > lock {
				lock = projected
			}
			out.Notional = route.Notional
			out.Fee = route.Fee
			out.CashDebit = route.CashDebit
		}
		out.ReservedCash = lock
		out.CashAfter = cash
		out.AvailableAfter = avail - lock
		out.InsufficientCash = avail+1e-9 < lock
		out.Risk = snap.build(out.Quantity*price, true)
		return
	}
	if !inventoryShort {
		out.ReservedQuantity = out.Quantity
	}
	if complete {
		out.Notional = route.Notional
		out.Fee = route.Fee
		out.CashCredit = route.CashCredit
	}
	out.CashAfter = cash
	out.AvailableAfter = avail
	out.Risk = snap.sellView(out.Quantity * price)
}

func spreadInputs(sideRaw, symbol, lotRaw string, maxSlippagePct, qty float64, slices int) (domain.TradeSide, string, domain.LotMethod, float64, error) {
	side := domain.TradeSide(strings.ToLower(strings.TrimSpace(sideRaw)))
	if !domain.IsValidTradeSide(string(side)) {
		return "", "", "", 0, fmt.Errorf("%w: side must be buy or sell", domain.ErrInvalidArgument)
	}
	if qty < domain.MinTradeQuantity || qty > domain.MaxTradeQuantity || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return "", "", "", 0, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	target := domain.SpreadTarget(qty, 0, slices)
	if target < domain.MinTradeQuantity {
		return "", "", "", 0, fmt.Errorf("%w: quantity is too small to spread", domain.ErrInvalidArgument)
	}
	lotMethod, err := domain.NormalizeLotMethod(lotRaw)
	if err != nil {
		return "", "", "", 0, err
	}
	canon, err := domain.CanonicalRouteSymbol(symbol)
	if err != nil {
		return "", "", "", 0, err
	}
	slip, err := domain.ParseMaxSlippagePct(maxSlippagePct)
	if err != nil {
		return "", "", "", 0, err
	}
	return side, canon, lotMethod, slip, nil
}

func spreadHoldsFromBooks(qty float64, books []domain.PaperRouteBook) ([]domain.SpreadHold, bool) {
	free := make([]domain.SpreadFree, 0, len(books))
	for _, b := range books {
		if b.Missing != "" || b.Skipped != "" || b.CapQty == nil {
			continue
		}
		free = append(free, domain.SpreadFree{Exchange: b.Exchange, Symbol: b.Symbol, Free: *b.CapQty})
	}
	holds, ok := domain.AllocateSpreadHolds(qty, free)
	return holds, !ok
}

func booksWithHolds(books []domain.PaperRouteBook, holds []domain.SpreadHold) []domain.PaperRouteBook {
	qty := map[string]float64{}
	for _, h := range holds {
		qty[string(h.Exchange)+"|"+h.Symbol] = h.Quantity
	}
	out := append([]domain.PaperRouteBook(nil), books...)
	caps := make([]float64, len(out))
	for i := range out {
		if out[i].Skipped != "" || out[i].Missing != "" {
			continue
		}
		caps[i] = qty[string(out[i].Exchange)+"|"+out[i].Symbol]
		out[i].CapQty = &caps[i]
	}
	return out
}

func spreadSliceComplete(route domain.PaperRoute, qty float64) bool {
	return qty > 0 && !route.Exhausted && !route.InventoryShort && !route.SlippageLimited &&
		route.FilledQuantity+1e-12 >= qty && len(route.Legs) > 0
}

func rejectIncompleteSlice(route domain.PaperRoute, qty float64) error {
	if route.InventoryShort {
		return fmt.Errorf("%w: insufficient position quantity", domain.ErrInvalidArgument)
	}
	if route.SlippageLimited {
		return fmt.Errorf("%w: max slippage cannot fill this size", domain.ErrInvalidArgument)
	}
	if !spreadSliceComplete(route, qty) {
		return domain.ErrBookExhausted
	}
	return nil
}

func spreadBuyLock(qty float64, route domain.PaperRoute, slip float64) float64 {
	lock := domain.SpreadBuyReserve(qty, route.TouchPrice, slip, domain.CoinbaseFeeRate)
	if route.FilledQuantity > 0 {
		projected := route.CashDebit * (qty / route.FilledQuantity)
		if projected > lock {
			lock = projected
		}
	}
	return lock
}

func fitBuyRoute(route domain.PaperRoute, cash float64, reroute func(float64) domain.PaperRoute) domain.PaperRoute {
	if route.CashDebit <= cash+1e-9 || route.FilledQuantity <= 0 {
		return route
	}
	if cash <= 1e-9 || route.CashDebit <= 0 {
		return domain.PaperRoute{Side: route.Side, RequestedQuantity: route.RequestedQuantity, Exhausted: true, TouchPrice: route.TouchPrice}
	}
	next := reroute(route.RequestedQuantity * (cash / route.CashDebit) * 0.999)
	if next.CashDebit > cash+1e-6 && next.CashDebit > 0 && next.RequestedQuantity > 0 {
		next = reroute(next.RequestedQuantity * (cash / next.CashDebit) * 0.999)
	}
	if next.CashDebit > cash+1e-6 {
		return domain.PaperRoute{Side: route.Side, RequestedQuantity: route.RequestedQuantity, Exhausted: true, TouchPrice: route.TouchPrice}
	}
	return next
}
