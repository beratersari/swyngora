package portfolio

import (
	"context"
	"errors"
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
	paperNote = "Paper trading only — simulated fills at last price plus venue slippage and taker fee. Not financial advice. No real money."
)

// PriceFetcher loads last prices for paper fills and marks.
type PriceFetcher interface {
	GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error)
}

// AccountChecker reports whether a tenant account is closed (optional).
type AccountChecker interface {
	IsClosed(ctx context.Context, clientID string) (bool, *domain.Account, error)
}

// Service orchestrates paper-trading portfolios.
type Service struct {
	store    domain.PortfolioPort
	market   PriceFetcher
	sink     domain.PortfolioChangeSink
	account  AccountChecker
	costFor  func(domain.Exchange) domain.TradingCost
	clientMu sync.Map // clientID → *sync.Mutex; serializes RMW per client
}

// New constructs a portfolio service.
func New(store domain.PortfolioPort, market PriceFetcher) *Service {
	return &Service{store: store, market: market, costFor: domain.TradingCostFor}
}

// WithPaperCosts overrides per-exchange fee/slippage (tests use domain.ZeroTradingCosts).
func (s *Service) WithPaperCosts(fn func(domain.Exchange) domain.TradingCost) *Service {
	if s != nil && fn != nil {
		s.costFor = fn
	}
	return s
}

// SetAccountChecker skips worker ticks for closed tenants (optional).
func (s *Service) SetAccountChecker(a AccountChecker) {
	if s != nil {
		s.account = a
	}
}

func (s *Service) paperCost(ex domain.Exchange) domain.TradingCost {
	if s != nil && s.costFor != nil {
		return s.costFor(ex)
	}
	return domain.TradingCostFor(ex)
}

// SetChangeSink receives order/position/cash mutations for realtime subscribers.
func (s *Service) SetChangeSink(sink domain.PortfolioChangeSink) {
	if s != nil {
		s.sink = sink
	}
}

func (s *Service) notifyChange(ctx context.Context, bookID, reason string, order *domain.PendingOrder, trade *domain.Trade, view *domain.PortfolioView) {
	if s == nil || s.sink == nil {
		return
	}
	if view == nil && bookID != "" && s.store != nil {
		if p, err := s.store.GetPortfolio(ctx, bookID); err == nil && p != nil {
			view, _ = s.buildView(ctx, p, domain.PortfolioRoleOwner)
			if view != nil {
				bookID = view.ID
			}
		}
	}
	if bookID == "" && view != nil {
		bookID = view.ID
	}
	if bookID == "" && order != nil {
		bookID = order.ClientID
	}
	if bookID == "" {
		return
	}
	s.sink.OnPortfolioChange(domain.PortfolioChange{
		PortfolioID: bookID,
		Reason:      reason,
		Order:       order,
		Trade:       trade,
		View:        view,
	})
}

// CanViewPortfolio reports whether actor may read the book (owner / trader / viewer).
func (s *Service) CanViewPortfolio(ctx context.Context, actorClientID, portfolioID string) error {
	_, err := s.requireAccessErr(ctx, actorClientID, domain.PortfolioRoleViewer, portfolioID)
	return err
}

// RealtimeSnapshot is the selected book's view plus open pending orders.
func (s *Service) RealtimeSnapshot(ctx context.Context, actorClientID, portfolioID string) (*domain.PortfolioView, []domain.PendingOrder, error) {
	view, err := s.View(ctx, actorClientID, portfolioID)
	if err != nil {
		return nil, nil, err
	}
	orders, err := s.ListPendingOrders(ctx, actorClientID, string(domain.PendingStatusOpen), 200, 0, view.ID)
	if err != nil {
		return view, nil, err
	}
	return view, orders, nil
}

// CreateInput creates a paper portfolio.
type CreateInput struct {
	ClientID        string
	Name            string
	StartingBalance float64
	Currency        string
}

// OrderInput is a market buy/sell.
type OrderInput struct {
	ClientID       string
	PortfolioID    string
	OwnerClientID  string
	Exchange       string
	Symbol         string
	Side           string // buy | sell
	Quantity       float64
	LotMethod      string // fifo | lifo (sells; ignored on buys)
	IdempotencyKey string // optional; same key + same request returns the original fill
	// MarkPrice, when > 0, is the last print used instead of fetching again.
	// Recurring cash buys pin the mark they sized on so a later ticker cannot
	// overspend the plan amount. Not mapped from HTTP.
	MarkPrice float64
	// RecurringPlanID, when set, re-reads that plan under lockClient before the
	// debit so a concurrent maxPrice / budget / pause / end PATCH cannot fill
	// with stale limits.
	RecurringPlanID       string
	RecurringScheduledFor time.Time
	RecurringPeriodKey    string
	RecurringNextRunAt    time.Time
	RecurringRunID        string
}

// requireBook loads a book the caller owns (owner role).
func (s *Service) requireBook(ctx context.Context, owner string, portfolioID ...string) (*domain.Portfolio, error) {
	p, _, err := s.requireAccess(ctx, owner, domain.PortfolioRoleOwner, portfolioID...)
	return p, err
}

