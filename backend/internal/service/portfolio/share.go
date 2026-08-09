package portfolio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SharedBook is a book the caller can access plus their role.
type SharedBook struct {
	Portfolio domain.Portfolio
	Role      domain.PortfolioShareRole
}

// Share grants viewer or trader access to one of the owner's books. Owner only.
func (s *Service) Share(ctx context.Context, ownerClientID, portfolioID, granteeClientID, role string) (*domain.PortfolioShare, error) {
	p, err := s.requireBook(ctx, ownerClientID, portfolioID)
	if err != nil {
		return nil, err
	}
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return nil, err
	}
	if grantee == p.ClientID {
		return nil, fmt.Errorf("%w: cannot share a portfolio with yourself", domain.ErrInvalidArgument)
	}
	r, err := domain.NormalizePortfolioShareRole(role)
	if err != nil {
		return nil, err
	}
	n, err := s.store.CountPortfolioShares(ctx, p.BookID())
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxPortfolioSharesPerBook {
		return nil, fmt.Errorf("%w: at most %d shares per portfolio", domain.ErrInvalidArgument, domain.MaxPortfolioSharesPerBook)
	}
	now := time.Now().UTC()
	return s.store.CreatePortfolioShare(ctx, domain.PortfolioShare{
		PortfolioID: p.BookID(), OwnerClientID: p.ClientID, GranteeClientID: grantee,
		Role: r, CreatedAt: now, UpdatedAt: now,
	})
}

// UpdateShareRole changes an existing share. Owner only.
func (s *Service) UpdateShareRole(ctx context.Context, ownerClientID, portfolioID, granteeClientID, role string) (*domain.PortfolioShare, error) {
	p, err := s.requireBook(ctx, ownerClientID, portfolioID)
	if err != nil {
		return nil, err
	}
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return nil, err
	}
	r, err := domain.NormalizePortfolioShareRole(role)
	if err != nil {
		return nil, err
	}
	return s.store.UpdatePortfolioShareRole(ctx, p.BookID(), grantee, r, time.Now().UTC())
}

// RevokeShare removes access. Owner only.
func (s *Service) RevokeShare(ctx context.Context, ownerClientID, portfolioID, granteeClientID string) error {
	p, err := s.requireBook(ctx, ownerClientID, portfolioID)
	if err != nil {
		return err
	}
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return err
	}
	return s.store.DeletePortfolioShare(ctx, p.BookID(), grantee)
}

// ListShares lists outgoing shares. Empty portfolioID lists every book the owner shared.
func (s *Service) ListShares(ctx context.Context, ownerClientID, portfolioID string) ([]domain.PortfolioShare, error) {
	owner, err := normalizeClientID(ownerClientID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(portfolioID) == "" {
		return s.store.ListPortfolioSharesByOwner(ctx, owner)
	}
	p, err := s.requireBook(ctx, owner, portfolioID)
	if err != nil {
		return nil, err
	}
	return s.store.ListPortfolioSharesByBook(ctx, p.BookID())
}

// ListSharedWithMe lists books shared with the caller.
func (s *Service) ListSharedWithMe(ctx context.Context, granteeClientID string) ([]SharedBook, error) {
	grantee, err := normalizeClientID(granteeClientID)
	if err != nil {
		return nil, err
	}
	shares, err := s.store.ListPortfolioSharesForGrantee(ctx, grantee)
	if err != nil {
		return nil, err
	}
	out := make([]SharedBook, 0, len(shares))
	for _, sh := range shares {
		p, err := s.store.GetPortfolio(ctx, sh.PortfolioID)
		if err != nil {
			continue
		}
		out = append(out, SharedBook{Portfolio: *p, Role: sh.Role})
	}
	return out, nil
}
