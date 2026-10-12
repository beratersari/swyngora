package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// attachSituationHeld stamps how long each live clause has held.
// A failed history read leaves that clock blank. The request still returns.
func (s *Service) attachSituationHeld(ctx context.Context, symbol string, now time.Time, sit *domain.CoinSituation, cvd *domain.CVDReport, funding *domain.FundingSnapshot, longs *domain.LongShortSnapshot, prints domain.SituationPrintInput) {
	if s == nil || sit == nil {
		return
	}
	in := domain.SituationHeldInput{
		Now:          now,
		Symbol:       symbol,
		Coverage:     prints.Coverage,
		Prints:       prints.Minutes,
		OpenInterest: s.situationOISnapshots(ctx, symbol, now),
	}
	if cvd != nil && cvd.Combined != nil && cvd.Combined.Complete {
		in.CVD = cvd.Combined.Points
	}
	if funding != nil {
		in.Funding = funding.PressureHistory
	}
	if longs != nil {
		in.Crowd = longs.Venues
	}
	domain.StampSituationHeld(sit, in)
}

func (s *Service) situationOISnapshots(ctx context.Context, symbol string, now time.Time) []domain.FuturesSnapshot {
	if s == nil || s.futHist == nil {
		return nil
	}
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric:   domain.FuturesMetricOpenInterest,
		Exchange: "all",
		Symbol:   symbol,
		From:     now.Add(-26 * time.Hour),
		To:       now,
		Limit:    domain.MaxFuturesHistoryLimit,
	})
	if err != nil {
		return nil
	}
	return futuresSnapshots(raw)
}

// situationPrintMinutes loads 48h of stored print buckets plus book prints
// newer than the archive. Without the archive the clock stays blank.
func (s *Service) situationPrintMinutes(ctx context.Context, symbol string, now time.Time, latest map[domain.Exchange]time.Time, tail []domain.LiquidationEvent) []domain.SituationPrintMinute {
	if s == nil || s.futHist == nil {
		return nil
	}
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric:   "liquidations",
		Exchange: "all",
		Symbol:   symbol,
		From:     now.Add(-48 * time.Hour),
		To:       now,
		Bucket:   5 * time.Minute,
	})
	if err != nil || raw == nil {
		return nil
	}
	events, ok := raw.([]domain.LiquidationEvent)
	if !ok {
		return nil
	}
	out := append([]domain.LiquidationEvent{}, events...)
	from := now.Add(-48 * time.Hour)
	for _, event := range tail {
		if event.Time.IsZero() || event.Notional <= 0 || event.Time.Before(from) || event.Time.After(now) {
			continue
		}
		if last, found := latest[event.Exchange]; found && !event.Time.After(last) {
			continue
		}
		out = append(out, event)
	}
	return domain.PrintMinutes(out)
}
