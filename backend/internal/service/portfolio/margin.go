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

// MarginOrderInput opens a leveraged long/short via market or limit.
type MarginOrderInput struct {
	ClientID       string
	PortfolioID    string
	OwnerClientID  string
	Exchange       string
	Symbol         string
	Side           string // long | short
	Type           string // market | limit
	Quantity       float64
	Leverage       int
	LimitPrice     float64  // required for limit
	StopLoss       *float64 // optional
	TakeProfit     *float64 // optional
	IdempotencyKey string
}

// MarginAdjustInput adds (positive) or removes (negative) margin from an isolated position.
type MarginAdjustInput struct {
	ClientID      string
	PortfolioID   string
	OwnerClientID string
	PositionID    string
	Delta         float64 // >0 add from cash; <0 return to cash
}

// SetMarginModeInput changes account-wide margin mode.
type SetMarginModeInput struct {
	ClientID    string
	PortfolioID string
	Mode        string // isolated | cross
}

// MarginRepayInput pays debt without closing (interest first, then principal).
// Amount is in debt units: quote cash for long, base coins for short.
type MarginRepayInput struct {
	ClientID      string
	PortfolioID   string
	OwnerClientID string
	PositionID    string
	Amount        float64
}

// MarginCloseInput closes all or part of a margin position at market.
type MarginCloseInput struct {
	ClientID       string
	PortfolioID    string
	OwnerClientID  string
	PositionID     string
	Quantity       float64 // 0 = full close
	IdempotencyKey string
}

// MarginBracketsInput sets or clears stop-loss / take-profit on an open position.
// Omit pointer (nil) to leave unchanged; use ClearStopLoss / ClearTakeProfit to remove.
type MarginBracketsInput struct {
	ClientID        string
	PortfolioID     string
	OwnerClientID   string
	PositionID      string
	StopLoss        *float64
	TakeProfit      *float64
	ClearStopLoss   bool
	ClearTakeProfit bool
}

// availableCashForTrading returns unused wallet cash (spot + margin reservations).
// Unrealized margin PnL is not spendable — it only counts toward cross equity / liq.
func (s *Service) availableCashForTrading(ctx context.Context, clientID string, cashBalance float64) (float64, error) {
	return s.availableCashForTradingMode(ctx, clientID, cashBalance, "")
}

func (s *Service) availableCashForTradingMode(ctx context.Context, clientID string, cashBalance float64, _ domain.MarginMode) (float64, error) {
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

// marginReserveNeed is initial margin plus worst-case open fee (slipped limit * fee).
func (s *Service) marginReserveNeed(ex domain.Exchange, side domain.MarginSide, qty, limitPrice float64, lev int) (float64, error) {
	cost := s.paperCost(ex)
	worst := domain.ApplySlippage(limitPrice, domain.MarginOpenTradeSide(side), cost.SlippageRate)
	im, err := domain.InitialMargin(qty, worst, lev)
	if err != nil {
		return 0, err
	}
	return im + domain.FeeAmount(qty, worst, cost.FeeRate), nil
}

func (s *Service) sumOpenMarginUnrealized(ctx context.Context, clientID string) (float64, error) {
	list, err := s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return 0, err
	}
	var sum float64
	for i := range list {
		s.markMarginPosition(ctx, &list[i])
		sum += list[i].UnrealizedPnL
	}
	return sum, nil
}

