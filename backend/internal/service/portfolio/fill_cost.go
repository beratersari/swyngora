package portfolio

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// rememberExecutionQuote stores the comparison and returns it with an id the ticket can send back.
// A store failure still returns the prices. The id is omitted, so nothing later claims a quote was kept.
func (s *Service) rememberExecutionQuote(ctx context.Context, clientID string, got domain.ExecutionCompare) domain.ExecutionCompare {
	if s == nil || s.store == nil || strings.TrimSpace(clientID) == "" {
		return got
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	saved := got
	saved.ID = id
	err := s.store.SaveExecutionQuote(ctx, domain.StoredExecutionQuote{
		ID: id, ClientID: clientID, Compare: saved, CreatedAt: now,
	})
	if err != nil {
		slog.Default().Debug("execution quote", "err", err)
		return got
	}
	if derr := s.store.DeleteExpiredExecutionQuotes(ctx, now.Add(-domain.ExecutionQuoteTTL)); derr != nil {
		slog.Default().Debug("execution quote cleanup", "err", derr)
	}
	return saved
}

// takeFillQuote loads the comparison the ticket named, when it still matches this order.
// A missing, stale, or different quote returns nil. The order is still placed.
func (s *Service) takeFillQuote(ctx context.Context, clientID, quoteID string, style domain.ExecutionStyle, symbol string, side domain.TradeSide, qty float64, venue domain.Exchange, slipPct float64, minutes, slices int) *domain.FillCostCheck {
	quoteID = strings.TrimSpace(quoteID)
	if s == nil || s.store == nil || quoteID == "" {
		return nil
	}
	q, err := s.store.GetExecutionQuote(ctx, clientID, quoteID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.Default().Debug("execution quote read", "err", err)
		}
		return nil
	}
	if !domain.QuoteFresh(q.CreatedAt, time.Now().UTC()) {
		return nil
	}
	c := q.Compare
	canon, cerr := domain.CanonicalRouteSymbol(symbol)
	if cerr != nil || canon != c.Symbol || c.Side != side || math.Abs(c.Quantity-qty) > 1e-8 {
		return nil
	}
	row, ok := executionRow(c, style)
	if !ok || row.Status != domain.ExecutionFilled {
		return nil
	}
	if style == domain.ExecutionMarket && c.Venue != venue {
		return nil
	}
	if style == domain.ExecutionBestCost || style == domain.ExecutionSpread {
		if math.Abs(c.MaxSlippagePct-slipPct) > 1e-6 {
			return nil
		}
	}
	if style == domain.ExecutionSpread {
		plan, perr := domain.NormalizeSpreadSchedule(minutes, slices)
		if perr != nil || plan.Minutes != row.SpreadMinutes || plan.SliceCount != row.SliceCount {
			return nil
		}
	}
	return &domain.FillCostCheck{
		ID: q.ID, ClientID: clientID, Style: style, Symbol: symbol, Side: side,
		ExpectedFee: row.Fee, ExpectedSlippage: row.SlippageCash(side), TouchPrice: row.TouchPrice,
		QuotedQuantity: row.FilledQuantity, QuotedAt: q.CreatedAt, Status: domain.FillCostOpen,
	}
}

func executionRow(c domain.ExecutionCompare, style domain.ExecutionStyle) (domain.ExecutionQuote, bool) {
	for _, row := range c.Rows {
		if row.Style == style {
			return row, true
		}
	}
	return domain.ExecutionQuote{}, false
}

// keepFillCost stores the quote against the fills of an order that just finished,
// or against a spread order that has not filled yet.
// The lead trade is the last fill. A quote already used by another order is left alone.
func (s *Service) keepFillCost(ctx context.Context, seed *domain.FillCostCheck, trades []domain.Trade, spreadID string, complete bool) *domain.FillCostCheck {
	if s == nil || s.store == nil || seed == nil {
		return nil
	}
	check := *seed
	check.SpreadOrderID = spreadID
	if complete {
		check.Status = domain.FillCostComplete
	}
	for _, tr := range trades {
		check.ActualFee += tr.Fee
		check.ActualSlippage += domain.SlippageCash(check.Side, check.TouchPrice, tr.Price, tr.Quantity)
		check.FilledQuantity += tr.Quantity
		check.LeadTradeID = tr.ID
	}
	created, err := s.store.SaveFillCostCheck(ctx, check)
	if err != nil {
		slog.Default().Debug("fill cost", "err", err)
		return nil
	}
	if !created {
		return nil
	}
	if check.Status == domain.FillCostComplete && check.LeadTradeID != "" {
		return &check
	}
	return nil
}

// applySpreadFillCost adds this slice to the quote kept for the spread order.
// The difference stays off the trade list until the order is filled or canceled.
func (s *Service) applySpreadFillCost(ctx context.Context, o *domain.SpreadOrder, fills []domain.RoutedFill) {
	if s == nil || s.store == nil || o == nil || strings.TrimSpace(o.ID) == "" {
		return
	}
	check, err := s.store.GetFillCostBySpread(ctx, o.ClientID, o.ID)
	if err != nil || check == nil {
		return
	}
	var fee, slip, qty float64
	var lead string
	for _, fill := range fills {
		tr := fill.Trade
		fee += tr.Fee
		slip += domain.SlippageCash(check.Side, check.TouchPrice, tr.Price, tr.Quantity)
		qty += tr.Quantity
		lead = tr.ID
	}
	complete := o.Status != domain.SpreadStatusOpen
	if err := s.store.AddFillCostActual(ctx, o.ClientID, o.ID, lead, fee, slip, qty, complete); err != nil {
		slog.Default().Debug("fill cost slice", "id", o.ID, "err", err)
	}
}

// attachFillCosts sets CostCheck on the newest fill of each finished order in this page.
func (s *Service) attachFillCosts(ctx context.Context, clientID string, list []domain.Trade) {
	if s == nil || s.store == nil || len(list) == 0 {
		return
	}
	ids := make([]string, len(list))
	for i := range list {
		ids[i] = list[i].ID
	}
	checks, err := s.store.ListCompleteFillCosts(ctx, clientID, ids)
	if err != nil {
		slog.Default().Debug("fill cost list", "err", err)
		return
	}
	for i := range list {
		if c, ok := checks[list[i].ID]; ok {
			list[i].CostCheck = &c
		}
	}
}
