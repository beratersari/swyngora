package portfolio

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetPerformance returns equity history + period P&L (amount and percent) for a lookback window.
func (s *Service) GetPerformance(ctx context.Context, clientID, periodRaw string, portfolioID ...string) (*domain.PortfolioPerformance, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	bookID := p.BookID()
	period, dur, err := domain.ParsePerformancePeriod(periodRaw)
	if err != nil {
		return nil, err
	}
	view, err := s.View(ctx, p.ClientID, p.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_ = s.recordViewSnapshot(ctx, view, now)

	requestedStart := now.Add(-dur)
	// Young books start at startingBalance; a 15m bucket can sit slightly before CreatedAt.
	var carry *domain.EquitySnapshot
	if !view.CreatedAt.After(requestedStart) {
		carry, err = s.store.LatestEquitySnapshotBefore(ctx, bookID, requestedStart)
		if err != nil {
			return nil, err
		}
	}
	snaps, err := s.store.ListEquitySnapshots(ctx, bookID, requestedStart, now)
	if err != nil {
		return nil, err
	}
	live := domain.EquityPoint{
		Time:           now,
		Equity:         view.Equity,
		CashBalance:    view.CashBalance,
		PositionsValue: view.PositionsValue,
		MarginEquity:   view.MarginEquity,
	}
	out := domain.AssemblePerformance(
		period, requestedStart, now, view.CreatedAt,
		view.StartingBalance, view.Currency, view.ClientID,
		carry, snaps, live, paperNote,
	)
	return &out, nil
}

// SnapshotAll marks every paper portfolio at now (background worker).
func (s *Service) SnapshotAll(ctx context.Context, now time.Time) (int, error) {
	if s.store == nil {
		return 0, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ids, err := s.store.ListPortfolioIDs(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	var firstErr error
	for _, id := range ids {
		book, err := s.store.GetPortfolio(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		view, err := s.View(ctx, book.ClientID, book.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := s.recordViewSnapshot(ctx, view, now); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		n++
	}
	return n, firstErr
}

// PruneSnapshots drops history older than retention.
func (s *Service) PruneSnapshots(ctx context.Context, before time.Time) (int64, error) {
	if s.store == nil {
		return 0, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	return s.store.DeleteEquitySnapshotsBefore(ctx, before.UTC())
}

func (s *Service) recordViewSnapshot(ctx context.Context, view *domain.PortfolioView, now time.Time) error {
	if view == nil {
		return nil
	}
	snapID := view.ID
	if snapID == "" {
		snapID = view.ClientID
	}
	return s.store.UpsertEquitySnapshot(ctx, domain.EquitySnapshot{
		ClientID:       snapID,
		BucketAt:       domain.SnapshotBucket(now, domain.DefaultSnapshotInterval),
		TakenAt:        now,
		Equity:         view.Equity,
		CashBalance:    view.CashBalance,
		PositionsValue: view.PositionsValue,
		MarginEquity:   view.MarginEquity,
		UnrealizedPnL:  view.UnrealizedPnL,
		RealizedPnL:    view.RealizedPnLTotal,
	})
}

func (s *Service) recordCreateSnapshot(ctx context.Context, p *domain.Portfolio) {
	if p == nil {
		return
	}
	at := p.CreatedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_ = s.store.UpsertEquitySnapshot(ctx, domain.EquitySnapshot{
		ClientID:    p.BookID(),
		BucketAt:    domain.SnapshotBucket(at, domain.DefaultSnapshotInterval),
		TakenAt:     at,
		Equity:      p.StartingBalance,
		CashBalance: p.CashBalance,
	})
}