// Create opens a new named paper portfolio with starting cash.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Portfolio, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	name, err := domain.ValidatePortfolioName(in.Name)
	if err != nil {
		return nil, err
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	if in.StartingBalance < domain.MinStartingBalance || in.StartingBalance > domain.MaxStartingBalance ||
		math.IsNaN(in.StartingBalance) || math.IsInf(in.StartingBalance, 0) {
		return nil, fmt.Errorf("%w: startingBalance must be between %g and %g", domain.ErrInvalidArgument, domain.MinStartingBalance, domain.MaxStartingBalance)
	}
	cur := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cur == "" {
		cur = domain.DefaultPaperCurrency
	}
	n, err := s.store.CountPortfolios(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxPortfoliosPerClient {
		return nil, fmt.Errorf("%w: at most %d paper portfolios per account", domain.ErrInvalidArgument, domain.MaxPortfoliosPerClient)
	}
	existing, err := s.store.ListPortfolios(ctx, clientID)
	if err != nil {
		return nil, err
	}
	for _, b := range existing {
		if strings.EqualFold(b.Name, name) {
			return nil, fmt.Errorf("%w: a portfolio named %q already exists", domain.ErrInvalidArgument, name)
		}
	}
	id := clientID
	if n > 0 {
		id = uuid.NewString()
	}
	now := time.Now().UTC()
	p, err := s.store.CreatePortfolio(ctx, domain.Portfolio{
		ID:               id,
		ClientID:         clientID,
		Name:             name,
		Currency:         cur,
		StartingBalance:  in.StartingBalance,
		CashBalance:      in.StartingBalance,
		RealizedPnLTotal: 0,
		MarginMode:       domain.MarginModeIsolated,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return nil, err
	}
	s.recordCreateSnapshot(ctx, p)
	s.recordOpeningCashMovement(ctx, p)
	return p, nil
}

// List returns every paper book for the owner (oldest first).
func (s *Service) List(ctx context.Context, clientID string) ([]domain.Portfolio, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListPortfolios(ctx, clientID)
}

// Rename changes a book's display name (unique per owner).
func (s *Service) Rename(ctx context.Context, clientID, portfolioID, name string) (*domain.Portfolio, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID)
	if err != nil {
		return nil, err
	}
	name, err = domain.ValidatePortfolioName(name)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(p.Name, name) && p.Name == name {
		return p, nil
	}
	list, err := s.store.ListPortfolios(ctx, p.ClientID)
	if err != nil {
		return nil, err
	}
	for _, b := range list {
		if b.ID != p.ID && strings.EqualFold(b.Name, name) {
			return nil, fmt.Errorf("%w: a portfolio named %q already exists", domain.ErrInvalidArgument, name)
		}
	}
	return s.store.UpdatePortfolioName(ctx, p.ClientID, p.ID, name, time.Now().UTC())
}

// Delete removes a book and all of its positions, orders, and history.
func (s *Service) Delete(ctx context.Context, clientID, portfolioID string) error {
	p, err := s.requireBook(ctx, clientID, portfolioID)
	if err != nil {
		return err
	}
	return s.store.DeletePortfolio(ctx, p.ClientID, p.ID)
}

// Get returns portfolio row or ErrNotFound.
func (s *Service) Get(ctx context.Context, clientID string, portfolioID ...string) (*domain.Portfolio, error) {
	p, _, err := s.requireAccess(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	return p, err
}

// View returns cash, reservations, positions marked to market, and P&L summary.
func (s *Service) View(ctx context.Context, clientID string, portfolioID ...string) (*domain.PortfolioView, error) {
	p, role, err := s.requireAccess(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	return s.buildView(ctx, p, role)
}

func (s *Service) buildView(ctx context.Context, p *domain.Portfolio, role domain.PortfolioShareRole) (*domain.PortfolioView, error) {
	bookID := p.BookID()
	reservedCash, err := s.store.SumReservedCash(ctx, bookID)
	if err != nil {
		return nil, err
	}
	reservedMargin, err := s.store.SumReservedMargin(ctx, bookID)
	if err != nil {
		return nil, err
	}
	positions, err := s.store.ListPositions(ctx, bookID)
	if err != nil {
		return nil, err
	}
	openLots, _ := s.store.ListTaxLots(ctx, bookID, "", "", true)
	lotsByPair := map[string][]domain.TaxLot{}
	for _, l := range openLots {
		k := string(l.Exchange) + ":" + l.Symbol
		lotsByPair[k] = append(lotsByPair[k], l)
	}
	views := make([]domain.PositionView, 0, len(positions))
	var posValue, spotUnreal float64
	for _, pos := range positions {
		mark, merr := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
		if merr != nil {
			// Skip mark on error — still show cost basis; mark=avg for display safety
			mark = pos.AvgCost
		}
		resQty, rerr := s.store.SumReservedQuantity(ctx, bookID, pos.Exchange, pos.Symbol)
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
			Lots:              lotsByPair[string(pos.Exchange)+":"+pos.Symbol],
		})
		posValue += mv
		spotUnreal += u
	}
	marginPositions, err := s.store.ListOpenMarginPositions(ctx, bookID)
	if err != nil {
		return nil, err
	}
	var marginLocked, marginUnreal float64
	for i := range marginPositions {
		s.markMarginPosition(ctx, &marginPositions[i])
		marginLocked += marginPositions[i].Margin
		marginUnreal += marginPositions[i].UnrealizedPnL
	}
	// Cash already excludes locked margin; equity adds it back + unrealized + spot marks.
	equity := p.CashBalance + posValue + marginLocked + marginUnreal
	avail := domain.AvailableCash(p.CashBalance, reservedCash+reservedMargin)
	mode := p.MarginMode
	if mode == "" {
		mode = domain.MarginModeIsolated
	}
	return &domain.PortfolioView{
		ID:                  p.ID,
		ClientID:            p.ClientID,
		Name:                p.Name,
		Currency:            p.Currency,
		StartingBalance:     p.StartingBalance,
		CashBalance:         p.CashBalance,
		NetDeposits:         p.NetDeposits,
		ContributedCapital:  domain.ContributedCapital(p.StartingBalance, p.NetDeposits),
		ReservedCash:        reservedCash,
		ReservedMargin:      reservedMargin,
		AvailableCash:       avail,
		PositionsValue:      posValue,
		MarginMode:          mode,
		MarginLocked:        marginLocked,
		MarginUnrealizedPnL: marginUnreal,
		MarginEquity:        marginLocked + marginUnreal,
		Equity:              equity,
		UnrealizedPnL:       spotUnreal + marginUnreal,
		RealizedPnLTotal:    p.RealizedPnLTotal,
		TotalPnL:            domain.PortfolioTotalPnL(equity, p.StartingBalance, p.NetDeposits),
		Positions:           views,
		MarginPositions:     marginPositions,
		Note:                paperNote,
		Role:                role,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}, nil
}

// PlaceOrder executes a paper market order at last trade price.
func (s *Service) PlaceOrder(ctx context.Context, in OrderInput) (*domain.Trade, *domain.PortfolioView, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	p, _, err := s.requireAccess(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.rejectClosedOwner(ctx, p); err != nil {
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
	lotMethod, err := domain.NormalizeLotMethod(in.LotMethod)
	if err != nil {
		return nil, nil, err
	}
	if err := domain.RequireQuoteMatchesCurrency(ex, sym, p.Currency); err != nil {
		return nil, nil, err
	}
	idempHash := hashParts("market", string(ex), sym, string(side), in.Quantity, string(lotMethod))
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, err
	} else if rec != nil {
		return s.replayTrade(ctx, rec, in.ClientID, p.ID)
	}
	var last float64
	if in.MarkPrice > 0 {
		last = in.MarkPrice
	} else {
		var lerr error
		last, lerr = s.lastPrice(ctx, string(ex), sym)
		if lerr != nil {
			return nil, nil, lerr
		}
	}
	cost := s.paperCost(ex)
	price := domain.ApplySlippage(last, side, cost.SlippageRate)
	if price <= 0 {
		return nil, nil, fmt.Errorf("%w: invalid fill price after slippage", domain.ErrInvalidArgument)
	}
	if in.RecurringPlanID != "" {
		qty, err := s.enforceRecurringLimits(ctx, clientID, in, last, cost, price)
		if err != nil {
			return nil, nil, err
		}
		in.Quantity = qty
	}
	fee := domain.FeeAmount(in.Quantity, price, cost.FeeRate)
	if side == domain.TradeSideBuy {
		base, _ := domain.SplitBaseQuote(ex, sym)
		if err := s.guardNewRisk(ctx, clientID, base, in.Quantity*price); err != nil {
			return nil, nil, err
		}
	}
	availCash, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
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
	reservedQty, err := s.store.SumReservedQuantity(ctx, clientID, ex, sym)
	if err != nil {
		return nil, nil, err
	}
	availPos := domain.AvailablePosition(posQty, reservedQty)

	now := time.Now().UTC()
	existingLots, err := s.loadOpenLots(ctx, clientID, ex, sym)
	if err != nil {
		return nil, nil, err
	}
	var (
		newCash, newQty, newAvg, realized float64
		lotOps                            *domain.LotOps
	)
	tradeID := uuid.NewString()
	switch side {
	case domain.TradeSideBuy:
		debit := domain.BuyCashDebit(in.Quantity, price, cost.FeeRate)
		if availCash+1e-9 < debit || p.CashBalance+1e-9 < debit {
			return nil, nil, fmt.Errorf("%w: insufficient cash balance", domain.ErrInvalidArgument)
		}
		unit := domain.BuyUnitCost(price, cost.FeeRate)
		_, newQty, newAvg, err = domain.ApplyBuy(availCash, in.Quantity, unit, posQty, avg)
		if err == nil {
			newCash = p.CashBalance - debit
			if newCash < 0 {
				return nil, nil, fmt.Errorf("%w: insufficient cash balance", domain.ErrInvalidArgument)
			}
			lotOps = prepareBuyLots(clientID, ex, sym, existingLots, posQty, avg, in.Quantity, unit, tradeID, now)
			merged := append(append([]domain.TaxLot(nil), existingLots...), lotOps.Created...)
			if a := domain.AvgCostFromLots(merged); a > 0 {
				newAvg = a
			}
		}
		realized = 0
	case domain.TradeSideSell:
		if in.Quantity > availPos+domain.PositionEpsilon {
			return nil, nil, fmt.Errorf("%w: insufficient position quantity", domain.ErrInvalidArgument)
		}
		lotOps, realized, newAvg, err = prepareSellLots(existingLots, clientID, ex, sym, posQty, avg, in.Quantity, price, lotMethod, tradeID, now, cost.FeeRate)
		if err == nil {
			newCash = p.CashBalance + domain.SellCashCredit(in.Quantity, price, cost.FeeRate)
			newQty = posQty - in.Quantity
			if newQty < domain.PositionEpsilon {
				newQty = 0
				newAvg = 0
			}
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
		ID:          tradeID,
		ClientID:    clientID,
		Exchange:    ex,
		Symbol:      sym,
		Side:        side,
		Quantity:    in.Quantity,
		Price:       price,
		Notional:    in.Quantity * price,
		RealizedPnL: realized,
		LotMethod:   lotMethod,
		Fee:         fee,
		LastPrice:   last,
		CreatedAt:   now,
	}
	if lotOps != nil {
		tr.LotFills = lotOps.Fills
	}
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindTrade, idempIDs{TradeID: tradeID})
	if in.RecurringPlanID != "" && in.RecurringPeriodKey != "" {
		cashOut := tr.Notional + tr.Fee
		ctx = domain.ContextWithRecurringFill(ctx, &domain.RecurringFillCommit{
			PlanID:        in.RecurringPlanID,
			LastPeriodKey: in.RecurringPeriodKey,
			NextRunAt:     in.RecurringNextRunAt,
			Run: domain.RecurringBuyRun{
				ID:     runIDOr(in.RecurringRunID, in.RecurringPlanID, in.RecurringPeriodKey),
				PlanID: in.RecurringPlanID, ClientID: clientID, PeriodKey: in.RecurringPeriodKey,
				Status: domain.RecurringBuyRunSucceeded, Amount: cashOut,
				Quantity: in.Quantity, Price: price, TradeID: tradeID,
				ScheduledFor: in.RecurringScheduledFor, ExecutedAt: now,
			},
		})
	}
	if err := s.store.ExecuteTrade(ctx, p, posOut, tr, lotOps); err != nil {
		if errors.Is(err, domain.ErrRecurringPeriodDone) {
			return s.replayRecurringPeriod(ctx, clientID, in)
		}
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayTrade(ctx, rec, in.ClientID, p.ID)
			}
		}
		return nil, nil, err
	}
	view, err := s.View(ctx, in.ClientID, p.ID)
	if err != nil {
		return &tr, nil, err
	}
	s.notifyChange(ctx, p.ID, domain.PortfolioChangeOrderFilled, nil, &tr, view)
	return &tr, view, nil
}

