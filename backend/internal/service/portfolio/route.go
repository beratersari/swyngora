package portfolio

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// previewRouted prices a spot market order split across Binance, Coinbase, and Bybit.
// Nothing is written.
func (s *Service) previewRouted(ctx context.Context, p *domain.Portfolio, in OrderPreviewInput) (*domain.OrderPreview, error) {
	side := domain.TradeSide(strings.ToLower(strings.TrimSpace(in.Side)))
	if !domain.IsValidTradeSide(string(side)) {
		return nil, fmt.Errorf("%w: side must be buy or sell", domain.ErrInvalidArgument)
	}
	lotMethod, err := domain.NormalizeLotMethod(in.LotMethod)
	if err != nil {
		return nil, err
	}
	canon, err := domain.CanonicalRouteSymbol(in.Symbol)
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
	slip, err := domain.ParseMaxSlippagePct(in.MaxSlippagePct)
	if err != nil {
		return nil, err
	}
	books, err := s.loadRouteBooks(ctx, clientID, canon, p.Currency, side)
	if err != nil {
		return nil, err
	}
	route := domain.RoutePaperBooks(side, in.Quantity, p.Currency, slip, books)
	var compare *domain.PaperRouteCompare
	if ex := domain.ParseRouteCompare(in.CompareExchange); ex != "" {
		got := domain.ComparePaperRoute(route, ex, side, in.Quantity, p.Currency, slip, books)
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
	held := routeHeld(books)
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
		Note:              domain.OrderPreviewNote,
	}
	s.stampRoutePreview(out, route, p.Currency, compare, p.CashBalance, avail, snap)
	if side == domain.TradeSideSell && out.FillsNow {
		if pnl, ok := s.routeSellPnL(ctx, clientID, route, lotMethod); ok {
			out.EstimatedRealizedPnL = &pnl
		}
	}
	return s.finishPreview(ctx, clientID, out, in.Reason)
}

// PlaceRoutedOrder fills a spot market order as one trade per venue.
// The cash change is one balance update. A book that cannot hold the size fills nothing.
func (s *Service) PlaceRoutedOrder(ctx context.Context, in OrderInput) ([]domain.Trade, *domain.PortfolioView, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	if in.RecurringPlanID != "" {
		return nil, nil, fmt.Errorf("%w: a recurring buy stays on one venue", domain.ErrInvalidArgument)
	}
	p, _, err := s.requireAccess(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.rejectClosedOwner(ctx, p); err != nil {
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
	lotMethod, err := domain.NormalizeLotMethod(in.LotMethod)
	if err != nil {
		return nil, nil, err
	}
	canon, err := domain.CanonicalRouteSymbol(in.Symbol)
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
	slip, err := domain.ParseMaxSlippagePct(in.MaxSlippagePct)
	if err != nil {
		return nil, nil, err
	}
	idempHash := hashParts("route", canon, string(side), in.Quantity, string(lotMethod), slip, in.Reason.HashPart())
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, err
	} else if rec != nil {
		return s.replayRouted(ctx, rec, in.ClientID, p.ID)
	}
	books, err := s.loadRouteBooks(ctx, clientID, canon, p.Currency, side)
	if err != nil {
		return nil, nil, err
	}
	route := domain.RoutePaperBooks(side, in.Quantity, p.Currency, slip, books)
	if route.InventoryShort {
		return nil, nil, fmt.Errorf("%w: insufficient position quantity", domain.ErrInvalidArgument)
	}
	if route.SlippageLimited {
		return nil, nil, fmt.Errorf("%w: max slippage cannot fill this size", domain.ErrInvalidArgument)
	}
	if route.Exhausted || route.FilledQuantity+1e-12 < in.Quantity || len(route.Legs) == 0 {
		return nil, nil, domain.ErrBookExhausted
	}
	if side == domain.TradeSideBuy {
		base, _ := domain.SplitBaseQuote(domain.ExchangeBinance, canon)
		if err := s.guardNewRisk(ctx, clientID, base, route.Notional); err != nil {
			return nil, nil, err
		}
	}
	availCash, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
	if err != nil {
		return nil, nil, err
	}
	if side == domain.TradeSideBuy && (availCash+1e-9 < route.CashDebit || p.CashBalance+1e-9 < route.CashDebit) {
		return nil, nil, fmt.Errorf("%w: insufficient cash balance", domain.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	sit := s.captureEntryRead(ctx, domain.ExchangeBinance, canon, side, in.Reason)
	fills, cash, realized, err := s.buildRoutedFills(ctx, clientID, p, side, lotMethod, route, sit, availCash, now)
	if err != nil {
		return nil, nil, err
	}
	p.CashBalance = cash
	p.RealizedPnLTotal += realized
	p.UpdatedAt = now
	ids := make([]string, len(fills))
	for i := range fills {
		ids[i] = fills[i].Trade.ID
	}
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindTrade, idempIDs{TradeID: ids[0], TradeIDs: ids})
	if err := s.store.ExecuteRoutedTrades(ctx, p, fills); err != nil {
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayRouted(ctx, rec, in.ClientID, p.ID)
			}
		}
		return nil, nil, err
	}
	view, verr := s.View(ctx, in.ClientID, p.ID)
	out := make([]domain.Trade, len(fills))
	for i := range fills {
		out[i] = *s.stampTrade(ctx, &fills[i].Trade)
		if view != nil {
			s.notifyChange(ctx, p.ID, domain.PortfolioChangeOrderFilled, nil, &out[i], view)
		}
	}
	if verr != nil {
		return out, nil, verr
	}
	return out, view, nil
}

