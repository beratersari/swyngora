package portfolio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListLots returns tax lots for the selected book (default open only).
func (s *Service) ListLots(ctx context.Context, clientID, exchange, symbol, status string, portfolioID ...string) ([]domain.TaxLot, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	st := strings.ToLower(strings.TrimSpace(status))
	openOnly := st != "all" && st != "closed"
	var ex domain.Exchange
	if strings.TrimSpace(exchange) != "" {
		ex = domain.ParseExchange(exchange)
		if ex == "" {
			return nil, fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
	}
	lots, err := s.store.ListTaxLots(ctx, p.BookID(), ex, symbol, openOnly)
	if err != nil {
		return nil, err
	}
	if st == "closed" {
		out := lots[:0]
		for _, l := range lots {
			if !l.Open() {
				out = append(out, l)
			}
		}
		return out, nil
	}
	return lots, nil
}

func pendingLotMethod(raw string) domain.LotMethod {
	m, err := domain.NormalizeLotMethod(raw)
	if err != nil {
		return domain.DefaultLotMethod
	}
	return m
}

func (s *Service) loadOpenLots(ctx context.Context, bookID string, ex domain.Exchange, symbol string) ([]domain.TaxLot, error) {
	return s.store.ListOpenTaxLots(ctx, bookID, ex, symbol)
}

// prepareBuyLots backfills a legacy position if needed and adds the new buy lot.
func prepareBuyLots(bookID string, ex domain.Exchange, symbol string, existing []domain.TaxLot, posQty, avgCost, buyQty, buyPrice float64, tradeID string, now time.Time) *domain.LotOps {
	ops := &domain.LotOps{}
	if domain.OpenLotQuantity(existing) <= domain.PositionEpsilon && posQty > domain.PositionEpsilon {
		ops.Created = append(ops.Created, domain.SyntheticOpeningLot(uuid.NewString(), bookID, ex, symbol, posQty, avgCost, now))
	}
	ops.Created = append(ops.Created, domain.NewTaxLot(uuid.NewString(), bookID, ex, symbol, buyQty, buyPrice, now, tradeID))
	return ops
}

func prepareSellLots(existing []domain.TaxLot, bookID string, ex domain.Exchange, symbol string, posQty, avgCost, sellQty, sellPrice float64, method domain.LotMethod, tradeID string, now time.Time, feeRate float64) (*domain.LotOps, float64, float64, error) {
	open := append([]domain.TaxLot(nil), existing...)
	ops := &domain.LotOps{}
	if domain.OpenLotQuantity(open) <= domain.PositionEpsilon && posQty > domain.PositionEpsilon {
		syn := domain.SyntheticOpeningLot(uuid.NewString(), bookID, ex, symbol, posQty, avgCost, now)
		ops.Created = append(ops.Created, syn)
		open = append(open, syn)
	}
	fills, updated, realized, err := domain.ConsumeLots(open, sellQty, sellPrice, method, now, feeRate)
	if err != nil {
		return nil, 0, 0, err
	}
	for i := range fills {
		fills[i].TradeID = tradeID
		if fills[i].ID == "" {
			fills[i].ID = uuid.NewString()
		}
	}
	merged := domain.MergeLotUpdates(open, updated)
	ops.Updated = updated
	ops.Fills = fills
	return ops, realized, domain.AvgCostFromLots(merged), nil
}
