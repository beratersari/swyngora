package portfolio

import (
	"context"
	"fmt"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SituationLeave splits this book's closed crypto lots by whether each kind stored
// on the buy still matched the sell. Equities are left out. A crypto close whose
// buy stored no read is counted as skipped. A close whose sell stored no read is
// counted as unreadable and stays out of both groups. Open positions are not included.
// The same close can appear on more than one row.
func (s *Service) SituationLeave(ctx context.Context, clientID string, refs ...string) (domain.SituationLeave, error) {
	if s == nil || s.store == nil {
		return domain.SituationLeave{}, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, refs...)
	if err != nil {
		return domain.SituationLeave{}, err
	}
	closes, err := s.store.ListSituationCloses(ctx, p.BookID())
	if err != nil {
		return domain.SituationLeave{}, err
	}
	return domain.BuildSituationLeave(closes), nil
}