func (s *Service) loadRouteBooks(ctx context.Context, bookID, canon, currency string, side domain.TradeSide) ([]domain.PaperRouteBook, error) {
	if s == nil || s.books == nil {
		return nil, fmt.Errorf("%w: order book is not configured", domain.ErrUpstream)
	}
	out := make([]domain.PaperRouteBook, 0, len(domain.PaperRouteVenues()))
	eligible := 0
	live := 0
	for _, ex := range domain.PaperRouteVenues() {
		pair := domain.CrossVenueSymbol(ex, canon)
		_, quote := domain.SplitBaseQuote(ex, pair)
		row := domain.PaperRouteBook{
			Exchange: ex, Symbol: pair, Quote: quote, FeeRate: s.paperCost(ex).FeeRate,
		}
		if pair == "" || !domain.QuotesAlias(currency, quote) {
			row.Skipped = domain.PaperRouteSkipQuote
			row.Detail = quote
			out = append(out, row)
			continue
		}
		eligible++
		if side == domain.TradeSideSell {
			cap, err := s.freeQuantity(ctx, bookID, ex, pair)
			if err != nil {
				return nil, err
			}
			row.CapQty = &cap
		}
		raw, err := s.books.GetRawOrderBook(ctx, string(ex), pair)
		if err != nil || raw == nil || (len(raw.Bids) == 0 && len(raw.Asks) == 0) {
			if err != nil {
				row.Missing = err.Error()
			} else {
				row.Missing = "empty order book"
			}
			out = append(out, row)
			continue
		}
		row.Bids = raw.Bids
		row.Asks = raw.Asks
		live++
		out = append(out, row)
	}
	if eligible == 0 {
		return nil, fmt.Errorf("%w: no spot venue quotes this book's currency", domain.ErrInvalidArgument)
	}
	if live == 0 {
		return nil, fmt.Errorf("%w: no live order books for %s", domain.ErrUpstream, canon)
	}
	return out, nil
}

func (s *Service) freeQuantity(ctx context.Context, bookID string, ex domain.Exchange, symbol string) (float64, error) {
	var posQty float64
	pos, err := s.store.GetPosition(ctx, bookID, ex, symbol)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return 0, err
	}
	if pos != nil {
		posQty = pos.Quantity
	}
	reserved, err := s.store.SumReservedQuantity(ctx, bookID, ex, symbol)
	if err != nil {
		return 0, err
	}
	return domain.AvailablePosition(posQty, reserved), nil
}

