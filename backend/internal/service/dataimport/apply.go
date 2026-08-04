package dataimport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *Service) runJob(ctx context.Context, job *domain.ImportJob) error {
	canceled := func() bool {
		st, err := s.store.GetStatus(ctx, job.ID)
		return err == nil && st == domain.ImportCanceled
	}
	if canceled() {
		return s.store.Finish(ctx, job.ID, domain.ImportCanceled, nil, "", time.Now().UTC())
	}
	pl, err := s.loadPayload(job)
	if err != nil {
		return err
	}
	mode := job.Mode
	if mode == "" {
		mode = domain.ImportModeMerge
	}
	added := map[domain.ExportSection]int{}
	steps := []struct {
		sec domain.ExportSection
		fn  func(context.Context, string, domain.ImportMode, *payload) (int, error)
	}{
		{domain.ExportSectionWatchlist, s.applyWatchlist},
		{domain.ExportSectionShares, s.applyShares},
		{domain.ExportSectionAlerts, s.applyAlerts},
		{domain.ExportSectionBacktests, s.applyBacktests},
	}
	for i, step := range steps {
		if canceled() {
			return s.store.Finish(ctx, job.ID, domain.ImportCanceled, added, "", time.Now().UTC())
		}
		pct := float64(i) / float64(len(steps)) * 100
		_ = s.store.UpdateProgress(ctx, job.ID, pct, string(step.sec))
		n, err := step.fn(ctx, job.ClientID, mode, pl)
		if err != nil {
			return err
		}
		added[step.sec] = n
	}
	if canceled() {
		return s.store.Finish(ctx, job.ID, domain.ImportCanceled, added, "", time.Now().UTC())
	}
	return s.store.Finish(ctx, job.ID, domain.ImportCompleted, added, "", time.Now().UTC())
}

func (s *Service) applyWatchlist(ctx context.Context, clientID string, mode domain.ImportMode, pl *payload) (int, error) {
	if s.data.Watchlist == nil {
		return 0, fmt.Errorf("watchlist store not configured")
	}
	if len(pl.WatchlistItems) == 0 && mode != domain.ImportModeReplace {
		return 0, nil
	}
	uncond := domain.WatchlistUnconditionalVersion
	if mode == domain.ImportModeReplace {
		// Replace entire list with imported items (may be empty).
		if _, err := s.data.Watchlist.Set(ctx, clientID, pl.WatchlistItems, uncond); err != nil {
			return 0, err
		}
		return len(pl.WatchlistItems), nil
	}
	// Merge: Add upserts; count only newly added keys
	before := map[string]struct{}{}
	wl, err := s.data.Watchlist.Get(ctx, clientID)
	if err == nil && wl != nil {
		for _, it := range wl.Items {
			before[string(it.Exchange)+"|"+it.Symbol] = struct{}{}
		}
	}
	n := 0
	for _, it := range pl.WatchlistItems {
		key := string(it.Exchange) + "|" + it.Symbol
		if _, ok := before[key]; ok {
			// still upsert note/added via Add — but do not count as new
			_, _ = s.data.Watchlist.Add(ctx, clientID, it, uncond)
			continue
		}
		if _, err := s.data.Watchlist.Add(ctx, clientID, it, uncond); err != nil {
			return n, err
		}
		before[key] = struct{}{}
		n++
	}
	return n, nil
}

func (s *Service) applyShares(ctx context.Context, clientID string, mode domain.ImportMode, pl *payload) (int, error) {
	if s.data.Watchlist == nil {
		return 0, fmt.Errorf("watchlist store not configured")
	}
	if mode == domain.ImportModeReplace {
		existing, err := s.data.Watchlist.ListSharesByOwner(ctx, clientID)
		if err != nil {
			return 0, err
		}
		for _, sh := range existing {
			_ = s.data.Watchlist.DeleteShare(ctx, clientID, sh.GranteeClientID)
		}
	}
	existing := map[string]struct{}{}
	if mode == domain.ImportModeMerge {
		list, err := s.data.Watchlist.ListSharesByOwner(ctx, clientID)
		if err != nil {
			return 0, err
		}
		for _, sh := range list {
			existing[sh.GranteeClientID] = struct{}{}
		}
	}
	n := 0
	for _, sh := range pl.Shares {
		sh.OwnerClientID = clientID
		if _, ok := existing[sh.GranteeClientID]; ok {
			continue
		}
		if _, err := s.data.Watchlist.CreateShare(ctx, sh); err != nil {
			// skip unique / validation conflicts
			if errors.Is(err, domain.ErrInvalidArgument) {
				continue
			}
			return n, err
		}
		existing[sh.GranteeClientID] = struct{}{}
		n++
	}
	return n, nil
}

