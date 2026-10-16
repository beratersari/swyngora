package portfolio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// FillPocket is the open paper fill on this book's coin, measured against the
// nearer estimated liquidation pocket. Equities return a mark with no prices.
// A book with no open lot and no margin position returns open=false and does
// not read the map. A venue the map does not cover leaves the pocket blank.
func (s *Service) FillPocket(ctx context.Context, clientID, exchange, symbol string, portfolioID ...string) (domain.FillPocketMark, error) {
	if strings.TrimSpace(exchange) == "" {
		return domain.FillPocketMark{}, fmt.Errorf("%w: exchange is required", domain.ErrInvalidArgument)
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		return domain.FillPocketMark{}, fmt.Errorf("%w: unknown exchange %q", domain.ErrInvalidArgument, exchange)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return domain.FillPocketMark{}, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	out := domain.FillPocketMark{Symbol: sym, Exchange: string(ex), Note: domain.FillPocketNote}
	if domain.IsEquityExchange(ex) {
		out.Equity = true
		return out, nil
	}
	if _, err := domain.ValidateOpenInterestSymbol(sym); err != nil {
		return domain.FillPocketMark{}, err
	}
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return domain.FillPocketMark{}, err
	}
	bookID := p.BookID()
	lots, err := s.store.ListTaxLots(ctx, bookID, ex, sym, true)
	if err != nil {
		return domain.FillPocketMark{}, err
	}
	var fill float64
	kind := ""
	if lot, ok := domain.LatestOpenLot(lots); ok && lot.Price > 0 {
		fill = lot.Price
		kind = domain.FillPocketSpot
		if strings.TrimSpace(lot.SourceTradeID) != "" {
			tr, terr := s.store.GetTrade(ctx, bookID, lot.SourceTradeID)
			if terr != nil && !errors.Is(terr, domain.ErrNotFound) {
				return domain.FillPocketMark{}, terr
			}
			if tr != nil && tr.Price > 0 {
				fill = tr.Price
			}
		}
	}
	margin, err := s.openMarginFor(ctx, bookID, ex, sym)
	if err != nil {
		return domain.FillPocketMark{}, err
	}
	var marginLiq float64
	if margin != nil {
		marginLiq = margin.LiquidationPrice
		if fill <= 0 && margin.EntryPrice > 0 {
			fill = margin.EntryPrice
			kind = domain.FillPocketMargin
		}
	}
	if fill <= 0 && marginLiq <= 0 {
		return out, nil
	}
	facts := domain.FillPocketFacts{
		Symbol: sym, Exchange: string(ex),
		FillPrice: fill, FillKind: kind, MarginLiqPrice: marginLiq,
	}
	if s.pockets == nil {
		return domain.BuildFillPocketMark(facts), nil
	}
	pocket, err := s.pockets.CloserPocket(ctx, sym)
	if err != nil {
		if ctx.Err() != nil {
			return domain.FillPocketMark{}, err
		}
		slog.Warn("fill pocket map", "symbol", sym, "err", err)
		return domain.BuildFillPocketMark(facts), nil
	}
	facts.Pocket = pocket
	facts.PocketLoaded = true
	return domain.BuildFillPocketMark(facts), nil
}

func (s *Service) openMarginFor(ctx context.Context, bookID string, ex domain.Exchange, symbol string) (*domain.MarginPosition, error) {
	list, err := s.store.ListOpenMarginPositions(ctx, bookID)
	if err != nil {
		return nil, err
	}
	var best *domain.MarginPosition
	for i := range list {
		row := &list[i]
		if row.Exchange != ex || row.Symbol != symbol || row.Status != domain.MarginPositionOpen {
			continue
		}
		if best == nil || laterMargin(*row, *best) {
			best = row
		}
	}
	return best, nil
}

func laterMargin(a, b domain.MarginPosition) bool {
	if !a.OpenedAt.Equal(b.OpenedAt) {
		return a.OpenedAt.After(b.OpenedAt)
	}
	return a.ID > b.ID
}
