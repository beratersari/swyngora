package market

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Service orchestrates market-data use cases. Handlers call this layer only.
type Service struct {
	market domain.MarketDataPort
	supply domain.SupplyPort
}

// New constructs a market application service.
func New(market domain.MarketDataPort, supply domain.SupplyPort) *Service {
	return &Service{market: market, supply: supply}
}

// GetCandles validates and fetches OHLCV candles for a Binance-style symbol.
func (s *Service) GetCandles(ctx context.Context, symbol, interval string, limit int, start, end *time.Time) ([]domain.Candle, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if !domain.IsValidInterval(interval) {
		return nil, fmt.Errorf("%w: interval must be one of %v", domain.ErrInvalidArgument, domain.SupportedIntervals)
	}
	if limit < 0 {
		return nil, fmt.Errorf("%w: limit must be >= 0", domain.ErrInvalidArgument)
	}
	if limit == 0 {
		limit = 100
	}
	if limit > 1000 {
		return nil, fmt.Errorf("%w: limit must be <= 1000", domain.ErrInvalidArgument)
	}

	q := domain.CandleQuery{
		Symbol:   symbol,
		Interval: domain.CandleInterval(interval),
		Limit:    limit,
	}
	if start != nil {
		q.StartTime = start.UTC()
	}
	if end != nil {
		q.EndTime = end.UTC()
	}
	if !q.StartTime.IsZero() && !q.EndTime.IsZero() && q.EndTime.Before(q.StartTime) {
		return nil, fmt.Errorf("%w: endTime must be >= startTime", domain.ErrInvalidArgument)
	}

	return s.market.GetCandles(ctx, q)
}

// GetTicker24h returns rolling 24h volume and price stats for a symbol.
func (s *Service) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return s.market.GetTicker24h(ctx, symbol)
}

// GetSupply returns circulating / total / max supply for a base asset (or pair).
func (s *Service) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	return s.supply.GetSupply(ctx, asset)
}

// ListIntervals returns supported candle intervals.
func (s *Service) ListIntervals() []domain.CandleInterval {
	out := make([]domain.CandleInterval, len(domain.SupportedIntervals))
	copy(out, domain.SupportedIntervals)
	return out
}
