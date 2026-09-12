package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetForensics returns the around-the-move tape for a chosen time plus
// similar past setups (fail-soft: tape still returns if similar fails).
func (s *Service) GetForensics(ctx context.Context, exchange, symbol, window, during string, at time.Time, lookback, interval, direction string, minReturnPct float64, limit int, fields, weights, minCoverage, horizons string) (*domain.ForensicsReport, error) {
	tape, err := s.GetAround(ctx, exchange, symbol, window, during, at)
	if err != nil {
		return nil, err
	}
	out := &domain.ForensicsReport{
		Symbol: tape.Symbol, Exchange: tape.Exchange, At: tape.At,
		Window: tape.Window, During: tape.During, AsOf: tape.AsOf,
		Tape: tape, Note: domain.ForensicsDisclaimer,
	}
	sim, err := s.GetAroundSimilar(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, limit, window, during, fields, weights, minCoverage, horizons, at)
	if err != nil {
		out.SimilarError = err.Error()
	} else {
		out.Similar = sim
	}
	out.Why = domain.ExplainForensicsWhy(*out)
	out.Timeline = domain.BuildForensicsTimeline(*out)
	out.Summary = domain.ExplainForensics(*out)
	return out, nil
}