// enforceRecurringLimits re-reads the plan under lockClient (caller holds it)
// and applies the live maxPrice / budget / pause / end. This is the fill gate:
// a PATCH that already committed cannot lose to this buy.
func (s *Service) enforceRecurringLimits(ctx context.Context, bookID string, in OrderInput, last float64, cost domain.TradingCost, slipped float64) (float64, error) {
	plan, err := s.store.GetRecurringBuyPlan(ctx, bookID, in.RecurringPlanID)
	if err != nil || plan == nil {
		return 0, fmt.Errorf("%w: plan unavailable", domain.ErrInvalidArgument)
	}
	if plan.Status != domain.RecurringBuyActive {
		if plan.Status == domain.RecurringBuyEnded {
			return 0, fmt.Errorf("%w: plan ended", domain.ErrInvalidArgument)
		}
		return 0, fmt.Errorf("%w: plan paused", domain.ErrInvalidArgument)
	}
	if !domain.RecurringPlanAllowsRun(*plan, in.RecurringScheduledFor) {
		return 0, fmt.Errorf("%w: plan ended", domain.ErrInvalidArgument)
	}
	if reason := domain.RecurringMaxPriceBlocks(last, cost.SlippageRate, cost.FeeRate, plan.MaxPrice); reason != "" {
		return 0, fmt.Errorf("%w: %s", domain.ErrInvalidArgument, reason)
	}
	spend, reason := domain.RecurringSpendAmount(*plan)
	if reason != "" {
		return 0, fmt.Errorf("%w: %s", domain.ErrInvalidArgument, reason)
	}
	unit := domain.BuyUnitCost(slipped, cost.FeeRate)
	if unit <= 0 {
		return 0, fmt.Errorf("%w: market price unavailable", domain.ErrInvalidArgument)
	}
	qty := in.Quantity
	debit := domain.BuyCashDebit(qty, slipped, cost.FeeRate)
	if debit > spend+1e-9 {
		qty = spend / unit
		debit = domain.BuyCashDebit(qty, slipped, cost.FeeRate)
	}
	if qty < domain.MinTradeQuantity || debit+1e-9 < domain.MinRecurringBuyAmount && plan.Budget > 0 && spend < domain.MinRecurringBuyAmount {
		return 0, fmt.Errorf("%w: budget exhausted", domain.ErrInvalidArgument)
	}
	if qty < domain.MinTradeQuantity {
		return 0, fmt.Errorf("%w: buy quantity too small for amount", domain.ErrInvalidArgument)
	}
	return qty, nil
}

// ListTrades returns trade history.
func (s *Service) ListTrades(ctx context.Context, clientID string, limit, offset int, portfolioID ...string) ([]domain.Trade, int, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, 0, err
	}
	clientID = p.BookID()
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	total, err := s.store.CountTrades(ctx, clientID)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListTrades(ctx, clientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	fills, ferr := s.store.ListTaxLotFillsForTrades(ctx, ids)
	if ferr == nil && len(fills) > 0 {
		byTrade := map[string][]domain.TaxLotFill{}
		for _, f := range fills {
			byTrade[f.TradeID] = append(byTrade[f.TradeID], f)
		}
		for i := range list {
			list[i].LotFills = byTrade[list[i].ID]
		}
	}
	return list, total, nil
}

// PendingOrderInput creates a limit, stop, or trailing_stop resting order.
type PendingOrderInput struct {
	ClientID      string
	PortfolioID   string
	OwnerClientID string
	Exchange      string
	Symbol        string
	Type          string // limit_buy | limit_sell | stop_loss | trailing_stop
	Quantity      float64
	TriggerPrice  float64 // limit/stop; ignored for trailing_stop (derived from peak)
	// TrailType: percent | offset (trailing_stop only).
	TrailType string
	// TrailValue: fraction e.g. 0.05 or fixed price offset (trailing_stop only).
	TrailValue     float64
	TimeInForce    string     // gtc (default) | ioc | fok
	ExpiresAt      *time.Time // optional; GTC only
	LotMethod      string     // fifo | lifo for sell types
	IdempotencyKey string
}

