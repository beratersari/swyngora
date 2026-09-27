package market

import (
	"context"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const sweepRecoveryNote = "After a liquidity sweep returns, this reads the stored 1-minute spot ladder near the swept level. Before is the resting size on that side just before the poke. The clock starts when price comes back and stops when that size is at least 80% restored, or after two hours. Minutes before the API was sampling are missing. Informational only — not financial advice."

// GetSweepRecovery measures how long spot depth took to refill after recent sweeps.
func (s *Service) GetSweepRecovery(ctx context.Context, exchange, symbol string) (*domain.SweepRecoveryReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	sweeps, err := s.GetLiquiditySweeps(ctx, ex, symbol)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rows := make([]domain.SweepRecovery, 0)
	if sweeps != nil {
		for _, ven := range sweeps.Venues {
			for _, sw := range ven.Sweeps {
				if sw.Status != domain.LiquiditySweepSwept || sw.ReclaimedAt.IsZero() {
					continue
				}
				snaps := s.recoveryLadders(ctx, ven.Exchange, symbol, sw)
				rows = append(rows, domain.MeasureSweepRecovery(string(ven.Exchange), symbol, sw, snaps))
			}
		}
	}
	out := domain.BuildSweepRecoveryReport(symbol, ex, rows, now)
	out.Note = sweepRecoveryNote
	return &out, nil
}

func (s *Service) recoveryLadders(ctx context.Context, ex domain.Exchange, symbol string, sw domain.LiquiditySweep) []domain.BookHistorySnapshot {
	if s == nil || s.bookHist == nil {
		return nil
	}
	from := sw.PiercedAt.Add(-domain.SweepRecoveryLookback)
	to := sw.ReclaimedAt.Add(domain.SweepRecoveryHorizon)
	rows, err := s.bookHist.ListLadders(ctx, domain.BookHistoryQuery{
		Exchange: string(ex),
		Symbol:   strings.ToUpper(symbol),
		From:     from.UTC(),
		To:       to.UTC(),
		Limit:    domain.SweepRecoverySampleLimit,
	})
	if err != nil {
		return nil
	}
	return rows
}
