package portfolio

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// OrderPreviewInput is a paper spot or margin order that should be priced, not placed.
type OrderPreviewInput struct {
	ClientID        string
	PortfolioID     string
	OwnerClientID   string
	Product         string // spot (default) or margin
	Exchange        string
	Symbol          string
	Side            string
	Type            string
	Quantity        float64
	TriggerPrice    float64
	TakeProfitPrice float64
	StopLossPrice   float64
	TrailType       string
	TrailValue      float64
	LotMethod       string
	Leverage        int
	LimitPrice      float64
}

// riskSnap is the live book used to price one preview. It is not saved as an order.
type riskSnap struct {
	lim         domain.RiskLimits
	equity      float64
	asset       string
	assetVal    float64
	exposure    float64
	startEquity float64
	dailyMark   float64
}

func (r riskSnap) build(additional float64, applies bool) domain.OrderPreviewRisk {
	return domain.BuildOrderPreviewRisk(r.lim, r.startEquity, r.dailyMark, r.equity, r.asset, r.assetVal, r.exposure, additional, applies)
}

// sellView keeps today's loss on the card and shows the coin getting smaller.
// A sell is not a new-risk order, so it is not blocked.
func (r riskSnap) sellView(reduction float64) domain.OrderPreviewRisk {
	out := r.build(0, false)
	if reduction > 0 && r.equity > domain.PositionEpsilon {
		w := domain.ProjectedAssetWeightPct(r.equity, r.assetVal, -reduction)
		e := domain.ProjectedExposurePct(r.equity, r.exposure, -reduction)
		out.ProjectedWeightPct = &w
		out.ProjectedExposurePct = &e
	}
	return out
}

