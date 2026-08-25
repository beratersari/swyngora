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

const aroundPrecursorsDisclaimer = "Scans historical candles for strong up and down legs, then compares the tape in the window before each move. Singles are one condition at a time. Combos are conditions that fired together in the same before-window (volume + book + OI, etc.) and say whether that group shows up more before increases or drops. Common on a combo is the overall hit rate (all sampled before-windows), not one side alone — at least 60% of those windows with 3 or more samples. Informational only — not financial advice."

// GetAroundPrecursors finds important moves and what often changed before them.
func (s *Service) GetAroundPrecursors(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (*domain.AroundPrecursorReport, error) {
	if lookback == "" {
		lookback = domain.DefaultAroundPrecursorsLookback
	}
	if limit <= 0 {
		limit = domain.MaxAroundMovesLimit
	}
	moves, err := s.FindAroundMoves(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, limit, window, during)
	if err != nil {
		return nil, err
	}
	out := domain.SummarizeAroundPrecursors(moves.Moves)
	out.Symbol = moves.Symbol
	out.Exchange = moves.Exchange
	out.Lookback = moves.Lookback
	out.Interval = moves.Interval
	out.Direction = moves.Direction
	out.MinReturnPct = moves.MinReturnPct
	out.From = moves.From
	out.To = moves.To
	out.AsOf = moves.AsOf
	out.Note = aroundPrecursorsDisclaimer
	out.Summary = domain.ExplainAroundPrecursors(out)
	return &out, nil
}

const aroundSimilarDisclaimer = "Compares the current tape (last window) to the setup before past important moves. Each past match uses only tape that was already available before that move started (dataFrom–dataTo). fields selects which pieces to use (price, volume, takers, oi, book). weights sets importance (default book and oi 3, takers 1.5, volume and price 1). minCoverage (default 60) drops cases that lack enough selected data from matches into skipped. Uncompared fields have no score. Overlapping matches, shared setup windows, or same-move legs count as one past event (highest similarity kept). afterHorizons is how price usually moved 15m / 1h / 4h after those unique events. Informational only — not financial advice."

// GetAroundSimilar finds past important-move setups that look like the current tape.
func (s *Service) GetAroundSimilar(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights string, minCoverageRaw string) (*domain.AroundSimilarReport, error) {
	if lookback == "" {
		lookback = domain.DefaultAroundPrecursorsLookback
	}
	if window == "" {
		window = domain.DefaultAroundWindow
	}
	want, err := domain.ParseAroundSimilarFields(fields)
	if err != nil {
		return nil, err
	}
	wts, err := domain.ParseAroundSimilarWeights(weights, want)
	if err != nil {
		return nil, err
	}
	minCov, err := domain.ParseAroundSimilarMinCoverage(minCoverageRaw)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	curRep, err := s.GetAround(ctx, exchange, symbol, window, "15m", now.Add(-time.Second))
	if err != nil {
		return nil, err
	}
	current, ok := domain.AroundReportBefore(curRep)
	if !ok {
		return &domain.AroundSimilarReport{
			Symbol: curRep.Symbol, Exchange: curRep.Exchange, Lookback: lookback, Window: window,
			Fields: want.IDs(), Weights: domain.ListAroundSimilarWeights(want, wts),
			MinCoverage: minCov, AsOf: now, Note: aroundSimilarDisclaimer,
			Summary: "Not enough current tape to compare.",
		}, nil
	}
	moves, err := s.FindAroundMoves(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, domain.MaxAroundMovesLimit, window, during)
	if err != nil {
		return nil, err
	}
	hits, skipped := domain.MatchAroundSimilar(current, moves.Moves, limit, want, wts, minCov)
	out := &domain.AroundSimilarReport{
		Symbol: moves.Symbol, Exchange: moves.Exchange, Lookback: moves.Lookback,
		Window: window, Interval: moves.Interval, Fields: want.IDs(),
		Weights: domain.ListAroundSimilarWeights(want, wts), MinCoverage: minCov,
		MinReturnPct: moves.MinReturnPct,
		AsOf:         now, Current: current, Matches: hits, Skipped: skipped, Note: aroundSimilarDisclaimer,
	}
	if len(hits) > 0 {
		out.AfterHorizons = domain.SummarizeAroundSimilarHorizons(hits, s.aroundSimilarHorizonBars(ctx, moves.Exchange, moves.Symbol, hits, now), now)
	}
	domain.FinishAroundSimilar(out)
	return out, nil
}

func (s *Service) aroundSimilarHorizonBars(ctx context.Context, exchange, symbol string, matches []domain.AroundSimilarHit, now time.Time) []domain.AroundBar {
	var earliest, latest time.Time
	for _, m := range matches {
		if m.Move.At.IsZero() {
			continue
		}
		if earliest.IsZero() || m.Move.At.Before(earliest) {
			earliest = m.Move.At
		}
		if latest.IsZero() || m.Move.At.After(latest) {
			latest = m.Move.At
		}
	}
	if earliest.IsZero() {
		return nil
	}
	from := earliest.Add(-30 * time.Minute)
	to := latest.Add(4 * time.Hour)
	if to.After(now) {
		to = now
	}
	spotEx := domain.ExchangeBinance
	if exchange != "" && exchange != "all" {
		spotEx = domain.Exchange(exchange)
	}
	if _, err := s.port(spotEx); err != nil {
		if exchange == "all" || exchange == "" {
			spotEx = domain.ExchangeBybit
		}
		if _, err2 := s.port(spotEx); err2 != nil {
			return nil
		}
	}
	candles, err := s.GetCandles(ctx, string(spotEx), symbol, "15m", 1000, &from, &to)
	if err != nil || len(candles) == 0 {
		return nil
	}
	return domain.AroundBarsFromCandles(candles)
}