// OCOOrderInput places a linked take-profit limit_sell + stop_loss for the same quantity.
type OCOOrderInput struct {
	ClientID        string
	PortfolioID     string
	OwnerClientID   string
	Exchange        string
	Symbol          string
	Quantity        float64
	TakeProfitPrice float64 // limit_sell trigger
	StopLossPrice   float64 // stop_loss trigger
	ExpiresAt       *time.Time
	LotMethod       string
	IdempotencyKey  string
}

// BracketOrderInput places a limit-buy entry with inactive take-profit + stop-loss exits.
// Exits stay pending until entry fills; exit size tracks cumulative entry filled quantity.
type BracketOrderInput struct {
	ClientID        string
	PortfolioID     string
	OwnerClientID   string
	Exchange        string
	Symbol          string
	Quantity        float64
	EntryPrice      float64 // limit_buy trigger
	TakeProfitPrice float64
	StopLossPrice   float64
	ExpiresAt       *time.Time
	LotMethod       string
	IdempotencyKey  string
}

// PlaceBracketOrder creates entry (open limit_buy) + TP/SL (pending) linked by bracket id.
// Exit OCO becomes open only for filled entry size; peer cancel prevents double-selling.
func (s *Service) PlaceBracketOrder(ctx context.Context, in BracketOrderInput) (entry, tp, sl *domain.PendingOrder, err error) {
	if s.store == nil {
		return nil, nil, nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, nil, err
	}
	if _, err := domain.NormalizeLotMethod(in.LotMethod); err != nil {
		return nil, nil, nil, err
	}
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	fresh, ferr := s.store.GetPortfolio(ctx, clientID)
	if ferr != nil {
		return nil, nil, nil, ferr
	}
	p = fresh
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, nil, nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	if err := domain.ValidateBracketPrices(in.EntryPrice, in.TakeProfitPrice, in.StopLossPrice); err != nil {
		return nil, nil, nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, nil, nil, err
	}
	idempKey, err := domain.NormalizeIdempotencyKey(in.IdempotencyKey)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := domain.RequireQuoteMatchesCurrency(ex, sym, p.Currency); err != nil {
		return nil, nil, nil, err
	}
	idempHash := hashParts("bracket", string(ex), sym, in.Quantity, in.EntryPrice, in.TakeProfitPrice, in.StopLossPrice, in.LotMethod)
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, nil, err
	} else if rec != nil {
		return s.replayBracket(ctx, rec)
	}
	base, _ := domain.SplitBaseQuote(ex, sym)
	if err := s.guardNewRisk(ctx, clientID, base, in.Quantity*in.EntryPrice); err != nil {
		return nil, nil, nil, err
	}
	n, err := s.store.CountOpenPendingOrders(ctx, clientID)
	if err != nil {
		return nil, nil, nil, err
	}
	// Entry counts as open; exits are pending until fill.
	if n+1 > domain.MaxOpenPendingOrders {
		return nil, nil, nil, fmt.Errorf("%w: max open pending orders (%d) reached", domain.ErrInvalidArgument, domain.MaxOpenPendingOrders)
	}
	need := domain.BuyReserveCash(in.Quantity, in.EntryPrice, s.paperCost(ex))
	avail, rerr := s.availableCashForTrading(ctx, clientID, p.CashBalance)
	if rerr != nil {
		return nil, nil, nil, rerr
	}
	if avail+1e-9 < need {
		return nil, nil, nil, fmt.Errorf("%w: insufficient available cash to reserve (need %g, available %g)", domain.ErrInvalidArgument, need, avail)
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if in.ExpiresAt != nil && !in.ExpiresAt.IsZero() {
		exp := in.ExpiresAt.UTC()
		if !exp.After(now) {
			return nil, nil, nil, fmt.Errorf("%w: expiresAt must be in the future", domain.ErrInvalidArgument)
		}
		expiresAt = &exp
	}
	bracketID := uuid.NewString()
	ocoID := uuid.NewString()
	entryID := uuid.NewString()
	tpID := uuid.NewString()
	slID := uuid.NewString()
	entryOrd := domain.PendingOrder{
		ID: entryID, ClientID: clientID, Exchange: ex, Symbol: sym,
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy,
		Quantity: in.Quantity, RemainingQuantity: in.Quantity,
		TriggerPrice: in.EntryPrice, ReservedCash: need,
		TimeInForce: domain.TimeInForceGTC, ExpiresAt: expiresAt,
		Status: domain.PendingStatusOpen, BracketID: bracketID, BracketRole: domain.BracketRoleEntry,
		CreatedAt: now, UpdatedAt: now,
	}
	// Exits inactive: pending, zero size until entry fills.
	tpOrd := domain.PendingOrder{
		ID: tpID, ClientID: clientID, Exchange: ex, Symbol: sym,
		Type: domain.PendingLimitSell, Side: domain.TradeSideSell,
		Quantity: 0, RemainingQuantity: 0, TriggerPrice: in.TakeProfitPrice,
		TimeInForce: domain.TimeInForceGTC, ExpiresAt: expiresAt,
		Status:     domain.PendingStatusPending,
		OCOGroupID: ocoID, OCOPeerID: slID,
		BracketID: bracketID, BracketRole: domain.BracketRoleTakeProfit,
		LotMethod: pendingLotMethod(in.LotMethod),
		CreatedAt: now, UpdatedAt: now,
	}
	slOrd := domain.PendingOrder{
		ID: slID, ClientID: clientID, Exchange: ex, Symbol: sym,
		Type: domain.PendingStopLoss, Side: domain.TradeSideSell,
		Quantity: 0, RemainingQuantity: 0, TriggerPrice: in.StopLossPrice,
		TimeInForce: domain.TimeInForceGTC, ExpiresAt: expiresAt,
		Status:     domain.PendingStatusPending,
		OCOGroupID: ocoID, OCOPeerID: tpID,
		BracketID: bracketID, BracketRole: domain.BracketRoleStopLoss,
		LotMethod: pendingLotMethod(in.LotMethod),
		CreatedAt: now, UpdatedAt: now,
	}
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindBracket, idempIDs{EntryID: entryID, TakeProfitID: tpID, StopLossID: slID})
	entry, tp, sl, err = s.store.CreateBracket(ctx, entryOrd, tpOrd, slOrd)
	if err != nil {
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayBracket(ctx, rec)
			}
		}
		return nil, nil, nil, err
	}
	s.notifyChange(ctx, clientID, domain.PortfolioChangeOrderPlaced, entry, nil, nil)
	return entry, tp, sl, nil
}