// PreviewOrder prices a paper order and checks risk limits without writing a fill.
func (s *Service) PreviewOrder(ctx context.Context, in OrderPreviewInput) (*domain.OrderPreview, error) {
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
	product := strings.ToLower(strings.TrimSpace(in.Product))
	if product == "" {
		product = "spot"
	}
	if product != "spot" && product != "margin" {
		return nil, fmt.Errorf("%w: product must be spot or margin", domain.ErrInvalidArgument)
	}
	if in.Quantity < domain.MinTradeQuantity || in.Quantity > domain.MaxTradeQuantity ||
		math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) {
		return nil, fmt.Errorf("%w: quantity out of range", domain.ErrInvalidArgument)
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	lotMethod, err := domain.NormalizeLotMethod(in.LotMethod)
	if err != nil {
		return nil, err
	}
	last, err := s.lastPrice(ctx, string(ex), sym)
	if err != nil {
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
	if err := domain.RequireQuoteMatchesCurrency(ex, sym, p.Currency); err != nil {
		return nil, err
	}
	cost := s.paperCost(ex)
	avail, err := s.availableCashForTrading(ctx, clientID, p.CashBalance)
	if err != nil {
		return nil, err
	}
	posQty, avg := 0.0, 0.0
	pos, perr := s.store.GetPosition(ctx, clientID, ex, sym)
	if perr == nil && pos != nil {
		posQty, avg = pos.Quantity, pos.AvgCost
	} else if perr != nil && !errors.Is(perr, domain.ErrNotFound) {
		return nil, perr
	}
	reservedQty, err := s.store.SumReservedQuantity(ctx, clientID, ex, sym)
	if err != nil {
		return nil, err
	}
	availQty := domain.AvailablePosition(posQty, reservedQty)
	lots, err := s.loadOpenLots(ctx, clientID, ex, sym)
	if err != nil {
		return nil, err
	}
	base, _ := domain.SplitBaseQuote(ex, sym)
	snap, err := s.previewSnap(ctx, p, base)
	if err != nil {
		return nil, err
	}

	out := &domain.OrderPreview{
		Product:           product,
		Exchange:          ex,
		Symbol:            sym,
		Quantity:          in.Quantity,
		LastPrice:         last,
		FeeRate:           cost.FeeRate,
		SlippageRate:      cost.SlippageRate,
		AvailableCash:     avail,
		AvailableQuantity: availQty,
		CashAfter:         p.CashBalance,
		AvailableAfter:    avail,
		Note:              domain.OrderPreviewNote,
	}
	if product == "margin" {
		return s.previewMargin(ctx, p, out, in, ex, last, cost, avail, snap)
	}
	return s.previewSpot(out, in, clientID, ex, sym, last, cost, avail, p.CashBalance, posQty, avg, availQty, lots, lotMethod, snap)
}

func (s *Service) previewSnap(ctx context.Context, p *domain.Portfolio, asset string) (riskSnap, error) {
	lim, snap, err := s.loadRiskWithDaySnapshot(ctx, p)
	if err != nil {
		return riskSnap{}, err
	}
	st, err := s.buildRiskStatus(ctx, p, lim, snap)
	if err != nil {
		return riskSnap{}, err
	}
	asset = domain.NormalizeAllocationAsset(asset)
	var assetVal float64
	for _, a := range st.Assets {
		if a.Asset == asset {
			assetVal = a.Value
			break
		}
	}
	return riskSnap{
		lim:         lim,
		equity:      st.Equity,
		asset:       asset,
		assetVal:    assetVal,
		exposure:    st.ExposureValue,
		startEquity: st.StartOfDayEquity,
		dailyMark:   st.StartOfDayEquity + st.DailyPnL,
	}, nil
}

func (s *Service) previewSpot(
	out *domain.OrderPreview,
	in OrderPreviewInput,
	bookID string,
	ex domain.Exchange,
	sym string,
	last float64,
	cost domain.TradingCost,
	avail, cash, posQty, avg, availQty float64,
	lots []domain.TaxLot,
	lotMethod domain.LotMethod,
	snap riskSnap,
) (*domain.OrderPreview, error) {
	typ := strings.ToLower(strings.TrimSpace(in.Type))
	if typ == "" {
		typ = "market"
	}
	out.Type = typ
	switch typ {
	case "market":
		side := domain.TradeSide(strings.ToLower(strings.TrimSpace(in.Side)))
		if !domain.IsValidTradeSide(string(side)) {
			return nil, fmt.Errorf("%w: side must be buy or sell", domain.ErrInvalidArgument)
		}
		out.Side = string(side)
		out.FillsNow = true
		fill, notional, fee, debit, credit := domain.QuoteFill(side, last, in.Quantity, cost)
		if fill <= 0 {
			return nil, fmt.Errorf("%w: invalid fill price after slippage", domain.ErrInvalidArgument)
		}
		out.ReferencePrice = last
		out.FillPrice = fill
		out.Notional = notional
		out.Fee = fee
		out.CashDebit = debit
		out.CashCredit = credit
		if side == domain.TradeSideBuy {
			out.CashAfter = cash - debit
			out.AvailableAfter = avail - debit
			out.InsufficientCash = avail+1e-9 < debit
			out.Risk = snap.build(in.Quantity*fill, true)
		} else {
			out.CashAfter = cash + credit
			out.AvailableAfter = avail + credit
			out.InsufficientQuantity = in.Quantity > availQty+domain.PositionEpsilon
			if !out.InsufficientQuantity {
				out.EstimatedRealizedPnL = sellPnL(lots, bookID, ex, sym, posQty, avg, in.Quantity, fill, lotMethod, cost.FeeRate)
			}
			out.Risk = snap.sellView(in.Quantity * fill)
		}
	case "limit_buy", "limit_sell", "stop_loss":
		if badPrice(in.TriggerPrice) {
			return nil, fmt.Errorf("%w: triggerPrice out of range", domain.ErrInvalidArgument)
		}
		side := domain.SideForPendingType(domain.PendingOrderType(typ))
		out.Side = string(side)
		fillResting(out, side, in.TriggerPrice, in.Quantity, cost, avail, cash, availQty, lots, bookID, ex, sym, posQty, avg, lotMethod, snap, typ == "limit_buy")
	case "trailing_stop":
		trailType, err := domain.NormalizeTrailType(in.TrailType)
		if err != nil {
			return nil, err
		}
		if err := domain.ValidateTrailValue(trailType, in.TrailValue); err != nil {
			return nil, err
		}
		trigger := domain.TrailStopPrice(last, in.TrailValue, trailType)
		if trigger <= 0 {
			return nil, fmt.Errorf("%w: trail produces non-positive stop price at current market", domain.ErrInvalidArgument)
		}
		out.Side = string(domain.TradeSideSell)
		fillResting(out, domain.TradeSideSell, trigger, in.Quantity, cost, avail, cash, availQty, lots, bookID, ex, sym, posQty, avg, lotMethod, snap, false)
	case "oco":
		if err := domain.ValidateOCOPrices(in.TakeProfitPrice, in.StopLossPrice); err != nil {
			return nil, err
		}
		out.Side = string(domain.TradeSideSell)
		out.ReservedQuantity = in.Quantity
		out.ReferencePrice = last
		out.InsufficientQuantity = in.Quantity > availQty+domain.PositionEpsilon
		out.Legs = []domain.OrderPreviewLeg{
			sellLeg("take_profit", in.TakeProfitPrice, in.Quantity, cost, lots, bookID, ex, sym, posQty, avg, lotMethod, out.InsufficientQuantity),
			sellLeg("stop_loss", in.StopLossPrice, in.Quantity, cost, lots, bookID, ex, sym, posQty, avg, lotMethod, out.InsufficientQuantity),
		}
		out.Risk = snap.sellView(0)
	case "bracket":
		if err := domain.ValidateBracketPrices(in.TriggerPrice, in.TakeProfitPrice, in.StopLossPrice); err != nil {
			return nil, err
		}
		out.Side = string(domain.TradeSideBuy)
		fillResting(out, domain.TradeSideBuy, in.TriggerPrice, in.Quantity, cost, avail, cash, availQty, lots, bookID, ex, sym, posQty, avg, lotMethod, snap, true)
		unit := domain.BuyUnitCost(out.FillPrice, cost.FeeRate)
		now := time.Now().UTC()
		ops := prepareBuyLots(bookID, ex, sym, lots, posQty, avg, in.Quantity, unit, "preview", now)
		hyp := append(append([]domain.TaxLot{}, lots...), ops.Created...)
		hypQty := posQty + in.Quantity
		hypAvg := domain.AvgCostFromLots(hyp)
		out.Legs = []domain.OrderPreviewLeg{
			sellLeg("take_profit", in.TakeProfitPrice, in.Quantity, cost, hyp, bookID, ex, sym, hypQty, hypAvg, lotMethod, false),
			sellLeg("stop_loss", in.StopLossPrice, in.Quantity, cost, hyp, bookID, ex, sym, hypQty, hypAvg, lotMethod, false),
		}
	default:
		return nil, fmt.Errorf("%w: type must be market, limit_buy, limit_sell, stop_loss, trailing_stop, oco, or bracket", domain.ErrInvalidArgument)
	}
	return out, nil
}

func fillResting(
	out *domain.OrderPreview,
	side domain.TradeSide,
	reference, qty float64,
	cost domain.TradingCost,
	avail, cash, availQty float64,
	lots []domain.TaxLot,
	bookID string,
	ex domain.Exchange,
	sym string,
	posQty, avg float64,
	lotMethod domain.LotMethod,
	snap riskSnap,
	addsRisk bool,
) {
	fill, notional, fee, debit, credit := domain.QuoteFill(side, reference, qty, cost)
	out.ReferencePrice = reference
	out.FillPrice = fill
	out.Notional = notional
	out.Fee = fee
	if side == domain.TradeSideBuy {
		reserve := domain.BuyReserveCash(qty, reference, cost)
		out.CashDebit = debit
		out.ReservedCash = reserve
		out.CashAfter = cash
		out.AvailableAfter = avail - reserve
		out.InsufficientCash = avail+1e-9 < reserve
		// The order guard uses the raw trigger, not the slipped fill.
		out.Risk = snap.build(qty*reference, addsRisk)
		return
	}
	out.CashCredit = credit
	out.ReservedQuantity = qty
	out.CashAfter = cash
	out.AvailableAfter = avail
	out.InsufficientQuantity = qty > availQty+domain.PositionEpsilon
	if !out.InsufficientQuantity {
		out.EstimatedRealizedPnL = sellPnL(lots, bookID, ex, sym, posQty, avg, qty, fill, lotMethod, cost.FeeRate)
	}
	out.Risk = snap.sellView(qty * fill)
}

func (s *Service) previewMargin(
	ctx context.Context,
	p *domain.Portfolio,
	out *domain.OrderPreview,
	in OrderPreviewInput,
	ex domain.Exchange,
	last float64,
	cost domain.TradingCost,
	avail float64,
	snap riskSnap,
) (*domain.OrderPreview, error) {
	side := domain.MarginSide(strings.ToLower(strings.TrimSpace(in.Side)))
	if side != domain.MarginLong && side != domain.MarginShort {
		return nil, fmt.Errorf("%w: side must be long or short", domain.ErrInvalidArgument)
	}
	if !domain.IsValidMarginLeverage(in.Leverage) {
		return nil, fmt.Errorf("%w: leverage must be between %d and %d", domain.ErrInvalidArgument, domain.MinMarginLeverage, domain.MaxMarginLeverage)
	}
	typ := strings.ToLower(strings.TrimSpace(in.Type))
	if typ == "" {
		typ = "market"
	}
	if typ != "market" && typ != "limit" {
		return nil, fmt.Errorf("%w: margin type must be market or limit", domain.ErrInvalidArgument)
	}
	out.Type = typ
	out.Side = string(side)
	out.Leverage = in.Leverage
	tradeSide := domain.MarginOpenTradeSide(side)
	reference := last
	if typ == "limit" {
		if badPrice(in.LimitPrice) {
			return nil, fmt.Errorf("%w: limitPrice out of range", domain.ErrInvalidArgument)
		}
		reference = in.LimitPrice
	} else {
		out.FillsNow = true
	}
	fill, notional, fee, _, _ := domain.QuoteFill(tradeSide, reference, in.Quantity, cost)
	if fill <= 0 {
		return nil, fmt.Errorf("%w: invalid fill price after slippage", domain.ErrInvalidArgument)
	}
	var sl, tp *float64
	if in.StopLossPrice > 0 {
		sl = &in.StopLossPrice
	}
	if in.TakeProfitPrice > 0 {
		tp = &in.TakeProfitPrice
	}
	if err := domain.ValidateMarginBrackets(side, fill, sl, tp); err != nil {
		return nil, err
	}
	margin, err := domain.InitialMargin(in.Quantity, fill, in.Leverage)
	if err != nil {
		return nil, err
	}
	need := margin + fee
	if typ == "limit" {
		need, err = s.marginReserveNeed(ex, side, in.Quantity, in.LimitPrice, in.Leverage)
		if err != nil {
			return nil, err
		}
	}
	borrowP, debtAsset, err := domain.BorrowedPrincipalOnOpen(side, in.Quantity, fill, in.Leverage)
	if err != nil {
		return nil, err
	}
	liq, err := s.liqPriceFor(side, fill, in.Quantity, margin, borrowP, 0)
	if err != nil {
		return nil, err
	}
	borrowNotional := domain.DebtNotionalQuote(side, borrowP, 0, fill)
	if err := s.checkBorrowLimit(ctx, p, borrowNotional); err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) && strings.Contains(err.Error(), "borrow limit") {
			out.BorrowLimitHit = true
		} else {
			return nil, err
		}
	}
	riskNotional := in.Quantity * fill
	if typ == "limit" {
		riskNotional = in.Quantity * in.LimitPrice
	}
	out.ReferencePrice = reference
	out.FillPrice = fill
	out.Notional = notional
	out.Fee = fee
	out.InitialMargin = margin
	out.LiquidationPrice = liq
	out.BorrowedPrincipal = borrowP
	out.BorrowedAsset = string(debtAsset)
	out.CashDebit = need
	if typ == "market" {
		out.CashAfter = p.CashBalance - need
		out.AvailableAfter = avail - need
	} else {
		out.ReservedCash = need
		out.CashAfter = p.CashBalance
		out.AvailableAfter = avail - need
	}
	out.InsufficientCash = avail+1e-9 < need
	out.Risk = snap.build(riskNotional, true)
	if out.BorrowLimitHit {
		out.Risk.Blocked = true
		out.Risk.Reasons = append(out.Risk.Reasons, "borrow limit would be exceeded")
	}
	return out, nil
}