func (s *Service) routeLastPrice(ctx context.Context, compare, canon string, books []domain.PaperRouteBook) float64 {
	try := make([]domain.Exchange, 0, len(books)+1)
	if ex := domain.ParseRouteCompare(compare); ex != "" {
		try = append(try, ex)
	}
	for _, b := range books {
		if b.Missing == "" && b.Skipped == "" {
			try = append(try, b.Exchange)
		}
	}
	seen := map[domain.Exchange]bool{}
	for _, ex := range try {
		if seen[ex] {
			continue
		}
		seen[ex] = true
		pair := domain.CrossVenueSymbol(ex, canon)
		px, err := s.lastPrice(ctx, string(ex), pair)
		if err == nil && px > 0 {
			return px
		}
	}
	return 0
}

func (s *Service) stampRoutePreview(out *domain.OrderPreview, route domain.PaperRoute, currency string, compare *domain.PaperRouteCompare, cash, avail float64, snap riskSnap) {
	if out == nil {
		return
	}
	out.Routed = true
	out.BookSource = domain.PaperFillBook
	out.RouteLegs = route.Legs
	out.RouteMissing = route.Missing
	out.RouteSkipped = route.Skipped
	out.RouteCompare = compare
	out.RouteNote = domain.ExplainPaperRoute(currency, route)
	out.BookExhausted = route.Exhausted
	out.BookVisibleQuantity = route.VisibleQuantity
	out.BookFilledQuantity = route.FilledQuantity
	out.BookLevels = route.LevelsUsed
	out.FillPrice = route.AveragePrice
	out.ReferencePrice = out.LastPrice
	out.MaxSlippagePct = route.MaxSlippage * 100
	out.SlippageTouch = route.TouchPrice
	out.SlippageLimitPrice = route.LimitPrice
	out.SlippageInsideQuantity = route.InsideQuantity
	out.SlippageLimited = route.SlippageLimited
	full := !route.Exhausted && !route.InventoryShort && !route.SlippageLimited && route.FilledQuantity+1e-12 >= out.Quantity && len(route.Legs) > 0
	out.FillsNow = full
	out.InsufficientQuantity = route.InventoryShort
	if !full {
		return
	}
	out.Notional = route.Notional
	out.Fee = route.Fee
	out.CashDebit = route.CashDebit
	out.CashCredit = route.CashCredit
	if out.Side == string(domain.TradeSideBuy) {
		out.CashAfter = cash - route.CashDebit
		out.AvailableAfter = avail - route.CashDebit
		out.InsufficientCash = avail+1e-9 < route.CashDebit
		out.Risk = snap.build(route.Notional, true)
		return
	}
	out.CashAfter = cash + route.CashCredit
	out.AvailableAfter = avail + route.CashCredit
	out.Risk = snap.sellView(route.Notional)
}

func (s *Service) routeSellPnL(ctx context.Context, bookID string, route domain.PaperRoute, method domain.LotMethod) (float64, bool) {
	var sum float64
	for i := range route.Legs {
		leg := route.Legs[i]
		posQty, avg := 0.0, 0.0
		pos, err := s.store.GetPosition(ctx, bookID, leg.Exchange, leg.Symbol)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return 0, false
		}
		if pos != nil {
			posQty, avg = pos.Quantity, pos.AvgCost
		}
		lots, err := s.loadOpenLots(ctx, bookID, leg.Exchange, leg.Symbol)
		if err != nil {
			return 0, false
		}
		pnl := sellPnL(lots, bookID, leg.Exchange, leg.Symbol, posQty, avg, leg.Quantity, leg.FillPrice, method, leg.FeeRate)
		if pnl == nil {
			return 0, false
		}
		sum += *pnl
	}
	return sum, true
}

