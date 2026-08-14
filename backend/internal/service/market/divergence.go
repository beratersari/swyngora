package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVenueDivergence compares Binance vs Bybit for one coin.
func (s *Service) GetVenueDivergence(ctx context.Context, symbol string) (*domain.VenueDivergenceReport, error) {
	pos, err := s.GetPositioning(ctx, "all", symbol)
	if err != nil {
		return nil, err
	}
	var bin, byb domain.PositioningVenueReport
	var haveB, haveY bool
	for _, v := range pos.Venues {
		switch v.Exchange {
		case domain.ExchangeBinance:
			bin, haveB = v, true
		case domain.ExchangeBybit:
			byb, haveY = v, true
		}
	}
	if !haveB || !haveY {
		sym := symbol
		if pos != nil && pos.Symbol != "" {
			sym = pos.Symbol
		}
		return &domain.VenueDivergenceReport{
			Symbol:    sym,
			AsOf:      time.Now().UTC(),
			Alignment: domain.AlignUnknown,
			Title:     "Not enough venue data to compare",
			Summary:   "Need both Binance and Bybit prints for this coin. One venue is missing or failed.",
			Diffs:     []domain.VenueSignalDiff{},
			Note:      "Informational only.",
		}, nil
	}
	now := time.Now().UTC()
	if pos != nil && !pos.AsOf.IsZero() {
		now = pos.AsOf
	}
	got := domain.CompareVenuePositioning(pos.Symbol, bin, byb, now)
	return &got, nil
}
