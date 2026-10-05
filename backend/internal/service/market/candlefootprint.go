package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCandleFootprint returns buy and sell at the prices printed inside each candle.
// Closed minutes come from the flow store. The current minute comes from the
// live tape when that tape already covers it. This read does not walk REST.
func (s *Service) GetCandleFootprint(ctx context.Context, exchange, symbol, interval string, start, end *time.Time) (*domain.CandleFootprintReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	id, step, err := domain.ParseCandleFootprintInterval(interval)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	from, to, truncated, err := domain.ResolveCandleFootprintRange(step, start, end, now)
	if err != nil {
		return nil, err
	}
	s.noteFlowSymbol(symbol)
	want := configuredFutures(ex, func(v domain.Exchange) bool { return s.printPort(v) != nil })
	sources := make([]domain.CandleFootprintSource, 0, len(want))
	for _, v := range want {
		cells, present, err := s.candleFootprintCells(ctx, v, symbol, from, to, now)
		if err != nil {
			return nil, err
		}
		sources = append(sources, domain.CandleFootprintSource{Exchange: v, Cells: cells, Present: present})
	}
	out := domain.BuildCandleFootprints(symbol, ex, id, step, from, to, now, want, sources, truncated)
	return &out, nil
}

func (s *Service) candleFootprintCells(ctx context.Context, ex domain.Exchange, symbol string, from, to, now time.Time) ([]domain.FlowAtPricesCell, bool, error) {
	var stored []domain.FlowAtPricesCell
	if s != nil && s.flowHist != nil {
		var err error
		stored, err = s.flowHist.ListFlowCells(ctx, ex, symbol, from, to)
		if err != nil {
			return nil, false, err
		}
	}
	open := now.Truncate(time.Minute)
	live, covered := s.openMinuteCells(ex, symbol, from, to, open)
	if covered {
		kept := stored[:0]
		for _, c := range stored {
			if !c.Minute.UTC().Before(open) {
				continue
			}
			kept = append(kept, c)
		}
		stored = append(kept, live...)
	}
	return stored, len(stored) > 0 || covered, nil
}

func (s *Service) openMinuteCells(ex domain.Exchange, symbol string, from, to, open time.Time) ([]domain.FlowAtPricesCell, bool) {
	if s == nil || !to.After(open) {
		return nil, false
	}
	p := s.printPort(ex)
	mem, ok := p.(domain.MemoryPrintPort)
	if !ok || mem == nil {
		return nil, false
	}
	start := open
	if from.After(start) {
		start = from
	}
	prints, covered := mem.MemoryPrints(symbol, start, to)
	if !covered {
		return nil, false
	}
	cells, _ := domain.AccumulateFlowPrints(prints, time.Time{})
	out := make([]domain.FlowAtPricesCell, 0, len(cells))
	for _, c := range cells {
		if c.Minute.Before(open) || !c.Minute.Before(to) {
			continue
		}
		c.Exchange = ex
		c.Symbol = symbol
		out = append(out, c)
	}
	return out, true
}