func sellLeg(role string, reference, qty float64, cost domain.TradingCost, lots []domain.TaxLot, bookID string, ex domain.Exchange, sym string, posQty, avg float64, method domain.LotMethod, skipPnL bool) domain.OrderPreviewLeg {
	fill, notional, fee, _, credit := domain.QuoteFill(domain.TradeSideSell, reference, qty, cost)
	leg := domain.OrderPreviewLeg{
		Role: role, Side: domain.TradeSideSell, ReferencePrice: reference,
		FillPrice: fill, Notional: notional, Fee: fee, CashCredit: credit,
	}
	if !skipPnL {
		leg.EstimatedRealizedPnL = sellPnL(lots, bookID, ex, sym, posQty, avg, qty, fill, method, cost.FeeRate)
	}
	return leg
}

func sellPnL(lots []domain.TaxLot, bookID string, ex domain.Exchange, sym string, posQty, avg, qty, fill float64, method domain.LotMethod, feeRate float64) *float64 {
	_, realized, _, err := prepareSellLots(lots, bookID, ex, sym, posQty, avg, qty, fill, method, "preview", time.Now().UTC(), feeRate)
	if err != nil {
		return nil
	}
	return &realized
}

func badPrice(p float64) bool {
	return p < domain.MinTriggerPrice || p > domain.MaxTriggerPrice || math.IsNaN(p) || math.IsInf(p, 0)
}
