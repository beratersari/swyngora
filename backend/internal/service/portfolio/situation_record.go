package portfolio

import (
	"context"
	"fmt"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SituationRecord groups this book's closed crypto lots by the read stored on each buy.
// Equities are left out. A crypto close whose buy stored no read is counted as skipped.
// Open positions are not included. The same close can appear on more than one row.
func (s *Service) SituationRecord(ctx context.Context, clientID string, refs ...string) (domain.SituationRecord, error) {
	if s == nil || s.store == nil {
		return domain.SituationRecord{}, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, refs...)
	if err != nil {
		return domain.SituationRecord{}, err
	}
	closes, err := s.store.ListSituationCloses(ctx, p.BookID())
	if err != nil {
		return domain.SituationRecord{}, err
	}
	return domain.BuildSituationRecord(closes), nil
}