func (s *Service) applyAlerts(ctx context.Context, clientID string, mode domain.ImportMode, pl *payload) (int, error) {
	if s.data.Alerts == nil {
		return 0, fmt.Errorf("alerts store not configured")
	}
	if mode == domain.ImportModeReplace {
		list, err := s.data.Alerts.ListByClient(ctx, clientID)
		if err != nil {
			return 0, err
		}
		for _, a := range list {
			_ = s.data.Alerts.Delete(ctx, clientID, a.ID)
		}
	}
	existing := map[string]struct{}{}
	if mode == domain.ImportModeMerge {
		list, err := s.data.Alerts.ListByClient(ctx, clientID)
		if err != nil {
			return 0, err
		}
		for _, a := range list {
			existing[a.ID] = struct{}{}
		}
	}
	n := 0
	for _, a := range pl.Alerts {
		a.ClientID = clientID
		if _, ok := existing[a.ID]; ok {
			continue
		}
		// Cap check soft: skip if at max
		if cnt, err := s.data.Alerts.CountByClient(ctx, clientID); err == nil && cnt >= domain.MaxPriceAlertsPerClient {
			break
		}
		if _, err := s.data.Alerts.Create(ctx, a); err != nil {
			// duplicate id or validation — skip
			continue
		}
		existing[a.ID] = struct{}{}
		n++
	}
	return n, nil
}

func (s *Service) applyBacktests(ctx context.Context, clientID string, mode domain.ImportMode, pl *payload) (int, error) {
	if s.data.Scanner == nil {
		return 0, fmt.Errorf("scanner store not configured")
	}
	if mode == domain.ImportModeReplace {
		// page all backtests and delete
		offset := 0
		for {
			list, err := s.data.Scanner.ListBacktests(ctx, clientID, 100, offset)
			if err != nil {
				return 0, err
			}
			if len(list) == 0 {
				break
			}
			for _, b := range list {
				_ = s.data.Scanner.DeleteBacktest(ctx, clientID, b.ID)
			}
			// after deletes, keep offset 0
			if len(list) < 100 {
				break
			}
		}
	}
	existing := map[string]struct{}{}
	if mode == domain.ImportModeMerge {
		list, err := s.data.Scanner.ListBacktests(ctx, clientID, 500, 0)
		if err != nil {
			return 0, err
		}
		for _, b := range list {
			existing[b.ID] = struct{}{}
		}
	}
	n := 0
	for _, bundle := range pl.Backtests {
		job := bundle.Job
		job.ClientID = clientID
		if _, ok := existing[job.ID]; ok {
			continue
		}
		if cnt, err := s.data.Scanner.CountBacktests(ctx, clientID); err == nil && cnt >= domain.MaxScannerBacktestsPerClient {
			break
		}
		if _, err := s.data.Scanner.CreateBacktest(ctx, job); err != nil {
			continue
		}
		// Restore terminal status if not pending (Create may have set pending fields)
		if job.Status != domain.BacktestPending {
			_ = s.data.Scanner.FinishBacktest(ctx, job.ID, job.Status, job.SignalCount, job.ErrorMessage, time.Now().UTC())
		}
		for _, sig := range bundle.Signals {
			sig.BacktestID = job.ID
			_ = s.data.Scanner.InsertBacktestSignal(ctx, sig)
		}
		existing[job.ID] = struct{}{}
		n++
	}
	return n, nil
}