// SetMarginMode sets isolated|cross. Fails if any open margin position or pending margin order.
func (s *Service) SetMarginMode(ctx context.Context, in SetMarginModeInput) (*domain.Portfolio, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireBook(ctx, in.ClientID, in.PortfolioID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	mode, err := domain.NormalizeMarginMode(in.Mode)
	if err != nil {
		return nil, err
	}
	if p.MarginMode == mode {
		return p, nil
	}
	nPos, err := s.store.CountOpenMarginPositions(ctx, clientID)
	if err != nil {
		return nil, err
	}
	nOrd, err := s.store.CountOpenMarginOrders(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if nPos > 0 || nOrd > 0 {
		return nil, fmt.Errorf("%w: cannot change margin mode while open positions or pending margin orders exist", domain.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	if err := s.store.UpdatePortfolioMarginMode(ctx, clientID, mode, now); err != nil {
		return nil, err
	}
	return s.store.GetPortfolio(ctx, clientID)
}

// AdjustMargin adds or removes isolated margin and recalculates liquidation price.
func (s *Service) AdjustMargin(ctx context.Context, in MarginAdjustInput) (*domain.MarginPosition, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	if in.Delta == 0 || math.IsNaN(in.Delta) || math.IsInf(in.Delta, 0) {
		return nil, fmt.Errorf("%w: delta must be a non-zero number", domain.ErrInvalidArgument)
	}
	if p.MarginMode != domain.MarginModeIsolated {
		return nil, fmt.Errorf("%w: add/remove margin is only allowed in isolated mode", domain.ErrInvalidArgument)
	}
	for attempt := 0; attempt < maxDebtMutationRetries; attempt++ {
		book, err := s.store.GetPortfolio(ctx, clientID)
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
		if pos.Mode != domain.MarginModeIsolated {
			return nil, fmt.Errorf("%w: position is not isolated", domain.ErrInvalidArgument)
		}
		now := time.Now().UTC()
		newMargin := pos.Margin + in.Delta
		minM, err := domain.MinIsolatedMargin(pos.Quantity, pos.EntryPrice, pos.Leverage)
		if err != nil {
			return nil, err
		}
		if newMargin+1e-9 < minM {
			return nil, fmt.Errorf("%w: margin cannot go below initial margin %g", domain.ErrInvalidArgument, minM)
		}
		if in.Delta > 0 {
			avail, err := s.availableCashForTradingMode(ctx, clientID, book.CashBalance, domain.MarginModeIsolated)
			if err != nil {
				return nil, err
			}
			if avail+1e-9 < in.Delta {
				return nil, fmt.Errorf("%w: insufficient available cash to add margin", domain.ErrInvalidArgument)
			}
			book.CashBalance -= in.Delta
		} else {
			book.CashBalance += -in.Delta
		}
		book.UpdatedAt = now
		_ = s.accruePositionInterest(ctx, pos, now)
		liq, err := s.liqFromStoredEntry(pos, pos.Quantity, newMargin, pos.DebtPrincipal, pos.DebtInterest)
		if err != nil {
			return nil, err
		}
		pos.Margin = newMargin
		pos.LiquidationPrice = liq
		pos.UpdatedAt = now
		action := domain.MarginActionAddMargin
		if in.Delta < 0 {
			action = domain.MarginActionRemoveMargin
		}
		tr := domain.MarginTrade{
			ID: uuid.NewString(), ClientID: clientID, PositionID: pos.ID, Exchange: pos.Exchange, Symbol: pos.Symbol,
			Side: pos.Side, Action: action, Quantity: 0, Price: pos.EntryPrice, Notional: 0,
			MarginDelta: -in.Delta,
			Leverage:    pos.Leverage, CreatedAt: now,
		}
		err = s.store.ApplyMarginAdjust(ctx, book, *pos, tr)
		if errors.Is(err, domain.ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return s.GetMarginPosition(ctx, clientID, pos.ID)
	}
	return nil, fmt.Errorf("%w: margin adjust lost the debt race", domain.ErrConflict)
}

const maxDebtMutationRetries = 10

// RepayMarginDebt pays interest first, then principal, without closing the position.
// Concurrent interest accrual is serialized via debt-snapshot CAS + retries.
func (s *Service) RepayMarginDebt(ctx context.Context, in MarginRepayInput) (*domain.MarginPosition, *domain.MarginTrade, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	book, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, nil, err
	}
	clientID := book.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	if in.Amount <= 0 || math.IsNaN(in.Amount) || math.IsInf(in.Amount, 0) {
		return nil, nil, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidArgument)
	}
	for attempt := 0; attempt < maxDebtMutationRetries; attempt++ {
		p, err := s.store.GetPortfolio(ctx, clientID)
		if err != nil {
			return nil, nil, err
		}
		pos, err := s.store.GetMarginPosition(ctx, clientID, strings.TrimSpace(in.PositionID))
		if err != nil {
			return nil, nil, err
		}
		if pos.Status != domain.MarginPositionOpen {
			return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
		}
		now := time.Now().UTC()
		// Accrue only (never liquidate here — liquidation is a separate exclusive close).
		_ = s.accrueInterestOnly(ctx, pos, now)
		pos, err = s.store.GetMarginPosition(ctx, clientID, pos.ID)
		if err != nil {
			return nil, nil, err
		}
		if pos.Status != domain.MarginPositionOpen {
			return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
		}
		expected := domain.DebtSnapshotFromPos(pos)
		totalDebt := pos.DebtPrincipal + pos.DebtInterest
		if totalDebt <= domain.PositionEpsilon {
			return nil, nil, fmt.Errorf("%w: no outstanding debt", domain.ErrInvalidArgument)
		}
		pay := in.Amount
		if pay > totalDebt {
			pay = totalDebt
		}
		cashNeed := pay
		if pos.DebtAsset == domain.DebtAssetBase {
			mark, merr := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
			if merr != nil || mark <= 0 {
				return nil, nil, fmt.Errorf("%w: market price unavailable for coin repay", domain.ErrUpstream)
			}
			cashNeed = pay * mark
		}
		avail, err := s.availableCashForTradingMode(ctx, clientID, p.CashBalance, p.MarginMode)
		if err != nil {
			return nil, nil, err
		}
		if avail+1e-9 < cashNeed {
			return nil, nil, fmt.Errorf("%w: insufficient available cash to repay (need %g, available %g)", domain.ErrInvalidArgument, cashNeed, avail)
		}
		pp, ip, np, ni := domain.AllocateRepayment(pos.DebtPrincipal, pos.DebtInterest, pay)
		pos.DebtPrincipal, pos.DebtInterest = np, ni
		liq, err := s.liqFromStoredEntry(pos, pos.Quantity, pos.Margin, pos.DebtPrincipal, pos.DebtInterest)
		if err != nil {
			return nil, nil, err
		}
		pos.LiquidationPrice = liq
		pos.UpdatedAt = now
		// Keep last_interest_at: repayment does not rewind the accrual cursor.
		p.CashBalance -= cashNeed
		p.UpdatedAt = now
		tr := domain.MarginTrade{
			ID: uuid.NewString(), ClientID: clientID, PositionID: pos.ID, Exchange: pos.Exchange, Symbol: pos.Symbol,
			Side: pos.Side, Action: domain.MarginActionRepay, Quantity: pay, Price: cashNeed / pay, Notional: cashNeed,
			MarginDelta: -cashNeed, PrincipalPaid: pp, InterestPaid: ip, Leverage: pos.Leverage, CreatedAt: now,
		}
		err = s.store.ApplyMarginRepay(ctx, p, *pos, tr, expected)
		if err == nil {
			_ = s.recomputeCrossLiquidations(ctx, clientID, now)
			out, gerr := s.GetMarginPosition(ctx, clientID, pos.ID)
			return out, &tr, gerr
		}
		// Concurrent liquidation closed the position — repay cannot apply.
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
		}
		if !errors.Is(err, domain.ErrConflict) {
			return nil, nil, err
		}
		// Interest or close raced — retry with fresh debt snapshot.
	}
	return nil, nil, fmt.Errorf("%w: concurrent debt update, try again", domain.ErrConflict)
}

func (s *Service) liqPriceFor(side domain.MarginSide, entry, qty, margin, debtP, debtI float64) (float64, error) {
	return domain.LiquidationPriceWithDebt(side, entry, qty, margin, debtP, debtI, domain.DefaultMaintenanceMarginRate)
}

// liqFromStoredEntry computes liq from a persisted position. EntryPrice may bake
// the open fee; invert it so liq matches the raw fill used at open / on the UI.
func (s *Service) liqFromStoredEntry(pos *domain.MarginPosition, qty, margin, debtP, debtI float64) (float64, error) {
	if pos == nil {
		return 0, fmt.Errorf("%w: position is required", domain.ErrInvalidArgument)
	}
	cost := s.paperCost(pos.Exchange)
	raw := domain.RawFillFromEffectiveEntry(pos.Side, pos.EntryPrice, cost.FeeRate)
	return s.liqPriceFor(pos.Side, raw, qty, margin, debtP, debtI)
}

// totalDebtNotionalQuote sums all open position debt in quote currency.
func (s *Service) totalDebtNotionalQuote(ctx context.Context, clientID string) (float64, error) {
	list, err := s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return 0, err
	}
	var sum float64
	for i := range list {
		mark := list[i].EntryPrice
		if m, e := s.lastPrice(ctx, string(list[i].Exchange), list[i].Symbol); e == nil && m > 0 {
			mark = m
		}
		sum += domain.DebtNotionalQuote(list[i].Side, list[i].DebtPrincipal, list[i].DebtInterest, mark)
	}
	return sum, nil
}

func (s *Service) checkBorrowLimit(ctx context.Context, p *domain.Portfolio, addNotional float64) error {
	used, err := s.totalDebtNotionalQuote(ctx, p.BookID())
	if err != nil {
		return err
	}
	maxB := domain.MaxBorrowNotional(p.StartingBalance)
	if used+addNotional > maxB+1e-6 {
		return fmt.Errorf("%w: borrow limit exceeded (used %g + new %g > max %g)", domain.ErrInvalidArgument, used, addNotional, maxB)
	}
	return nil
}

// ProcessMarginInterest is the durable background interest job.
// For each open debt: O(1) catch-up from last_interest_at → now, CAS update so two
// workers cannot double-apply the same window, then recompute liq and liquidate if breached.
// Returns (positions that received interest, positions liquidated after interest).
func (s *Service) ProcessMarginInterest(ctx context.Context, now time.Time) (accrued, liquidated int, err error) {
	if s.store == nil {
		return 0, 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	list, err := s.store.ListOpenMarginPositionsWithDebt(ctx)
	if err != nil {
		return 0, 0, err
	}
	for i := range list {
		if s.ownerClosed(ctx, list[i].ClientID) {
			continue
		}
		did, liq, e := s.accrueInterestAndMaybeLiquidate(ctx, &list[i], now)
		if e != nil {
			continue
		}
		if did {
			accrued++
		}
		if liq {
			liquidated++
		}
	}
	return accrued, liquidated, nil
}

// accrueInterestOnly applies catch-up via CAS without liquidating.
// Used from repay/close/get so a concurrent user action cannot nest a second close.
// Safe under concurrent workers; no-op if principal is zero or clock is not ahead of last_interest_at.
func (s *Service) accrueInterestOnly(ctx context.Context, pos *domain.MarginPosition, now time.Time) error {
	_, err := s.accrueInterestCAS(ctx, pos, now)
	return err
}

// accruePositionInterest is the safe catch-up used by read/repay/close paths (no liquidate).
func (s *Service) accruePositionInterest(ctx context.Context, pos *domain.MarginPosition, now time.Time) error {
	return s.accrueInterestOnly(ctx, pos, now)
}

// accrueInterestCAS applies interest with full debt snapshot CAS and updates pos in memory on success.
// Does not liquidate. Returns whether interest (or cursor seed) was applied.
func (s *Service) accrueInterestCAS(ctx context.Context, pos *domain.MarginPosition, now time.Time) (applied bool, err error) {
	if pos == nil || pos.Status != domain.MarginPositionOpen {
		return false, nil
	}
	// Fully paid principal: interest stops (no accrual, no CAS).
	if pos.DebtPrincipal <= domain.PositionEpsilon {
		return false, nil
	}
	now = now.UTC()
	snap := domain.DebtSnapshotFromPos(pos)
	if snap.LastInterestAt.IsZero() {
		// Seed cursor without inventing past interest; full debt CAS (principal/interest must still match).
		seed := now.Truncate(time.Hour)
		if !pos.OpenedAt.IsZero() {
			seed = pos.OpenedAt.UTC()
		}
		liq, _ := s.liqFromStoredEntry(pos, pos.Quantity, pos.Margin, pos.DebtPrincipal, pos.DebtInterest)
		claimed, cerr := s.store.AccrueInterestCAS(ctx, pos.ID, snap, pos.DebtInterest, seed, liq, now)
		if cerr != nil {
			return false, cerr
		}
		if claimed {
			pos.LastInterestAt = seed
		}
		return false, nil
	}
	// Clock backward or no full hour: do not remove interest or reprocess.
	ni, nl, hours := domain.AccrueInterestHours(pos.DebtPrincipal, pos.DebtInterest, snap.LastInterestAt, now, domain.DefaultMarginHourlyInterestRate)
	if hours <= 0 {
		return false, nil
	}
	liq, lerr := s.liqFromStoredEntry(pos, pos.Quantity, pos.Margin, pos.DebtPrincipal, ni)
	if lerr != nil {
		return false, lerr
	}
	// Full debt CAS: repay/close that changed principal/interest causes claimed=false.
	claimed, cerr := s.store.AccrueInterestCAS(ctx, pos.ID, snap, ni, nl, liq, now)
	if cerr != nil {
		return false, cerr
	}
	if !claimed {
		// Another worker/repay/close changed debt or already advanced this period.
		return false, nil
	}
	pos.DebtInterest = ni
	pos.LastInterestAt = nl
	pos.LiquidationPrice = liq
	pos.UpdatedAt = now
	return true, nil
}

// accrueInterestAndMaybeLiquidate accrues interest with CAS, then may liquidate once if mark
// is past the new liq. Liquidation is idempotent under concurrent user close/repay.
// Returns (interestApplied, wasLiquidated, error).
func (s *Service) accrueInterestAndMaybeLiquidate(ctx context.Context, pos *domain.MarginPosition, now time.Time) (applied, liquidated bool, err error) {
	applied, err = s.accrueInterestCAS(ctx, pos, now)
	if err != nil || !applied {
		return applied, false, err
	}
	// Same operation: re-read after interest, re-check liquidate (user may have repaid/closed).
	if s.market == nil {
		return true, false, nil
	}
	cur, rerr := s.store.GetMarginPosition(ctx, pos.ClientID, pos.ID)
	if rerr != nil {
		if errors.Is(rerr, domain.ErrNotFound) {
			return true, false, nil
		}
		return true, false, rerr
	}
	if cur.Mode == domain.MarginModeCross {
		n, lerr := s.liquidateCrossIfUnderMaint(ctx, cur.ClientID, now)
		return true, n > 0, lerr
	}
	did, lerr := s.tryLiquidateIfBreached(ctx, pos.ClientID, pos.ID, now)
	return true, did, lerr
}

// tryLiquidateIfBreached re-reads the position and closes it once if still open and underwater.
// Concurrent user close/repay is safe: already-closed → no-op; debt/qty CAS + retries on conflict.
// Returns true only when this call successfully applied a liquidation close trade.
func (s *Service) tryLiquidateIfBreached(ctx context.Context, clientID, positionID string, now time.Time) (bool, error) {
	unlock := s.lockClient(clientID)
	defer unlock()
	cur, err := s.store.GetMarginPosition(ctx, clientID, positionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if cur.Status != domain.MarginPositionOpen {
		return false, nil
	}
	// Cross health is account-level. Isolated ShouldLiquidate here would close a
	// healthy shared book (and, after interest, even a mark above entry).
	if cur.Mode == domain.MarginModeCross {
		return false, nil
	}
	mark, merr := s.lastPrice(ctx, string(cur.Exchange), cur.Symbol)
	if merr != nil || mark <= 0 {
		return false, nil
	}
	// Recompute liq from current debt (may have been partially repaid after interest).
	// Use the raw fill, not fee-baked EntryPrice, so liq matches the open/displayed price.
	liq := cur.LiquidationPrice
	if fresh, lerr := s.liqFromStoredEntry(cur, cur.Quantity, cur.Margin, cur.DebtPrincipal, cur.DebtInterest); lerr == nil {
		liq = fresh
	}
	if !domain.ShouldLiquidate(cur.Side, mark, liq) {
		return false, nil
	}
	exit := liq
	if cur.Side == domain.MarginLong && mark < exit {
		exit = mark
	}
	if cur.Side == domain.MarginShort && mark > exit {
		exit = mark
	}
	_, _, cerr := s.closeMarginAt(ctx, cur, cur.Quantity, exit, domain.MarginCloseLiquidation)
	if cerr == nil {
		return true, nil
	}
	// Already closed by concurrent user close (or another liquidator): not an error.
	if errors.Is(cerr, domain.ErrNotFound) || isPositionNotOpenErr(cerr) {
		return false, nil
	}
	// Conflict exhausted or other — surface only unexpected failures.
	if errors.Is(cerr, domain.ErrConflict) {
		return false, nil
	}
	return false, cerr
}

func isPositionNotOpenErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrInvalidArgument) && strings.Contains(err.Error(), "position is not open") {
		return true
	}
	if errors.Is(err, domain.ErrNotFound) {
		return true
	}
	return false
}

// PlaceMarginOrder opens a market position immediately or rests a limit open order.
func (s *Service) PlaceMarginOrder(ctx context.Context, in MarginOrderInput) (*domain.MarginPosition, *domain.MarginOrder, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
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
	idempHash := hashParts("margin_open", string(typ), string(ex), sym, string(side), in.Quantity, in.Leverage, in.LimitPrice, in.StopLoss, in.TakeProfit)
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, err
	} else if rec != nil {
		return s.replayMarginOpen(ctx, rec)
	}
	if err := domain.RequireQuoteMatchesCurrency(ex, sym, p.Currency); err != nil {
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

	mode := p.MarginMode
	if mode == "" {
		mode = domain.MarginModeIsolated
	}

	if typ == domain.MarginOrderLimit {
		if in.LimitPrice < domain.MinTriggerPrice || in.LimitPrice > domain.MaxTriggerPrice ||
			math.IsNaN(in.LimitPrice) || math.IsInf(in.LimitPrice, 0) {
			return nil, nil, fmt.Errorf("%w: limitPrice out of range", domain.ErrInvalidArgument)
		}
		base, _ := domain.SplitBaseQuote(ex, sym)
		if err := s.guardNewRisk(ctx, clientID, base, in.Quantity*in.LimitPrice); err != nil {
			return nil, nil, err
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
		need, err := s.marginReserveNeed(ex, side, in.Quantity, in.LimitPrice, in.Leverage)
		if err != nil {
			return nil, nil, err
		}
		avail, err := s.availableCashForTradingMode(ctx, clientID, p.CashBalance, mode)
		if err != nil {
			return nil, nil, err
		}
		if avail+1e-9 < need {
			return nil, nil, fmt.Errorf("%w: insufficient available cash for margin (need %g, available %g)", domain.ErrInvalidArgument, need, avail)
		}
		// Reserve required margin; released on cancel/reject/fill (fill then debits cash).
		o := domain.MarginOrder{
			ID: uuid.NewString(), ClientID: clientID, Exchange: ex, Symbol: sym,
			Side: side, Type: domain.MarginOrderLimit, Quantity: in.Quantity, Leverage: in.Leverage,
			LimitPrice: in.LimitPrice, ReservedMargin: need, StopLoss: in.StopLoss, TakeProfit: in.TakeProfit,
			Status: domain.MarginOrderOpen, CreatedAt: now, UpdatedAt: now,
		}
		ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindMarginOpen, idempIDs{OrderID: o.ID})
		out, err := s.store.CreateMarginOrder(ctx, o)
		if err != nil && isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayMarginOpen(ctx, rec)
			}
		}
		return nil, out, err
	}

	// Market open
	last, err := s.lastPrice(ctx, string(ex), sym)
	if err != nil {
		return nil, nil, err
	}
	cost := s.paperCost(ex)
	price := domain.ApplySlippage(last, domain.MarginOpenTradeSide(side), cost.SlippageRate)
	if price <= 0 {
		return nil, nil, fmt.Errorf("%w: invalid fill price after slippage", domain.ErrInvalidArgument)
	}
	openFee := domain.FeeAmount(in.Quantity, price, cost.FeeRate)
	entry := domain.EffectiveMarginEntry(side, price, cost.FeeRate)
	base, _ := domain.SplitBaseQuote(ex, sym)
	if err := s.guardNewRisk(ctx, clientID, base, in.Quantity*price); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateMarginBrackets(side, price, in.StopLoss, in.TakeProfit); err != nil {
		return nil, nil, err
	}
	margin, err := domain.InitialMargin(in.Quantity, price, in.Leverage)
	if err != nil {
		return nil, nil, err
	}
	needCash := margin + openFee
	avail, err := s.availableCashForTradingMode(ctx, clientID, p.CashBalance, mode)
	if err != nil {
		return nil, nil, err
	}
	if avail+1e-9 < needCash || p.CashBalance+1e-9 < needCash {
		return nil, nil, fmt.Errorf("%w: insufficient available cash for margin (need %g, available %g)", domain.ErrInvalidArgument, needCash, avail)
	}
	borrowP, debtAsset, err := domain.BorrowedPrincipalOnOpen(side, in.Quantity, price, in.Leverage)
	if err != nil {
		return nil, nil, err
	}
	borrowNotional := domain.DebtNotionalQuote(side, borrowP, 0, price)
	if err := s.checkBorrowLimit(ctx, p, borrowNotional); err != nil {
		return nil, nil, err
	}
	liq, err := s.liqPriceFor(side, price, in.Quantity, margin, borrowP, 0)
	if err != nil {
		return nil, nil, err
	}
	lastInt := now.Truncate(time.Hour)
	pos := domain.MarginPosition{
		ID: uuid.NewString(), ClientID: clientID, Exchange: ex, Symbol: sym, Side: side, Mode: mode,
		Quantity: in.Quantity, EntryPrice: entry, Leverage: in.Leverage, Margin: margin,
		DebtPrincipal: borrowP, DebtInterest: 0, DebtAsset: debtAsset, LastInterestAt: lastInt,
		LiquidationPrice: liq, StopLoss: in.StopLoss, TakeProfit: in.TakeProfit,
		Status: domain.MarginPositionOpen, OpenedAt: now, UpdatedAt: now,
	}
	tr := domain.MarginTrade{
		ID: uuid.NewString(), ClientID: clientID, PositionID: pos.ID, Exchange: ex, Symbol: sym,
		Side: side, Action: "open", Quantity: in.Quantity, Price: price, Notional: in.Quantity * price,
		MarginDelta: -needCash, Leverage: in.Leverage, Fee: openFee, CreatedAt: now,
	}
	p.CashBalance -= needCash
	p.UpdatedAt = now
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindMarginOpen, idempIDs{PositionID: pos.ID, TradeID: tr.ID})
	if err := s.store.ApplyMarginOpen(ctx, p, pos, tr); err != nil {
		if isIdempotencyHit(err) {
			if rec, rerr := s.replayAfterHit(ctx, clientID, idempKey, idempHash); rerr == nil && rec != nil {
				return s.replayMarginOpen(ctx, rec)
			}
		}
		return nil, nil, err
	}
	_ = s.recomputeCrossLiquidations(ctx, clientID, now)
	out, err := s.store.GetMarginPosition(ctx, clientID, pos.ID)
	return out, nil, err
}

