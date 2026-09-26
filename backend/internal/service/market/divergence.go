package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVenueDivergence compares Binance, Bybit, and OKX for one coin.
func (s *Service) GetVenueDivergence(ctx context.Context, symbol string) (*domain.VenueDivergenceReport, error) {
	pos, err := s.GetPositioning(ctx, "all", symbol)
	if err != nil {
		return nil, err
	}
	venues := make([]domain.PositioningVenueReport, 0, len(pos.Venues))
	for _, v := range pos.Venues {
		if !domain.IsFuturesVenue(v.Exchange) || v.Error != "" {
			continue
		}
		venues = append(venues, v)
	}
	now := time.Now().UTC()
	sym := symbol
	if pos != nil {
		if pos.Symbol != "" {
			sym = pos.Symbol
		}
		if !pos.AsOf.IsZero() {
			now = pos.AsOf
		}
	}
	got := domain.CompareVenuePositioning(sym, venues, now)
	return &got, nil
}
