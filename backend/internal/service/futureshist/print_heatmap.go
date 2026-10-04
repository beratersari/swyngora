package futureshist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// AggregateLiquidationPrints reads price × time bins from the store.
// A store without the aggregate interface falls back to the capped event list.
func (s *Service) AggregateLiquidationPrints(ctx context.Context, q domain.MeasuredPrintQuery) (domain.MeasuredPrintScan, error) {
	if s == nil || s.Store == nil {
		return domain.MeasuredPrintScan{}, nil
	}
	q.Symbol = domain.NormalizeLiquidationSymbol(q.Symbol)
	if agg, ok := s.Store.(domain.LiquidationPrintAggregator); ok {
		return agg.AggregateLiquidationPrints(ctx, q)
	}
	return s.aggregatePrintsFromList(ctx, q)
}

// LiquidationPrintPriceSpan is the stored print high and low for the window.
// A store without the method returns zeros.
func (s *Service) LiquidationPrintPriceSpan(ctx context.Context, symbol string, from, to time.Time, exchanges []domain.Exchange) (float64, float64, error) {
	if s == nil || s.Store == nil {
		return 0, 0, nil
	}
	if agg, ok := s.Store.(domain.LiquidationPrintAggregator); ok {
		return agg.LiquidationPrintPriceSpan(ctx, symbol, from, to, exchanges)
	}
	return 0, 0, nil
}

func (s *Service) aggregatePrintsFromList(ctx context.Context, q domain.MeasuredPrintQuery) (domain.MeasuredPrintScan, error) {
	exchanges := q.Exchanges
	if len(exchanges) == 0 {
		exchanges = domain.FuturesVenues()
	}
	events := make([]domain.LiquidationEvent, 0, 64)
	for _, ex := range exchanges {
		rows, err := s.Store.ListLiquidations(ctx, string(ex), q.Symbol, q.Origin, q.Until, domain.MaxHuntHeatmapHistory)
		if err != nil {
			return domain.MeasuredPrintScan{}, err
		}
		for _, e := range rows {
			if e.Exchange != ex {
				continue
			}
			events = append(events, e)
		}
	}
	return domain.BinMeasuredPrintEvents(q, events), nil
}
