package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const aroundMovesDisclaimer = "Finds the strongest up and down legs in recent spot candles, then attaches the around-the-move tape (price, volume, VWAP, vs typical, POC, sweeps, stored book/futures) for each one. Legs are same-direction stretches; overlapping quiet bars stay in the current leg. Ranked by |return|. Informational only — not financial advice."

const aroundMovesWorkers = 3

// FindAroundMoves finds strong up/down stretches and shows what happened around each.
func (s *Service) FindAroundMoves(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (*domain.AroundMovesReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	lookID, lookDur, err := domain.ParseAroundLookback(lookback)
	if err != nil {
		return nil, err
	}
	ivID, _, err := domain.ParseAroundMovesInterval(interval)
	if err != nil {
		return nil, err
	}
	dir, err := domain.ParseAroundMovesDirection(direction)
	if err != nil {
		return nil, err
	}
	if minReturnPct < 0 {
		return nil, fmt.Errorf("%w: minReturnPct must be >= 0", domain.ErrInvalidArgument)
	}
	if minReturnPct == 0 {
		minReturnPct = domain.DefaultAroundMovesMinPct(ivID)
	}
	limit = domain.ClampAroundMovesLimit(limit)
	if window == "" {
		window = domain.DefaultAroundWindow
	}
	now := time.Now().UTC()
	from := now.Add(-lookDur)
	spotEx := domain.ExchangeBinance
	if ex != "all" {
		spotEx = domain.Exchange(ex)
	}
	if _, err := s.port(spotEx); err != nil {
		if ex == "all" {
			spotEx = domain.ExchangeBybit
		}
		if _, err2 := s.port(spotEx); err2 != nil {
			return nil, err
		}
	}

	candles, err := s.GetCandles(ctx, string(spotEx), symbol, ivID, 700, &from, &now)
	if err != nil {
		return nil, err
	}
	bars := domain.AroundBarsFromCandles(candles)
	found := domain.FindImportantMoves(bars, minReturnPct, dir, limit)

	out := &domain.AroundMovesReport{
		Symbol: symbol, Exchange: ex, Lookback: lookID, Interval: ivID,
		Direction: dir, MinReturnPct: minReturnPct,
		From: from, To: now, AsOf: now,
		Moves: make([]domain.AroundMoveHit, 0, len(found)),
		Note:  aroundMovesDisclaimer,
	}

	hits := make([]domain.AroundMoveHit, len(found))
	sem := make(chan struct{}, aroundMovesWorkers)
	var wg sync.WaitGroup
	for i, mv := range found {
		hits[i] = domain.AroundMoveHit{AroundMove: mv}
		if !mv.At.IsZero() && now.Sub(mv.At) > domain.MaxAroundLookback {
			continue
		}
		wg.Add(1)
		go func(i int, mv domain.AroundMove) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			useDuring := during
			if useDuring == "" {
				useDuring = mv.During
			}
			rep, err := s.GetAround(ctx, ex, symbol, window, useDuring, mv.At)
			if err != nil || rep == nil {
				return
			}
			hits[i].Around = rep
		}(i, mv)
	}
	wg.Wait()
	out.Moves = hits
	out.Summary = domain.ExplainAroundMovesReport(*out)
	return out, nil
}