// ListMarginPositions lists open margin positions with marks.
func (s *Service) ListMarginPositions(ctx context.Context, clientID string, portfolioID ...string) ([]domain.MarginPosition, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
	list, err := s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for i := range list {
		_ = s.accruePositionInterest(ctx, &list[i], now)
		s.markMarginPosition(ctx, &list[i])
	}
	// Re-list after accrual so debt fields are fresh
	list, err = s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		s.markMarginPosition(ctx, &list[i])
	}
	return list, nil
}

// GetMarginPosition returns one open or closed position with mark if open.
func (s *Service) GetMarginPosition(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.MarginPosition, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: position id is required", domain.ErrInvalidArgument)
	}
	pos, err := s.store.GetMarginPosition(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	if pos.Status == domain.MarginPositionOpen {
		_ = s.accruePositionInterest(ctx, pos, time.Now().UTC())
		pos, err = s.store.GetMarginPosition(ctx, clientID, id)
		if err != nil {
			return nil, err
		}
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
		// Fail closed: do not pretend entry is a live mark (that skips liquidation).
		pos.MarkPrice = 0
		pos.UnrealizedPnL = 0
		pos.DebtNotional = domain.DebtNotionalQuote(pos.Side, pos.DebtPrincipal, pos.DebtInterest, pos.EntryPrice)
		return
	}
	pos.MarkPrice = mark
	// EntryPrice may bake open fee (EffectiveMarginEntry). Equity must not
	// double-count that fee (already debited from cash on open).
	cost := s.paperCost(pos.Exchange)
	rawEntry := domain.RawFillFromEffectiveEntry(pos.Side, pos.EntryPrice, cost.FeeRate)
	pos.UnrealizedPnL = domain.MarginUnrealizedPnL(pos.Side, pos.Quantity, rawEntry, mark)
	pos.DebtNotional = domain.DebtNotionalQuote(pos.Side, pos.DebtPrincipal, pos.DebtInterest, mark)
}

