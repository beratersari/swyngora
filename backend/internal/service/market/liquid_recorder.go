package market

import (
	"context"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// LiquidPerpSymbols is the recorder universe: the seed pairs, then the
// highest-volume USDT spot names, capped at limit. Spot volume is the public
// stand-in for which perpetuals are liquid. Binance is tried first. A failed
// catalog still returns the seeds.
func (s *Service) LiquidPerpSymbols(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = domain.LiquidRecorderLimit
	}
	if s == nil {
		return domain.MergeLiquidSymbols(domain.DefaultFuturesHistorySymbols, nil, limit), nil
	}
	ranked, err := s.orderCountUniverse(ctx, string(domain.ExchangeBinance), "USDT", limit)
	if err != nil || len(ranked) == 0 {
		var err2 error
		ranked, err2 = s.orderCountUniverse(ctx, string(domain.ExchangeBybit), "USDT", limit)
		if err2 != nil {
			err = err2
		} else {
			err = nil
		}
	}
	return domain.MergeLiquidSymbols(domain.DefaultFuturesHistorySymbols, ranked, limit), err
}