func (s *Service) buildRoutedFills(ctx context.Context, bookID string, p *domain.Portfolio, side domain.TradeSide, method domain.LotMethod, route domain.PaperRoute, sit *domain.CoinSituation, avail float64, now time.Time) ([]domain.RoutedFill, float64, float64, error) {
	cash := p.CashBalance
	running := avail
	var realized float64
	fills := make([]domain.RoutedFill, 0, len(route.Legs))
	for _, leg := range route.Legs {
		posQty, avg := 0.0, 0.0
		pos, err := s.store.GetPosition(ctx, bookID, leg.Exchange, leg.Symbol)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, 0, 0, err
		}
		if pos != nil {
			posQty, avg = pos.Quantity, pos.AvgCost
		}
		lots, err := s.loadOpenLots(ctx, bookID, leg.Exchange, leg.Symbol)
		if err != nil {
			return nil, 0, 0, err
		}
		last := leg.BestPrice
		if px, lerr := s.lastPrice(ctx, string(leg.Exchange), leg.Symbol); lerr == nil && px > 0 {
			last = px
		}
		tradeID := uuid.NewString()
		var (
			newQty, newAvg, legRealized float64
			lotOps                      *domain.LotOps
		)
		switch side {
		case domain.TradeSideBuy:
			unit := domain.BuyUnitCost(leg.FillPrice, leg.FeeRate)
			if _, _, _, err = domain.ApplyBuy(running, leg.Quantity, unit, posQty, avg); err != nil {
				return nil, 0, 0, err
			}
			running -= leg.CashDebit
			cash -= leg.CashDebit
			if cash < -1e-9 {
				return nil, 0, 0, fmt.Errorf("%w: insufficient cash balance", domain.ErrInvalidArgument)
			}
			lotOps = prepareBuyLots(bookID, leg.Exchange, leg.Symbol, lots, posQty, avg, leg.Quantity, unit, tradeID, now)
			merged := append(append([]domain.TaxLot(nil), lots...), lotOps.Created...)
			newQty = posQty + leg.Quantity
			newAvg = domain.AvgCostFromLots(merged)
			if newAvg <= 0 {
				newAvg = unit
			}
		case domain.TradeSideSell:
			lotOps, legRealized, newAvg, err = prepareSellLots(lots, bookID, leg.Exchange, leg.Symbol, posQty, avg, leg.Quantity, leg.FillPrice, method, tradeID, now, leg.FeeRate)
			if err != nil {
				return nil, 0, 0, err
			}
			cash += leg.CashCredit
			realized += legRealized
			newQty = posQty - leg.Quantity
			if newQty < domain.PositionEpsilon {
				newQty = 0
				newAvg = 0
			}
		}
		posOut := &domain.Position{
			ClientID: bookID, Exchange: leg.Exchange, Symbol: leg.Symbol,
			Quantity: newQty, AvgCost: newAvg, UpdatedAt: now,
		}
		if newQty <= domain.PositionEpsilon {
			posOut.Quantity = 0
			posOut.AvgCost = 0
		}
		tr := domain.Trade{
			ID: tradeID, ClientID: bookID, Exchange: leg.Exchange, Symbol: leg.Symbol,
			Side: side, Quantity: leg.Quantity, Price: leg.FillPrice, Notional: leg.Notional,
			RealizedPnL: legRealized, LotMethod: method, Fee: leg.Fee, LastPrice: last,
			Situation: sit, CreatedAt: now,
		}
		if lotOps != nil {
			tr.LotFills = lotOps.Fills
		}
		fills = append(fills, domain.RoutedFill{Position: posOut, Trade: tr, Lots: lotOps})
	}
	return fills, cash, realized, nil
}

func (s *Service) replayRouted(ctx context.Context, rec *domain.IdempotencyRecord, actor, portfolioID string) ([]domain.Trade, *domain.PortfolioView, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	keys := ids.TradeIDs
	if len(keys) == 0 && ids.TradeID != "" {
		keys = []string{ids.TradeID}
	}
	out := make([]domain.Trade, 0, len(keys))
	for _, id := range keys {
		tr, err := s.store.GetTrade(ctx, rec.ClientID, id)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, *s.stampTrade(ctx, tr))
	}
	view, err := s.View(ctx, actor, portfolioID)
	if err != nil {
		return out, nil, err
	}
	return out, view, nil
}

func routeHeld(books []domain.PaperRouteBook) float64 {
	var held float64
	for _, b := range books {
		if b.Missing != "" || b.Skipped != "" || b.CapQty == nil {
			continue
		}
		if *b.CapQty > 0 {
			held += *b.CapQty
		}
	}
	return held
}
