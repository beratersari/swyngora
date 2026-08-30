package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationFeed is venue health for liquidation_feed alerts.
func (s *Service) GetLiquidationFeed(exchange string) domain.LiquidationFeed {
	out := domain.LiquidationFeed{Venues: []domain.LiquidationVenueHealth{}, Missing: []string{}}
	ex, err := domain.ParseLiquidationExchange(exchange)
	if err != nil || s == nil || s.liq == nil {
		return out
	}
	return s.liq.Feed(ex)
}

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
		ev = s.liq.EventsSince(ex, q, now.Add(-24*time.Hour))
	}
	if sym != "all" && s.liqWatch != nil {
		s.liqWatch.Watch(sym)
		s.noteFutures(sym)
	}
	rep := domain.BuildCascadeReport(sym, ex, ev, now)
	if sym != "all" {
		s.fillCascadeEpisodePrices(ctx, sym, now, rep.Episodes)
	}
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
		ev = s.liq.EventsSince(ex, "", now.Add(-24*time.Hour))
	}
	scan := domain.BuildCascadeScan(ex, ev, now)
	return &scan, nil
}

func (s *Service) fillCascadeEpisodePrices(ctx context.Context, symbol string, now time.Time, eps []domain.CascadeEpisode) {
	if len(eps) == 0 {
		return
	}
	from := eps[0].StartedAt
	to := now
	for _, ep := range eps {
		if !ep.StartedAt.IsZero() && ep.StartedAt.Before(from) {
			from = ep.StartedAt
		}
		end := ep.EndedAt
		if end.IsZero() {
			end = now
		}
		if end.After(to) {
			to = end
		}
	}
	from = from.Add(-time.Minute)
	bn := s.cascadeCandles(ctx, string(domain.ExchangeBinance), symbol, from, to)
	bb := s.cascadeCandles(ctx, string(domain.ExchangeBybit), symbol, from, to)
	for i := range eps {
		bars := bn
		switch eps[i].Exchange {
		case string(domain.ExchangeBybit):
			if len(bb) > 0 {
				bars = bb
			}
		case domain.CascadeExchangeBoth:
			if len(bn) == 0 {
				bars = bb
			}
		}
		domain.ApplyCascadeCandlePrices(&eps[i], bars, now)
	}
}

func (s *Service) cascadeCandles(ctx context.Context, exchange, symbol string, from, to time.Time) []domain.Candle {
	mins := int(to.Sub(from)/time.Minute) + 4
	if mins < 8 {
		mins = 8
	}
	if mins > 1000 {
		mins = 1000
	}
	bars, err := s.GetCandles(ctx, exchange, symbol, "1m", mins, &from, &to)
	if err != nil || len(bars) == 0 {
		return nil
	}
	return bars
}