// PlaceOCOOrder creates take-profit + stop-loss legs that share one reserved position size.
// When one leg fully fills, the peer is canceled; partial fills shrink both remainings.
func (s *Service) PlaceOCOOrder(ctx context.Context, in OCOOrderInput) (tp, sl *domain.PendingOrder, err error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := domain.NormalizeLotMethod(in.LotMethod); err != nil {
		return nil, nil, err
	}
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	if _, ferr := s.store.GetPortfolio(ctx, clientID); ferr != nil {
		return nil, nil, ferr
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	if err := domain.ValidateOCOPrices(in.TakeProfitPrice, in.StopLossPrice); err != nil {
		return nil, nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, nil, err
	}
	idempKey, err := domain.NormalizeIdempotencyKey(in.IdempotencyKey)
	if err != nil {
		return nil, nil, err
	}
	idempHash := hashParts("oco", string(ex), sym, in.Quantity, in.TakeProfitPrice, in.StopLossPrice, in.LotMethod)
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, err
	} else if rec != nil {
		return s.replayOCO(ctx, rec)
	}
	n, err := s.store.CountOpenPendingOrders(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	// Two legs count toward the open-order cap.
	if n+2 > domain.MaxOpenPendingOrders {
		return nil, nil, fmt.Errorf("%w: max open pending orders (%d) reached", domain.ErrInvalidArgument, domain.MaxOpenPendingOrders)
	}
	var held float64
	pos, perr := s.store.GetPosition(ctx, clientID, ex, sym)
	if perr == nil && pos != nil {
		held = pos.Quantity
	} else if perr != nil && perr != domain.ErrNotFound {
		return nil, nil, perr
	}
	resQty, rerr := s.store.SumReservedQuantity(ctx, clientID, ex, sym)
	if rerr != nil {
		return nil, nil, rerr
	}
	avail := domain.AvailablePosition(held, resQty)
	if avail+domain.PositionEpsilon < in.Quantity {
		return nil, nil, fmt.Errorf("%w: insufficient available position to reserve (need %g, available %g)", domain.ErrInvalidArgument, in.Quantity, avail)
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if in.ExpiresAt != nil && !in.ExpiresAt.IsZero() {
		exp := in.ExpiresAt.UTC()
		if !exp.After(now) {
			return nil, nil, fmt.Errorf("%w: expiresAt must be in the future", domain.ErrInvalidArgument)
		}
		expiresAt = &exp
	}
	groupID := uuid.NewString()
	tpID := uuid.NewString()
	slID := uuid.NewString()
	// Reservation is counted once per OCO group (SumReservedQuantity); both legs store remaining size.
	tpOrd := domain.PendingOrder{
		ID: tpID, ClientID: clientID, Exchange: ex, Symbol: sym,
		Type: domain.PendingLimitSell, Side: domain.TradeSideSell,
		Quantity: in.Quantity, FilledQuantity: 0, RemainingQuantity: in.Quantity,
		TriggerPrice: in.TakeProfitPrice, ReservedQuantity: in.Quantity,
		TimeInForce: domain.TimeInForceGTC, ExpiresAt: expiresAt,
		Status: domain.PendingStatusOpen, OCOGroupID: groupID, OCOPeerID: slID,
		LotMethod: pendingLotMethod(in.LotMethod),
		CreatedAt: now, UpdatedAt: now,
	}
	slOrd := domain.PendingOrder{
		ID: slID, ClientID: clientID, Exchange: ex, Symbol: sym,
		Type: domain.PendingStopLoss, Side: domain.TradeSideSell,
		Quantity: in.Quantity, FilledQuantity: 0, RemainingQuantity: in.Quantity,
		TriggerPrice: in.StopLossPrice, ReservedQuantity: in.Quantity,
		TimeInForce: domain.TimeInForceGTC, ExpiresAt: expiresAt,
		Status: domain.PendingStatusOpen, OCOGroupID: groupID, OCOPeerID: tpID,
		LotMethod: pendingLotMethod(in.LotMethod),
		CreatedAt: now, UpdatedAt: now,
	}
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindOCO, idempIDs{TakeProfitID: tpID, StopLossID: slID})
	tp, sl, err = s.store.CreateOCOPair(ctx, tpOrd, slOrd)
	if err != nil {
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayOCO(ctx, rec)
			}
		}
		return nil, nil, err
	}
	s.notifyChange(ctx, clientID, domain.PortfolioChangeOrderPlaced, tp, nil, nil)
	return tp, sl, nil
}

// PlacePendingOrder creates an open resting order and reserves cash or position.
func (s *Service) PlacePendingOrder(ctx context.Context, in PendingOrderInput) (*domain.PendingOrder, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, err
	}
	if err := s.rejectClosedOwner(ctx, p); err != nil {
		return nil, err
	}
	idempKey, err := domain.NormalizeIdempotencyKey(in.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if _, err := domain.NormalizeLotMethod(in.LotMethod); err != nil {
		return nil, err
	}
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	fresh, ferr := s.store.GetPortfolio(ctx, clientID)
	if ferr != nil {
		return nil, ferr
	}
	p = fresh
	typ := domain.PendingOrderType(strings.ToLower(strings.TrimSpace(in.Type)))
	if !domain.IsValidPendingOrderType(string(typ)) {
		return nil, fmt.Errorf("%w: type must be limit_buy, limit_sell, stop_loss, or trailing_stop", domain.ErrInvalidArgument)
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	isTrail := typ == domain.PendingTrailingStop
	var trailType string
	var trailValue, trailPeak, trigger float64
	if isTrail {
		var nerr error
		trailType, nerr = domain.NormalizeTrailType(in.TrailType)
		if nerr != nil {
			return nil, nerr
		}
		if err := domain.ValidateTrailValue(trailType, in.TrailValue); err != nil {
			return nil, err
		}
		trailValue = in.TrailValue
	} else {
		if in.TriggerPrice < domain.MinTriggerPrice || in.TriggerPrice > domain.MaxTriggerPrice ||
			math.IsNaN(in.TriggerPrice) || math.IsInf(in.TriggerPrice, 0) {
			return nil, fmt.Errorf("%w: triggerPrice out of range", domain.ErrInvalidArgument)
		}
		trigger = in.TriggerPrice
	}
	tif, err := domain.NormalizeTimeInForce(in.TimeInForce)
	if err != nil {
		return nil, err
	}
	if isTrail && tif != domain.TimeInForceGTC {
		return nil, fmt.Errorf("%w: trailing_stop only supports gtc", domain.ErrInvalidArgument)
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	expKey := ""
	if in.ExpiresAt != nil && !in.ExpiresAt.IsZero() {
		expKey = in.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	idempHash := hashParts("pending", string(typ), string(ex), sym, in.Quantity, in.TriggerPrice, trailType, in.TrailValue, string(tif), in.LotMethod, expKey)
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, err
	} else if rec != nil {
		return s.replayPending(ctx, rec)
	}
	if typ == domain.PendingLimitBuy {
		base, _ := domain.SplitBaseQuote(ex, sym)
		if err := s.guardNewRisk(ctx, clientID, base, in.Quantity*trigger); err != nil {
			return nil, err
		}
	}
	if isTrail {
		last, lerr := s.lastPrice(ctx, string(ex), sym)
		if lerr != nil || last <= 0 {
			return nil, fmt.Errorf("%w: market price unavailable to seed trailing stop", domain.ErrUpstream)
		}
		trailPeak = last
		trigger = domain.TrailStopPrice(trailPeak, trailValue, trailType)
		if trigger <= 0 {
			return nil, fmt.Errorf("%w: trail produces non-positive stop price at current market", domain.ErrInvalidArgument)
		}
	}
	if err := domain.RequireQuoteMatchesCurrency(ex, sym, p.Currency); err != nil {
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
		need := domain.BuyReserveCash(in.Quantity, trigger, s.paperCost(ex))
		avail, rerr := s.availableCashForTrading(ctx, clientID, p.CashBalance)
		if rerr != nil {
			return nil, rerr
		}
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
		TriggerPrice:      trigger,
		ReservedCash:      reservedCash,
		ReservedQuantity:  reservedQty,
		TimeInForce:       tif,
		ExpiresAt:         expiresAt,
		Status:            domain.PendingStatusOpen,
		TrailType:         trailType,
		TrailValue:        trailValue,
		TrailPeak:         trailPeak,
		LotMethod:         pendingLotMethod(in.LotMethod),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindPending, idempIDs{OrderID: o.ID})
	created, err := s.store.CreatePendingOrder(ctx, o)
	if err != nil {
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayPending(ctx, rec)
			}
		}
		return nil, err
	}
	s.notifyChange(ctx, clientID, domain.PortfolioChangeOrderPlaced, created, nil, nil)
	return created, nil
}