// CloseMarginPosition closes full or partial size at market.
func (s *Service) CloseMarginPosition(ctx context.Context, in MarginCloseInput) (*domain.MarginPosition, *domain.MarginTrade, error) {
	if s.store == nil || s.market == nil {
		return nil, nil, fmt.Errorf("%w: portfolio service not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
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
	id := strings.TrimSpace(in.PositionID)
	if id == "" {
		return nil, nil, fmt.Errorf("%w: position id is required", domain.ErrInvalidArgument)
	}
	idempHash := hashParts("margin_close", id, in.Quantity)
	if rec, err := s.checkIdempotency(ctx, clientID, idempKey, idempHash); err != nil {
		return nil, nil, err
	} else if rec != nil {
		return s.replayMarginClose(ctx, rec)
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
	ctx = s.withIdempotency(ctx, clientID, idempKey, idempHash, domain.IdempotencyKindMarginClose, idempIDs{PositionID: id})
	return s.closeMarginAt(ctx, pos, closeQty, price, reason)
}

// SetMarginBrackets updates SL/TP on an open position.
func (s *Service) SetMarginBrackets(ctx context.Context, in MarginBracketsInput) (*domain.MarginPosition, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
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
	if err := s.store.UpdateMarginPositionMeta(ctx, *pos); err != nil {
		return nil, err
	}
	return s.GetMarginPosition(ctx, clientID, pos.ID)
}

// ListMarginOrders lists margin open orders.
func (s *Service) ListMarginOrders(ctx context.Context, clientID, status string, limit, offset int, portfolioID ...string) ([]domain.MarginOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
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
func (s *Service) CancelMarginOrder(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.MarginOrder, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleTrader, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
	unlock := s.lockClient(clientID)
	defer unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	return s.store.CancelMarginOrder(ctx, clientID, id, time.Now().UTC(), domain.CancelReasonUser)
}

// ListMarginTrades returns margin trade history.
func (s *Service) ListMarginTrades(ctx context.Context, clientID string, limit, offset int, portfolioID ...string) ([]domain.MarginTrade, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
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
	// Interest is owned by MarginInterestWorker (durable catch-up + CAS). This path
	// only fills limits, SL/TP, and price-based liquidation.

	// 1) Limit opens
	orders, err := s.store.ListAllOpenMarginOrders(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	for i := range orders {
		if s.ownerClosed(ctx, orders[i].ClientID) {
			continue
		}
		if s.tryFillMarginLimit(ctx, &orders[i], now) {
			filled++
		}
	}
	// 2) Refresh cross liquidation prices, then liquidation + SL/TP
	positions, err := s.store.ListAllOpenMarginPositions(ctx)
	if err != nil {
		return filled, 0, 0, err
	}
	// Group by client for cross recompute
	byClient := map[string][]domain.MarginPosition{}
	for i := range positions {
		byClient[positions[i].ClientID] = append(byClient[positions[i].ClientID], positions[i])
	}
	for clientID := range byClient {
		_ = s.recomputeCrossLiquidations(ctx, clientID, now)
	}
	positions, err = s.store.ListAllOpenMarginPositions(ctx)
	if err != nil {
		return filled, 0, 0, err
	}
	for i := range positions {
		pos := positions[i]
		if s.ownerClosed(ctx, pos.ClientID) {
			continue
		}
		mark, merr := s.lastPrice(ctx, string(pos.Exchange), pos.Symbol)
		if merr != nil || mark <= 0 {
			continue
		}
		// Isolated liquidation first (idempotent if user already closed).
		// Cross: do not batch-close on stale per-position liq prices — account equity and
		// remaining positions change after each close; sequential path below re-evaluates.
		isCross := pos.Mode == domain.MarginModeCross
		if !isCross && domain.ShouldLiquidate(pos.Side, mark, pos.LiquidationPrice) {
			if did, _ := s.tryLiquidateIfBreached(ctx, pos.ClientID, pos.ID, now); did {
				liquidated++
			}
			continue
		}
		// Stop loss
		if domain.ShouldTriggerStopLoss(pos.Side, mark, pos.StopLoss) {
			unlock := s.lockClient(pos.ClientID)
			_, _, e := s.closeMarginAt(ctx, &pos, pos.Quantity, mark, domain.MarginCloseStopLoss)
			unlock()
			if e == nil {
				stopped++
			}
			continue
		}
		// Take profit
		if domain.ShouldTriggerTakeProfit(pos.Side, mark, pos.TakeProfit) {
			unlock := s.lockClient(pos.ClientID)
			_, _, e := s.closeMarginAt(ctx, &pos, pos.Quantity, mark, domain.MarginCloseTakeProfit)
			unlock()
			if e == nil {
				stopped++
			}
		}
	}
	// Cross account-level: if equity < total maint, liquidate worst-first with re-eval after each.
	for clientID := range byClient {
		if s.ownerClosed(ctx, clientID) {
			continue
		}
		n, _ := s.liquidateCrossIfUnderMaint(ctx, clientID, now)
		liquidated += n
	}
	return filled, liquidated, stopped, nil
}

// recomputeCrossLiquidations updates liquidation prices for cross-mode positions of a client.
func (s *Service) recomputeCrossLiquidations(ctx context.Context, clientID string, now time.Time) error {
	p, err := s.store.GetPortfolio(ctx, clientID)
	if err != nil {
		return err
	}
	if p.MarginMode != domain.MarginModeCross {
		return nil
	}
	list, err := s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}
	var totalMaint, sumMargin, sumU float64
	for i := range list {
		s.markMarginPosition(ctx, &list[i])
		totalMaint += domain.MaintenanceMargin(list[i].Quantity, list[i].EntryPrice, domain.DefaultMaintenanceMarginRate)
		sumMargin += list[i].Margin
		sumU += list[i].UnrealizedPnL
	}
	// equity = cash + sum(margin) + sum(upnl); cash excludes locked margins
	for i := range list {
		// equity excluding this position's U: cash + sumMargin + (sumU - U_i)
		equityExcl := p.CashBalance + sumMargin + (sumU - list[i].UnrealizedPnL)
		liq, err := domain.CrossLiquidationPrice(list[i].Side, list[i].EntryPrice, list[i].Quantity, equityExcl, totalMaint)
		if err != nil {
			continue
		}
		list[i].LiquidationPrice = liq
		list[i].UpdatedAt = now
		_ = s.store.UpdateMarginPositionMeta(ctx, list[i])
	}
	return nil
}

// crossAccountRisk is a fresh snapshot of cross-margin equity vs maintenance.
type crossAccountRisk struct {
	portfolio  *domain.Portfolio
	positions  []domain.MarginPosition
	equity     float64
	totalMaint float64
}

// loadCrossAccountRisk re-reads portfolio and open positions with current marks.
// equity = cash + sum(margin) + sum(unrealizedPnL); totalMaint = sum of per-position maintenance.
func (s *Service) loadCrossAccountRisk(ctx context.Context, clientID string) (*crossAccountRisk, error) {
	p, err := s.store.GetPortfolio(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if p.MarginMode != domain.MarginModeCross {
		return &crossAccountRisk{portfolio: p}, nil
	}
	list, err := s.store.ListOpenMarginPositions(ctx, clientID)
	if err != nil {
		return nil, err
	}
	var totalMaint, sumMargin, sumU float64
	for i := range list {
		s.markMarginPosition(ctx, &list[i])
		totalMaint += domain.MaintenanceMargin(list[i].Quantity, list[i].EntryPrice, domain.DefaultMaintenanceMarginRate)
		sumMargin += list[i].Margin
		sumU += list[i].UnrealizedPnL
	}
	return &crossAccountRisk{
		portfolio:  p,
		positions:  list,
		equity:     p.CashBalance + sumMargin + sumU,
		totalMaint: totalMaint,
	}, nil
}

// pickWorstCrossPosition chooses the next position to liquidate under shared equity:
// most negative unrealized PnL, then larger notional, then stable id.
func pickWorstCrossPosition(list []domain.MarginPosition) *domain.MarginPosition {
	if len(list) == 0 {
		return nil
	}
	worst := &list[0]
	for i := 1; i < len(list); i++ {
		p := &list[i]
		if p.UnrealizedPnL < worst.UnrealizedPnL-1e-12 {
			worst = p
			continue
		}
		if math.Abs(p.UnrealizedPnL-worst.UnrealizedPnL) <= 1e-12 {
			notionalP := p.Quantity * p.EntryPrice
			notionalW := worst.Quantity * worst.EntryPrice
			if notionalP > notionalW+1e-12 || (math.Abs(notionalP-notionalW) <= 1e-12 && p.ID < worst.ID) {
				worst = p
			}
		}
	}
	return worst
}

// liquidateCrossIfUnderMaint restores cross-margin health while equity is below total
// maintenance. It liquidates only the worst position, and only the minimum quantity needed
// (partial close when the account is slightly under). After each successful close it reloads
// cash, remaining sizes, and marks — never batch-closes on a stale snapshot.
func (s *Service) liquidateCrossIfUnderMaint(ctx context.Context, clientID string, now time.Time) (int, error) {
	unlock := s.lockClient(clientID)
	defer unlock()
	n := 0
	// Bound rounds: partials may take more than one step per position.
	maxRounds := domain.MaxOpenMarginPositions*3 + 5
	for round := 0; round < maxRounds; round++ {
		risk, err := s.loadCrossAccountRisk(ctx, clientID)
		if err != nil {
			return n, err
		}
		if risk.portfolio == nil || risk.portfolio.MarginMode != domain.MarginModeCross {
			return n, nil
		}
		if len(risk.positions) == 0 {
			return n, nil
		}
		// Healthy: stop — remaining quantity stays open with the updated shared balance.
		if risk.equity+1e-9 >= risk.totalMaint {
			if n > 0 {
				_ = s.recomputeCrossLiquidations(ctx, clientID, now)
			}
			return n, nil
		}
		worst := pickWorstCrossPosition(risk.positions)
		if worst == nil {
			return n, nil
		}
		mark := worst.MarkPrice
		if mark <= 0 {
			// No live mark — do not liquidate (or invent cash) at entry.
			return n, nil
		}
		cq := domain.CrossPartialLiquidationQty(
			worst.Side, worst.Quantity, worst.EntryPrice, mark,
			worst.DebtPrincipal, worst.DebtInterest, worst.DebtAsset,
			risk.equity, risk.totalMaint, domain.DefaultMaintenanceMarginRate,
		)
		if cq <= domain.PositionEpsilon {
			// Cannot improve health from this book (or already healthy race) — stop.
			return n, nil
		}
		_, _, e := s.closeMarginAt(ctx, worst, cq, mark, domain.MarginCloseLiquidation)
		if e == nil {
			n++
			// Refresh cross liq prices for survivors before next equity check.
			_ = s.recomputeCrossLiquidations(ctx, clientID, now)
			continue
		}
		// Concurrent closer finished this id — re-read account and continue if still under.
		if isPositionNotOpenErr(e) || errors.Is(e, domain.ErrNotFound) {
			continue
		}
		// Unexpected failure — stop to avoid a tight retry loop.
		return n, e
	}
	return n, nil
}

func (s *Service) tryFillMarginLimit(ctx context.Context, o *domain.MarginOrder, now time.Time) bool {
	if o == nil || o.Status != domain.MarginOrderOpen {
		return false
	}
	unlock := s.lockClient(o.ClientID)
	defer unlock()
	last, err := s.lastPrice(ctx, string(o.Exchange), o.Symbol)
	if err != nil || last <= 0 {
		return false
	}
	if !domain.MarginLimitTriggered(o.Side, o.LimitPrice, last) {
		return false
	}
	cost := s.paperCost(o.Exchange)
	price := domain.ApplySlippage(last, domain.MarginOpenTradeSide(o.Side), cost.SlippageRate)
	if price <= 0 {
		return false
	}
	openFee := domain.FeeAmount(o.Quantity, price, cost.FeeRate)
	entry := domain.EffectiveMarginEntry(o.Side, price, cost.FeeRate)
	p, err := s.store.GetPortfolio(ctx, o.ClientID)
	if err != nil {
		return false
	}
	// Available cash must cover margin + fee; reserved margin already set aside conceptually —
	// reservation is not deducted from cashBalance, only from available — so we debit now
	// and clear reservation in the same tx.
	margin, err := domain.InitialMargin(o.Quantity, price, o.Leverage)
	if err != nil {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "invalid margin", now)
		return false
	}
	needCash := margin + openFee
	// Cash check: full balance minus other spot reserves and OTHER margin reserves (exclude this order)
	reservedSpot, _ := s.store.SumReservedCash(ctx, o.ClientID)
	allMarginRes, _ := s.store.SumReservedMargin(ctx, o.ClientID)
	otherMarginRes := allMarginRes - o.ReservedMargin
	if otherMarginRes < 0 {
		otherMarginRes = 0
	}
	avail := domain.AvailableCash(p.CashBalance, reservedSpot+otherMarginRes)
	if avail+1e-9 < needCash {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "insufficient cash at fill", now)
		return false
	}
	nPos, err := s.store.CountOpenMarginPositions(ctx, o.ClientID)
	if err != nil || nPos >= domain.MaxOpenMarginPositions {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "max positions", now)
		return false
	}
	mode := p.MarginMode
	if mode == "" {
		mode = domain.MarginModeIsolated
	}
	borrowP, debtAsset, err := domain.BorrowedPrincipalOnOpen(o.Side, o.Quantity, price, o.Leverage)
	if err != nil {
		return false
	}
	if err := s.checkBorrowLimit(ctx, p, domain.DebtNotionalQuote(o.Side, borrowP, 0, price)); err != nil {
		_ = s.store.RejectMarginOrder(ctx, o.ID, "borrow limit", now)
		return false
	}
	liq, err := s.liqPriceFor(o.Side, price, o.Quantity, margin, borrowP, 0)
	if err != nil {
		return false
	}
	lastInt := now.Truncate(time.Hour)
	pos := domain.MarginPosition{
		ID: uuid.NewString(), ClientID: o.ClientID, Exchange: o.Exchange, Symbol: o.Symbol, Side: o.Side, Mode: mode,
		Quantity: o.Quantity, EntryPrice: entry, Leverage: o.Leverage, Margin: margin,
		DebtPrincipal: borrowP, DebtInterest: 0, DebtAsset: debtAsset, LastInterestAt: lastInt,
		LiquidationPrice: liq, StopLoss: o.StopLoss, TakeProfit: o.TakeProfit,
		Status: domain.MarginPositionOpen, OpenedAt: now, UpdatedAt: now,
	}
	tr := domain.MarginTrade{
		ID: uuid.NewString(), ClientID: o.ClientID, PositionID: pos.ID, Exchange: o.Exchange, Symbol: o.Symbol,
		Side: o.Side, Action: "open", Quantity: o.Quantity, Price: price, Notional: o.Quantity * price,
		MarginDelta: -needCash, Leverage: o.Leverage, Fee: openFee, CreatedAt: now,
	}
	p.CashBalance -= needCash
	p.UpdatedAt = now
	// Reserved margin is released in ApplyMarginOpenFromOrder (reserved_margin=0 on fill).
	if err := s.store.ApplyMarginOpenFromOrder(ctx, p, o.ID, pos, tr, now); err != nil {
		return false
	}
	_ = s.recomputeCrossLiquidations(ctx, o.ClientID, now)
	return true
}

func (s *Service) closeMarginAt(ctx context.Context, pos *domain.MarginPosition, closeQty, price float64, reason string) (*domain.MarginPosition, *domain.MarginTrade, error) {
	clientID := pos.ClientID
	id := pos.ID
	for attempt := 0; attempt < maxDebtMutationRetries; attempt++ {
		p, err := s.store.GetPortfolio(ctx, clientID)
		if err != nil {
			return nil, nil, err
		}
		cur, err := s.store.GetMarginPosition(ctx, clientID, id)
		if err != nil {
			return nil, nil, err
		}
		if cur.Status != domain.MarginPositionOpen {
			// Concurrent liquidation/user close already finished — single-close invariant.
			return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
		}
		now := time.Now().UTC()
		// Accrue only (no nested liquidate) so close and liquidate cannot both fire from here.
		_ = s.accrueInterestOnly(ctx, cur, now)
		cur, err = s.store.GetMarginPosition(ctx, clientID, id)
		if err != nil {
			return nil, nil, err
		}
		if cur.Status != domain.MarginPositionOpen {
			return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
		}
		expected := domain.PositionCloseSnapshotFromPos(cur)

		cq := closeQty
		if cq > cur.Quantity {
			cq = cur.Quantity
		}
		if cq <= domain.PositionEpsilon {
			return nil, nil, fmt.Errorf("%w: nothing to close", domain.ErrInvalidArgument)
		}
		full := cq+domain.PositionEpsilon >= cur.Quantity
		frac := cq / cur.Quantity
		if full {
			frac = 1
		}
		marginRelease := cur.Margin * frac
		pp := cur.DebtPrincipal * frac
		ip := cur.DebtInterest * frac
		remainP := cur.DebtPrincipal - pp
		remainI := cur.DebtInterest - ip

		cost := s.paperCost(cur.Exchange)
		fill := domain.ApplySlippage(price, domain.MarginCloseTradeSide(cur.Side), cost.SlippageRate)
		if fill <= 0 {
			fill = price
		}
		closeFee := domain.FeeAmount(cq, fill, cost.FeeRate)
		exit := domain.EffectiveMarginExit(cur.Side, fill, cost.FeeRate)
		realized := domain.MarginRealizedPnL(cur.Side, cq, cur.EntryPrice, exit)
		// Cash uses raw slipped prices minus the close fee so the open fee
		// (already deducted at fill and baked into entry) is not charged twice.
		rawEntry := domain.RawFillFromEffectiveEntry(cur.Side, cur.EntryPrice, cost.FeeRate)
		rawPnL := domain.MarginRealizedPnL(cur.Side, cq, rawEntry, fill)
		// Collateral model: open only debited initial margin (+ open fee). Close
		// returns margin + raw PnL − close fee − interest. Do NOT also subtract
		// remaining quote principal — that double-counts the borrow (not in cash).
		cashDelta := marginRelease + rawPnL - closeFee
		switch cur.DebtAsset {
		case domain.DebtAssetBase:
			cashDelta -= ip * price
		default:
			// Interest only; principal (pp) is reduced on the position, not cash.
			cashDelta -= ip
		}
		mode := cur.Mode
		if mode == "" {
			mode = p.MarginMode
		}
		p.CashBalance, cashDelta = domain.ApplyMarginCloseCash(mode, p.CashBalance, cashDelta)
		p.RealizedPnLTotal += realized
		p.UpdatedAt = now

		closeReason := reason
		action := "close"
		switch reason {
		case domain.MarginCloseLiquidation:
			action = domain.MarginCloseLiquidation
		case domain.MarginCloseStopLoss:
			action = domain.MarginCloseStopLoss
		case domain.MarginCloseTakeProfit:
			action = domain.MarginCloseTakeProfit
		case domain.MarginClosePartialUser:
			action = "partial_close"
		}

		// Full forced closes use a deterministic trade id (restart cannot double-apply).
		// Partial liquidations use a unique id each time; debt+qty CAS prevents the same
		// snapshot from inserting two records for the same quantity.
		tradeID := uuid.NewString()
		if full {
			if sysID := domain.SystemCloseTradeID(action, cur.ID); sysID != "" {
				tradeID = sysID
				if ok, herr := s.store.HasMarginTradeID(ctx, sysID); herr == nil && ok {
					return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
				}
			}
		}

		tr := domain.MarginTrade{
			ID: tradeID, ClientID: cur.ClientID, PositionID: cur.ID, Exchange: cur.Exchange, Symbol: cur.Symbol,
			Side: cur.Side, Action: action, Quantity: cq, Price: fill, Notional: cq * fill,
			RealizedPnL: realized, MarginDelta: cashDelta, PrincipalPaid: pp, InterestPaid: ip,
			Leverage: cur.Leverage, Fee: closeFee, CreatedAt: now,
		}

		updated := *cur
		updated.RealizedPnL += realized
		updated.UpdatedAt = now
		if full {
			updated.Quantity = 0
			updated.Margin = 0
			updated.DebtPrincipal = 0
			updated.DebtInterest = 0
			updated.Status = domain.MarginPositionClosed
			updated.CloseReason = closeReason
			updated.ClosedAt = &now
			updated.StopLoss = nil
			updated.TakeProfit = nil
		} else {
			updated.Quantity = cur.Quantity - cq
			updated.Margin = cur.Margin - marginRelease
			if updated.Margin < 0 {
				updated.Margin = 0
			}
			updated.DebtPrincipal = remainP
			updated.DebtInterest = remainI
			if updated.DebtPrincipal < 0 {
				updated.DebtPrincipal = 0
			}
			if updated.DebtInterest < 0 {
				updated.DebtInterest = 0
			}
			liq, lerr := s.liqFromStoredEntry(cur, updated.Quantity, updated.Margin, updated.DebtPrincipal, updated.DebtInterest)
			if lerr == nil {
				updated.LiquidationPrice = liq
			}
			if closeReason == "" {
				closeReason = domain.MarginClosePartialUser
			}
		}
		fillIdempCloseIDs(ctx, cur.ID, tr.ID)
		err = s.store.ApplyMarginClose(ctx, p, updated, tr, full, expected)
		if isIdempotencyHit(err) {
			if rec := domain.IdempotencyFromContext(ctx); rec != nil {
				if hit, rerr := s.replayAfterHit(ctx, rec.ClientID, rec.Key, rec.RequestHash); rerr == nil && hit != nil {
					return s.replayMarginClose(ctx, hit)
				}
			}
		}
		if err == nil {
			_ = s.recomputeCrossLiquidations(ctx, cur.ClientID, now)
			out, gerr := s.store.GetMarginPosition(ctx, cur.ClientID, cur.ID)
			if gerr != nil {
				return &updated, &tr, nil
			}
			if out.Status == domain.MarginPositionOpen {
				s.markMarginPosition(ctx, out)
			}
			return out, &tr, nil
		}
		// Another closer won — do not insert a second close trade.
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: position is not open", domain.ErrInvalidArgument)
		}
		if !errors.Is(err, domain.ErrConflict) {
			return nil, nil, err
		}
		// Interest, repay, or partial close raced — retry with fresh debt+qty snapshot.
	}
	return nil, nil, fmt.Errorf("%w: concurrent debt update, try again", domain.ErrConflict)
}
