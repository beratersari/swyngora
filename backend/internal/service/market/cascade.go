package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationCascade scores short-burst long/short liquidations for one coin
// (or symbol=all for the pooled market) on Binance and Bybit separately.
func (s *Service) GetLiquidationCascade(ctx context.Context, exchange, symbol string) (*domain.CascadeReport, error) {
	_ = ctx
	sym, err := domain.ParseLiquidationLevelsSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseLiquidationExchange(exchange)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var ev []domain.LiquidationEvent
	if s.liq != nil {
		q := sym
		if q == "all" {
			q = ""
		}
		ev = s.liq.EventsSince(ex, q, now.Add(-6*time.Hour))
	}
	if sym != "all" && s.liqWatch != nil {
		s.liqWatch.Watch(sym)
		s.noteFutures(sym)
	}
	rep := domain.BuildCascadeReport(sym, ex, ev, now)
	return &rep, nil
}

// ScanLiquidationCascades scores the whole market and lists bursting coins.
func (s *Service) ScanLiquidationCascades(ctx context.Context, exchange string) (*domain.CascadeScan, error) {
	_ = ctx
	ex, err := domain.ParseLiquidationExchange(exchange)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var ev []domain.LiquidationEvent
	if s.liq != nil {
		ev = s.liq.EventsSince(ex, "", now.Add(-6*time.Hour))
	}
	scan := domain.BuildCascadeScan(ex, ev, now)
	return &scan, nil
}
