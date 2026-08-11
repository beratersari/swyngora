package account

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// DataPurgeDeps holds stores that own client-scoped data.
type DataPurgeDeps struct {
	Watchlist domain.WatchlistPort
	Alerts    domain.PriceAlertPort
	Scanner   domain.ScannerPort
	Exports   domain.ExportPort
	Imports   domain.ImportPort
	APIKeys   domain.APIKeyPort
}

// Service manages account close, reopen, and grace purges.
type Service struct {
	store domain.AccountPort
	data  DataPurgeDeps
	grace time.Duration
	now   func() time.Time
}

// New constructs an account service.
func New(store domain.AccountPort, data DataPurgeDeps) *Service {
	return &Service{
		store: store,
		data:  data,
		grace: domain.AccountCloseGrace,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// WithGrace overrides the 7-day grace (tests).
func (s *Service) WithGrace(d time.Duration) *Service {
	if d > 0 {
		s.grace = d
	}
	return s
}

// Status returns account state; missing row means active (never closed).
func (s *Service) Status(ctx context.Context, clientID string) (*domain.Account, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: account store not configured", domain.ErrUpstream)
	}
	a, err := s.store.Get(ctx, clientID)
	if err == domain.ErrNotFound {
		now := s.now()
		return &domain.Account{
			ClientID: clientID, Status: domain.AccountActive,
			CreatedAt: now, UpdatedAt: now,
		}, nil
	}
	return a, err
}

// IsClosed reports whether clientID is closed (blocks API access).
func (s *Service) IsClosed(ctx context.Context, clientID string) (bool, *domain.Account, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return false, nil, nil
	}
	a, err := s.Status(ctx, clientID)
	if err != nil {
		return false, nil, err
	}
	return a.IsClosed(), a, nil
}

// RequireActive returns ErrAccountClosed if the client is closed.
func (s *Service) RequireActive(ctx context.Context, clientID string) error {
	closed, a, err := s.IsClosed(ctx, clientID)
	if err != nil {
		return err
	}
	if closed {
		return &domain.ErrAccountClosed{ClientID: clientID, PurgeAt: a.PurgeAt}
	}
	return nil
}

// Close closes the account. Data is retained until PurgeAt (now + grace).
func (s *Service) Close(ctx context.Context, clientID string) (*domain.Account, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: account store not configured", domain.ErrUpstream)
	}
	now := s.now()
	// Idempotent if already closed
	if a, err := s.store.Get(ctx, clientID); err == nil && a.IsClosed() {
		return a, nil
	}
	// Cancel active export/import jobs so they stop running while closed.
	s.cancelJobs(ctx, clientID)
	purgeAt := now.Add(s.grace)
	return s.store.Close(ctx, clientID, now, purgeAt)
}

func (s *Service) cancelJobs(ctx context.Context, clientID string) {
	if s.data.Exports != nil {
		if list, err := s.data.Exports.ListByClient(ctx, clientID, 100, 0); err == nil {
			for _, j := range list {
				if j.IsActive() {
					_, _ = s.data.Exports.Cancel(ctx, clientID, j.ID, s.now())
				}
			}
		}
	}
	if s.data.Imports != nil {
		if list, err := s.data.Imports.ListByClient(ctx, clientID, 100, 0); err == nil {
			for _, j := range list {
				if j.IsActiveApply() || j.Status == domain.ImportPreviewed {
					_, _ = s.data.Imports.Cancel(ctx, clientID, j.ID, s.now())
				}
			}
		}
	}
	if s.data.Scanner != nil {
		if list, err := s.data.Scanner.ListBacktests(ctx, clientID, 100, 0); err == nil {
			for _, b := range list {
				if b.Status == domain.BacktestPending || b.Status == domain.BacktestRunning {
					_, _ = s.data.Scanner.CancelBacktest(ctx, clientID, b.ID, s.now())
				}
			}
		}
	}
}

// Reopen restores access within the grace period.
func (s *Service) Reopen(ctx context.Context, clientID string) (*domain.Account, error) {
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: account store not configured", domain.ErrUpstream)
	}
	a, err := s.store.Get(ctx, clientID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, fmt.Errorf("%w: account is not closed", domain.ErrInvalidArgument)
		}
		return nil, err
	}
	if a.Status == domain.AccountActive {
		return a, nil
	}
	if a.Status == domain.AccountPurged {
		return nil, fmt.Errorf("%w: account data has been permanently deleted", domain.ErrInvalidArgument)
	}
	if !a.CanReopen(s.now()) {
		return nil, fmt.Errorf("%w: reopen window expired; account is pending purge", domain.ErrInvalidArgument)
	}
	return s.store.Reopen(ctx, clientID, s.now())
}

// PurgeDue deletes data for accounts past grace and removes the account row.
func (s *Service) PurgeDue(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	list, err := s.store.ListDueForPurge(ctx, s.now(), 50)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range list {
		if err := s.purgeOne(ctx, list[i].ClientID); err != nil {
			continue
		}
		n++
	}
	return n, nil
}

func (s *Service) purgeOne(ctx context.Context, clientID string) error {
	// Files first from export/import job metadata
	if s.data.Exports != nil {
		jobs, err := s.data.Exports.PurgeClient(ctx, clientID)
		if err != nil {
			return err
		}
		for _, j := range jobs {
			if j.FilePath != "" {
				_ = os.Remove(j.FilePath)
			}
		}
	}
	if s.data.Imports != nil {
		jobs, err := s.data.Imports.PurgeClient(ctx, clientID)
		if err != nil {
			return err
		}
		for _, j := range jobs {
			if j.FilePath != "" {
				_ = os.Remove(j.FilePath)
			}
			if j.PayloadPath != "" {
				_ = os.Remove(j.PayloadPath)
			}
		}
	}
	if s.data.Watchlist != nil {
		if err := s.data.Watchlist.PurgeClient(ctx, clientID); err != nil {
			return err
		}
	}
	if s.data.Alerts != nil {
		if err := s.data.Alerts.PurgeClient(ctx, clientID); err != nil {
			return err
		}
	}
	if s.data.Scanner != nil {
		if err := s.data.Scanner.PurgeClient(ctx, clientID); err != nil {
			return err
		}
	}
	if s.data.APIKeys != nil {
		if err := s.data.APIKeys.DeleteAPIKeysByClient(ctx, clientID); err != nil {
			return err
		}
	}
	_ = s.store.MarkPurged(ctx, clientID, s.now())
	_ = s.store.Delete(ctx, clientID)
	return nil
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
}
