package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const maxPainNote = "Largest estimated liquidation pockets above and below last price. Size is modeled open-interest at assumed leverage plus 24h observed clusters — not a hunt target and not options max pain. Venues stay separate; combined above/below is the single largest pocket on that side (prices are never averaged). Informational only, not financial advice."

// GetLiquidationMaxPain is the biggest liquidation area above (shorts) and below (longs).
func (s *Service) GetLiquidationMaxPain(ctx context.Context, exchange, symbol string) (*domain.MaxPainReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	out := &domain.MaxPainReport{
		Symbol:      symbol,
		Exchange:    ex,
		AsOf:        now,
		Venues:      make([]domain.MaxPainVenue, 0, len(want)),
		Assumptions: domain.DefaultHuntAssumptions(),
		Note:        maxPainNote,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			hun := s.huntOne(ctx, v, symbol, now, domain.DefaultHuntScoreWeights())
			row := domain.MaxPainFromVenue(hun)
			mu.Lock()
			out.Venues = append(out.Venues, row)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	out.Above, out.Below = domain.CombineMaxPainPockets(out.Venues)
	out.Summary = domain.MaxPainSummary(out.Above, out.Below)
	return out, nil
}