// AmendPendingOrderInput changes trigger price and/or remaining size of an open pending order.
// At least one of TriggerPrice or RemainingQuantity must be set.
type AmendPendingOrderInput struct {
	ClientID          string
	PortfolioID       string
	OwnerClientID     string
	OrderID           string
	TriggerPrice      *float64
	RemainingQuantity *float64
}

// PendingOrderDetail is one order plus last price and amend capacity for the edit screen.
type PendingOrderDetail struct {
	Order                     domain.PendingOrder
	LastPrice                 float64
	Editable                  bool
	AvailableCashForOrder     float64
	AvailableQuantityForOrder float64
	MaxRemainingQuantity      float64
	MinRemainingQuantity      float64
}

// GetPendingOrder returns one pending order for the client.
func (s *Service) GetPendingOrder(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.PendingOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetPendingOrder(ctx, p.BookID(), id)
}

// GetPendingOrderDetail returns the order plus last price and amend hints for the edit UI.
func (s *Service) GetPendingOrderDetail(ctx context.Context, clientID, id string, portfolioID ...string) (*PendingOrderDetail, error) {
	o, err := s.GetPendingOrder(ctx, clientID, id, portfolioID...)
	if err != nil {
		return nil, err
	}
	p, err := s.store.GetPortfolio(ctx, o.ClientID)
	if err != nil {
		return nil, err
	}
	cashFor, qtyFor, err := s.amendCapacity(ctx, p, *o)
	if err != nil {
		return nil, err
	}
	last, _ := s.lastPrice(ctx, string(o.Exchange), o.Symbol)
	d := &PendingOrderDetail{
		Order:                     *o,
		LastPrice:                 last,
		Editable:                  domain.CanAmendPendingOrder(*o) == nil,
		AvailableCashForOrder:     cashFor,
		AvailableQuantityForOrder: qtyFor,
		MaxRemainingQuantity:      domain.MaxAmendRemaining(o.Side, o.TriggerPrice, cashFor, qtyFor),
		MinRemainingQuantity:      domain.MinTradeQuantity,
	}
	return d, nil
}

// AmendPendingOrder changes trigger and/or remaining of an open limit/stop in place (same id).
// Recalculates reservations; if the new price is already marketable, one fill attempt runs immediately.
func (s *Service) AmendPendingOrder(ctx context.Context, in AmendPendingOrderInput) (*domain.PendingOrder, *domain.PortfolioView, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	if in.TriggerPrice == nil && in.RemainingQuantity == nil {
		return nil, nil, fmt.Errorf("%w: triggerPrice or remainingQuantity is required", domain.ErrInvalidArgument)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	clientID := p.BookID()
	id := strings.TrimSpace(in.OrderID)
	if id == "" {
		return nil, nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	o, err := s.store.GetPendingOrder(ctx, clientID, id)
	if err != nil {
		return nil, nil, err
	}
	if err := domain.CanAmendPendingOrder(*o); err != nil {
		return nil, nil, err
	}
	newRemaining := o.RemainingQuantity
	if in.RemainingQuantity != nil {
		if err := domain.ValidateAmendRemaining(*in.RemainingQuantity); err != nil {
			return nil, nil, err
		}
		newRemaining = *in.RemainingQuantity
	}
	newTrigger := o.TriggerPrice
	if in.TriggerPrice != nil {
		if err := domain.ValidateAmendTriggerPrice(*in.TriggerPrice); err != nil {
			return nil, nil, err
		}
		newTrigger = *in.TriggerPrice
	}
	unchanged := math.Abs(newRemaining-o.RemainingQuantity) <= 1e-12 && math.Abs(newTrigger-o.TriggerPrice) <= 1e-12
	if unchanged {
		view, verr := s.View(ctx, in.ClientID, p.ID)
		if verr != nil {
			return o, nil, verr
		}
		return o, view, nil
	}

	cashFor, qtyFor, err := s.amendCapacity(ctx, p, *o)
	if err != nil {
		return nil, nil, err
	}
	var reservedCash, reservedQty float64
	switch o.Side {
	case domain.TradeSideBuy:
		need := domain.BuyReserveCash(newRemaining, newTrigger, s.paperCost(o.Exchange))
		if cashFor+1e-9 < need {
			return nil, nil, fmt.Errorf("%w: insufficient available cash to reserve (need %g, available %g)", domain.ErrInvalidArgument, need, cashFor)
		}
		if need > o.ReservedCash+1e-9 {
			base, _ := domain.SplitBaseQuote(o.Exchange, o.Symbol)
			if err := s.guardNewRisk(ctx, clientID, base, need); err != nil {
				return nil, nil, err
			}
		}
		reservedCash = need
	case domain.TradeSideSell:
		if qtyFor+domain.PositionEpsilon < newRemaining {
			return nil, nil, fmt.Errorf("%w: insufficient available position to reserve (need %g, available %g)", domain.ErrInvalidArgument, newRemaining, qtyFor)
		}
		reservedQty = newRemaining
	default:
		return nil, nil, fmt.Errorf("%w: invalid order side", domain.ErrInvalidArgument)
	}

	now := time.Now().UTC()
	updated, err := s.store.AmendPendingOrder(ctx, clientID, id, domain.PendingOrderAmend{
		RemainingQuantity: newRemaining,
		TriggerPrice:      newTrigger,
		Quantity:          domain.AmendOriginalQuantity(o.FilledQuantity, newRemaining),
		ReservedCash:      reservedCash,
		ReservedQuantity:  reservedQty,
		ExpectedRemaining: o.RemainingQuantity,
		ExpectedTrigger:   o.TriggerPrice,
		At:                now,
	})
	if err != nil {
		return nil, nil, err
	}

	if last, lerr := s.lastPrice(ctx, string(updated.Exchange), updated.Symbol); lerr == nil && last > 0 {
		if filled, ok, ferr := s.tryFillPendingOrderLocked(ctx, *updated, last, 0); ferr == nil && ok && filled != nil {
			updated = filled
		}
	}
	view, err := s.View(ctx, in.ClientID, p.ID)
	if err != nil {
		return updated, nil, err
	}
	s.notifyChange(ctx, p.ID, domain.PortfolioChangeOrderAmended, updated, nil, view)
	return updated, view, nil
}

// amendCapacity returns cash/qty available to back this order, including its current reservation.
func (s *Service) amendCapacity(ctx context.Context, p *domain.Portfolio, o domain.PendingOrder) (cashFor, qtyFor float64, err error) {
	bookID := p.BookID()
	availCash, err := s.availableCashForTrading(ctx, bookID, p.CashBalance)
	if err != nil {
		return 0, 0, err
	}
	cashFor = availCash
	if o.Status == domain.PendingStatusOpen && o.Side == domain.TradeSideBuy {
		cashFor += o.ReservedCash
	}
	var held float64
	pos, perr := s.store.GetPosition(ctx, bookID, o.Exchange, o.Symbol)
	if perr == nil && pos != nil {
		held = pos.Quantity
	} else if perr != nil && perr != domain.ErrNotFound {
		return 0, 0, perr
	}
	resQty, rerr := s.store.SumReservedQuantity(ctx, bookID, o.Exchange, o.Symbol)
	if rerr != nil {
		return 0, 0, rerr
	}
	qtyFor = domain.AvailablePosition(held, resQty)
	if o.Status == domain.PendingStatusOpen && o.Side == domain.TradeSideSell {
		qtyFor += o.ReservedQuantity
	}
	return cashFor, qtyFor, nil
}

// ListPendingOrders returns pending orders for a client (default: open only).
func (s *Service) ListPendingOrders(ctx context.Context, clientID string, status string, limit, offset int, portfolioID ...string) ([]domain.PendingOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
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
		case domain.PendingStatusOpen, domain.PendingStatusFilled, domain.PendingStatusCanceled,
			domain.PendingStatusRejected, domain.PendingStatusPending:
		default:
			return nil, fmt.Errorf("%w: status must be open, filled, canceled, rejected, pending, or all", domain.ErrInvalidArgument)
		}
	}
	return s.store.ListPendingOrders(ctx, clientID, st, limit, offset)
}

