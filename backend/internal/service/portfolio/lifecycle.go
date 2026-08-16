package portfolio

import (
	"context"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ownerClosed is true when the book owner (or the id itself) has a closed account.
func (s *Service) ownerClosed(ctx context.Context, bookOrOwner string) bool {
	if s == nil || s.account == nil {
		return false
	}
	id := strings.TrimSpace(bookOrOwner)
	if id == "" {
		return false
	}
	owner := id
	if s.store != nil {
		if p, err := s.store.GetPortfolio(ctx, id); err == nil && p != nil && strings.TrimSpace(p.ClientID) != "" {
			owner = p.ClientID
		}
	}
	closed, _, err := s.account.IsClosed(ctx, owner)
	return err == nil && closed
}

// FreezeOnClose pauses recurring plans and cancels open paper/margin orders for the owner.
func (s *Service) FreezeOnClose(ctx context.Context, clientID string) error {
	if s == nil || s.store == nil {
		return nil
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}
	books, err := s.store.ListPortfolios(ctx, clientID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range books {
		bookID := books[i].BookID()
		plans, perr := s.store.ListRecurringBuyPlans(ctx, bookID)
		if perr == nil {
			for _, pl := range plans {
				if pl.Status != domain.RecurringBuyActive {
					continue
				}
				_, _ = s.store.UpdateRecurringBuyPlanStatus(ctx, bookID, pl.ID, domain.RecurringBuyPaused, pl.NextRunAt, now)
			}
		}
		_, _ = s.store.CancelOpenPendingOrders(ctx, bookID, "", "", now, domain.CancelReasonAccountClosed)
		orders, oerr := s.store.ListMarginOrders(ctx, bookID, domain.MarginOrderOpen, 200, 0)
		if oerr == nil {
			for _, o := range orders {
				_, _ = s.store.CancelMarginOrder(ctx, bookID, o.ID, now, domain.CancelReasonAccountClosed)
			}
		}
	}
	return nil
}

// PurgeClient deletes every paper book owned by clientID.
func (s *Service) PurgeClient(ctx context.Context, clientID string) error {
	if s == nil || s.store == nil {
		return nil
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}
	books, err := s.store.ListPortfolios(ctx, clientID)
	if err != nil {
		return err
	}
	for i := range books {
		if err := s.store.DeletePortfolio(ctx, clientID, books[i].ID); err != nil {
			return err
		}
	}
	return nil
}
