package portfolio

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// requireAccessErr is requireAccess without returning the role.
func (s *Service) requireAccessErr(ctx context.Context, actor string, minRole domain.PortfolioShareRole, refs ...string) (*domain.Portfolio, error) {
	p, _, err := s.requireAccess(ctx, actor, minRole, refs...)
	return p, err
}

func bookRefs(refs []string) (portfolioID, ownerHint string) {
	if len(refs) > 0 {
		portfolioID = strings.TrimSpace(refs[0])
	}
	if len(refs) > 1 {
		ownerHint = strings.TrimSpace(refs[1])
	}
	return
}

// requireAccess loads a book the actor may use at minRole (viewer < trader < owner).
// refs[0] = portfolioId (id or name); refs[1] = optional ownerClientId for someone else's book.
func (s *Service) requireAccess(ctx context.Context, actor string, minRole domain.PortfolioShareRole, refs ...string) (*domain.Portfolio, domain.PortfolioShareRole, error) {
	if s.store == nil {
		return nil, "", fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	actor, err := normalizeClientID(actor)
	if err != nil {
		return nil, "", err
	}
	bookRef, ownerHint := bookRefs(refs)
	if ownerHint != "" && ownerHint != actor {
		owner, err := normalizeClientID(ownerHint)
		if err != nil {
			return nil, "", err
		}
		p, err := s.requireBookOwned(ctx, owner, bookRef)
		if err != nil {
			return nil, "", err
		}
		return s.sharedOrDeny(ctx, actor, p, minRole)
	}
	if bookRef == "" {
		list, err := s.store.ListPortfolios(ctx, actor)
		if err != nil {
			return nil, "", err
		}
		switch len(list) {
		case 0:
			return nil, "", domain.ErrNotFound
		case 1:
			if !domain.RoleAtLeast(domain.PortfolioRoleOwner, minRole) {
				return nil, "", fmt.Errorf("%w: insufficient permission", domain.ErrForbidden)
			}
			return &list[0], domain.PortfolioRoleOwner, nil
		default:
			return nil, "", fmt.Errorf("%w: portfolioId is required when more than one paper portfolio exists", domain.ErrInvalidArgument)
		}
	}
	list, err := s.store.ListPortfolios(ctx, actor)
	if err != nil {
		return nil, "", err
	}
	for i := range list {
		if list[i].ID == bookRef || strings.EqualFold(list[i].Name, bookRef) {
			if !domain.RoleAtLeast(domain.PortfolioRoleOwner, minRole) {
				return nil, "", fmt.Errorf("%w: insufficient permission", domain.ErrForbidden)
			}
			return &list[i], domain.PortfolioRoleOwner, nil
		}
	}
	p, err := s.store.GetPortfolio(ctx, bookRef)
	if err == nil && p != nil {
		if p.ClientID == actor {
			if !domain.RoleAtLeast(domain.PortfolioRoleOwner, minRole) {
				return nil, "", fmt.Errorf("%w: insufficient permission", domain.ErrForbidden)
			}
			return p, domain.PortfolioRoleOwner, nil
		}
		return s.sharedOrDeny(ctx, actor, p, minRole)
	}
	if err != nil && err != domain.ErrNotFound {
		return nil, "", err
	}
	incoming, err := s.store.ListPortfolioSharesForGrantee(ctx, actor)
	if err != nil {
		return nil, "", err
	}
	for _, sh := range incoming {
		bp, berr := s.store.GetPortfolio(ctx, sh.PortfolioID)
		if berr != nil || bp == nil {
			continue
		}
		if strings.EqualFold(bp.Name, bookRef) {
			if !domain.RoleAtLeast(sh.Role, minRole) {
				return nil, "", fmt.Errorf("%w: insufficient permission", domain.ErrForbidden)
			}
			return bp, sh.Role, nil
		}
	}
	return nil, "", domain.ErrNotFound
}

func (s *Service) requireBookOwned(ctx context.Context, owner, bookRef string) (*domain.Portfolio, error) {
	list, err := s.store.ListPortfolios(ctx, owner)
	if err != nil {
		return nil, err
	}
	if bookRef == "" {
		if len(list) == 0 {
			return nil, domain.ErrNotFound
		}
		if len(list) > 1 {
			return nil, fmt.Errorf("%w: portfolioId is required when more than one paper portfolio exists", domain.ErrInvalidArgument)
		}
		return &list[0], nil
	}
	for i := range list {
		if list[i].ID == bookRef || strings.EqualFold(list[i].Name, bookRef) {
			return &list[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *Service) sharedOrDeny(ctx context.Context, actor string, p *domain.Portfolio, minRole domain.PortfolioShareRole) (*domain.Portfolio, domain.PortfolioShareRole, error) {
	sh, err := s.store.GetPortfolioShare(ctx, p.BookID(), actor)
	if err == domain.ErrNotFound {
		return nil, "", domain.ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if !domain.RoleAtLeast(sh.Role, minRole) {
		return nil, "", fmt.Errorf("%w: insufficient permission", domain.ErrForbidden)
	}
	return p, sh.Role, nil
}