// CancelOpenOrdersInput cancels every open/pending paper order, or one market.
type CancelOpenOrdersInput struct {
	ClientID      string
	PortfolioID   string
	OwnerClientID string
	Exchange      string // optional; default binance when Symbol is set
	Symbol        string // empty = all markets (or all pairs on Exchange if only exchange is set)
}

// CancelOpenPendingOrders cancels open GTC/IOC/FOK/OCO/bracket/trailing orders in one action.
// Empty Symbol (and empty Exchange) = all markets. Symbol set = that pair. Exchange only = that venue.
func (s *Service) CancelOpenPendingOrders(ctx context.Context, in CancelOpenOrdersInput) ([]domain.PendingOrder, *domain.PortfolioView, error) {
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	if _, err := s.store.GetPortfolio(ctx, clientID); err != nil {
		return nil, nil, err
	}
	var ex domain.Exchange
	var sym string
	if strings.TrimSpace(in.Symbol) != "" {
		ex, sym, err = normalizeExchangeSymbol(in.Exchange, in.Symbol)
		if err != nil {
			return nil, nil, err
		}
	} else if strings.TrimSpace(in.Exchange) != "" {
		raw := strings.TrimSpace(in.Exchange)
		if !domain.IsValidExchange(raw) {
			return nil, nil, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(raw)
	}
	list, err := s.store.CancelOpenPendingOrders(ctx, clientID, ex, sym, time.Now().UTC(), domain.CancelReasonUser)
	if err != nil {
		return nil, nil, err
	}
	view, err := s.View(ctx, in.ClientID, p.ID)
	if err != nil {
		return list, nil, err
	}
	s.notifyChange(ctx, p.ID, domain.PortfolioChangeOrderCancelled, nil, nil, view)
	return list, view, nil
}

// CancelPendingOrder cancels an open order; filled/canceled/rejected cannot be canceled.
func (s *Service) CancelPendingOrder(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.PendingOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleTrader, portfolioID...)
	if err != nil {
		return nil, err
	}
	unlock := s.lockClient(clientID)
	defer unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	canceled, err := s.store.CancelPendingOrder(ctx, p.BookID(), id, time.Now().UTC(), domain.CancelReasonUser)
	if err != nil {
		return nil, err
	}
	s.notifyChange(ctx, p.BookID(), domain.PortfolioChangeOrderCancelled, canceled, nil, nil)
	return canceled, nil
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
func (s *Service) ProcessOpenOrder(ctx context.Context, o domain.PendingOrder, lastPrice float64, now time.Time, maxFillQty float64) (out *domain.PendingOrder, changed bool, err error) {
	defer func() {
		if err == nil && changed && out != nil {
			reason := domain.PortfolioChangeOrderUpdated
			switch out.Status {
			case domain.PendingStatusFilled:
				reason = domain.PortfolioChangeOrderFilled
			case domain.PendingStatusCanceled, domain.PendingStatusRejected:
				reason = domain.PortfolioChangeOrderCancelled
			}
			s.notifyChange(ctx, o.ClientID, reason, out, nil, nil)
		}
	}()
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	if o.Status != domain.PendingStatusOpen {
		return nil, false, nil
	}
	unlock := s.lockClient(o.ClientID)
	defer unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
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

	// Trailing stop: ratchet peak/stop on favorable moves before evaluating trigger.
	// Gaps below the stop still fire (last <= trigger after update).
	if o.Type == domain.PendingTrailingStop && lastPrice > 0 {
		newPeak, newStop, moved := domain.AdvanceTrailingStop(o.TrailPeak, lastPrice, o.TrailValue, o.TrailType)
		if moved && newPeak > o.TrailPeak+1e-15 {
			okUp, uerr := s.store.UpdatePendingTrail(ctx, o.ID, newPeak, newStop, now)
			if uerr != nil {
				return nil, false, uerr
			}
			if okUp {
				o.TrailPeak = newPeak
				o.TriggerPrice = newStop
			} else {
				// Reload in case another worker advanced the trail.
				if cur, gerr := s.store.GetPendingOrder(ctx, o.ClientID, o.ID); gerr == nil && cur != nil {
					o = *cur
				}
			}
		} else if o.TriggerPrice <= 0 && newStop > 0 {
			o.TriggerPrice = newStop
			o.TrailPeak = newPeak
		}
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

	// OCO legs: only the winner for this tick may fill (handled by ProcessOCOPair / caller).
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
		cost := s.paperCost(o.Exchange)
		fill := domain.ApplySlippage(lastPrice, domain.TradeSideBuy, cost.SlippageRate)
		return domain.MaxBuyFillQty(remaining, o.ReservedCash, fill, cost.FeeRate)
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

// ProcessOCOPair evaluates an OCO take-profit + stop-loss pair against one last price.
// At most one leg fills per tick; the peer is canceled on full fill or reduced on partial fill.
func (s *Service) ProcessOCOPair(ctx context.Context, a, b domain.PendingOrder, lastPrice float64, now time.Time, maxFillQty float64) (*domain.PendingOrder, bool, error) {
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	unlock := s.lockClient(a.ClientID)
	defer unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	// Fresh reload both legs.
	var tp, sl *domain.PendingOrder
	for _, o := range []domain.PendingOrder{a, b} {
		cur, err := s.store.GetPendingOrder(ctx, o.ClientID, o.ID)
		if err != nil {
			if err == domain.ErrNotFound {
				continue
			}
			return nil, false, err
		}
		if cur.Status != domain.PendingStatusOpen {
			continue
		}
		// Expiry on either leg cancels the group.
		if domain.PendingOrderExpired(*cur, now) {
			_ = s.store.CancelOCOGroup(ctx, cur.ClientID, cur.OCOGroupID, now, domain.CancelReasonExpired)
			return cur, true, nil
		}
		switch cur.Type {
		case domain.PendingLimitSell:
			tp = cur
		case domain.PendingStopLoss:
			sl = cur
		}
	}
	winner := domain.OCOWinnerForTick(tp, sl, lastPrice)
	if winner == nil {
		return nil, false, nil
	}
	return s.tryFillPendingOrderLocked(ctx, *winner, lastPrice, maxFillQty)
}

// TryFillPendingOrder evaluates one open order against lastPrice and applies a fill.
// maxFillQty > 0 caps this execution (partial fill); <= 0 fills as much remaining as possible.
// Returns (order, true, nil) when any quantity was filled; (nil, false, nil) when not triggered / no-op.
// Prefer ProcessOpenOrder for TIF/expiry handling. OCO legs use ExecuteOCOFill (peer sync/cancel).
func (s *Service) TryFillPendingOrder(ctx context.Context, o domain.PendingOrder, lastPrice, maxFillQty float64) (out *domain.PendingOrder, filled bool, err error) {
	defer func() {
		if err == nil && filled && out != nil {
			s.notifyChange(ctx, o.ClientID, domain.PortfolioChangeOrderFilled, out, nil, nil)
		}
	}()
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	if o.Status != domain.PendingStatusOpen {
		return nil, false, nil
	}
	// Direct callers (tests, amend fill attempt). ProcessOpenOrder/ProcessOCOPair
	// already hold lockClient and call tryFillPendingOrderLocked instead.
	unlock := s.lockClient(o.ClientID)
	defer unlock()
	return s.tryFillPendingOrderLocked(ctx, o, lastPrice, maxFillQty)
}

func (s *Service) tryFillPendingOrderLocked(ctx context.Context, o domain.PendingOrder, lastPrice, maxFillQty float64) (*domain.PendingOrder, bool, error) {
	if s.ownerClosed(ctx, o.ClientID) {
		return nil, false, nil
	}
	// Always re-read under the book lock so concurrent workers never fill from a stale snapshot.
	fresh, ferr := s.store.GetPendingOrder(ctx, o.ClientID, o.ID)
	if ferr != nil {
		if ferr == domain.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, ferr
	}
	o = *fresh
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
	cost := s.paperCost(o.Exchange)
	fillPrice := domain.ApplySlippage(lastPrice, o.Side, cost.SlippageRate)
	if fillPrice <= 0 {
		return nil, false, nil
	}
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
		// Bound by remaining reserved cash at slipped fill + fee.
		maxByRes := domain.MaxBuyFillQty(remaining, o.ReservedCash, fillPrice, cost.FeeRate)
		if maxFillQty > 0 && maxFillQty < maxByRes {
			fillQty = maxFillQty
		} else {
			fillQty = maxByRes
		}
		fillQty = domain.ClampFillQty(remaining, fillQty, 0)
	case domain.TradeSideSell:
		// Bound by remaining size (OCO: reserved may mirror remaining on both legs).
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

	existingLots, lerr := s.loadOpenLots(ctx, o.ClientID, o.Exchange, o.Symbol)
	if lerr != nil {
		return nil, false, lerr
	}
	lotMethod, _ := domain.NormalizeLotMethod(string(o.LotMethod))
	tradeID := uuid.NewString()
	var newCash, newQty, newAvg, realized float64
	var lotOps *domain.LotOps
	fee := domain.FeeAmount(fillQty, fillPrice, cost.FeeRate)
	switch o.Side {
	case domain.TradeSideBuy:
		debit := domain.BuyCashDebit(fillQty, fillPrice, cost.FeeRate)
		unit := domain.BuyUnitCost(fillPrice, cost.FeeRate)
		newCash, newQty, newAvg, err = domain.ApplyBuy(p.CashBalance, fillQty, unit, posQty, avg)
		realized = 0
		if err == nil {
			newCash = p.CashBalance - debit
			lotOps = prepareBuyLots(o.ClientID, o.Exchange, o.Symbol, existingLots, posQty, avg, fillQty, unit, tradeID, now)
			merged := append(append([]domain.TaxLot(nil), existingLots...), lotOps.Created...)
			if a := domain.AvgCostFromLots(merged); a > 0 {
				newAvg = a
			}
		}
	case domain.TradeSideSell:
		lotOps, realized, newAvg, err = prepareSellLots(existingLots, o.ClientID, o.Exchange, o.Symbol, posQty, avg, fillQty, fillPrice, lotMethod, tradeID, now, cost.FeeRate)
		if err == nil {
			newCash = p.CashBalance + domain.SellCashCredit(fillQty, fillPrice, cost.FeeRate)
			newQty = posQty - fillQty
			if newQty < domain.PositionEpsilon {
				newQty = 0
				newAvg = 0
			}
		}
	}
	if err != nil {
		// Reservation should prevent this; fail closed and release remainder.
		reason := err.Error()
		if o.IsOCO() && o.OCOGroupID != "" {
			_ = s.store.CancelOCOGroup(ctx, o.ClientID, o.OCOGroupID, now, reason)
			return nil, false, nil
		}
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
	updated.FillPrice = fillPrice
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
			updated.ReservedCash = domain.AfterBuyFillReservation(remainingAfter, o.TriggerPrice, cost)
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
		ID: tradeID, ClientID: o.ClientID, Exchange: o.Exchange, Symbol: o.Symbol,
		Side: o.Side, Quantity: fillQty, Price: fillPrice, Notional: fillQty * fillPrice,
		RealizedPnL: realized, PendingOrderID: o.ID, LotMethod: lotMethod,
		Fee: fee, LastPrice: lastPrice, CreatedAt: now,
	}
	if lotOps != nil {
		tr.LotFills = lotOps.Fills
	}
	updated.FillTradeID = tr.ID

	if o.IsOCO() && o.OCOPeerID != "" {
		peer, perr := s.store.GetPendingOrder(ctx, o.ClientID, o.OCOPeerID)
		if perr != nil && perr != domain.ErrNotFound {
			return nil, false, perr
		}
		if perr == domain.ErrNotFound {
			peer = nil
		}
		if err := s.store.ExecuteOCOFill(ctx, &updated, peer, p, posOut, tr, now, lotOps); err != nil {
			if err == domain.ErrNotFound || err == domain.ErrConflict {
				return nil, false, nil
			}
			return nil, false, err
		}
	} else {
		if err := s.store.ExecutePendingFill(ctx, &updated, p, posOut, tr, now, lotOps); err != nil {
			if err == domain.ErrNotFound || err == domain.ErrConflict {
				return nil, false, nil
			}
			return nil, false, err
		}
	}
	// Bracket entry: grow/activate TP+SL to match cumulative filled size (exits stay OCO).
	if o.IsBracketEntry() && o.BracketID != "" {
		_ = s.store.SyncBracketExitsToFilled(ctx, o.ClientID, o.BracketID, updated.FilledQuantity, now)
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
	if tkr.Halted {
		return 0, fmt.Errorf("%w: last price is a halted delist print", domain.ErrUpstream)
	}
	p, err := strconv.ParseFloat(tkr.LastPrice, 64)
	if err != nil || p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
		return 0, fmt.Errorf("%w: invalid last price", domain.ErrUpstream)
	}
	return p, nil
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
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
