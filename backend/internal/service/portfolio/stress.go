package portfolio

import (
	"context"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Stress reprices the open book for a list of asset percent moves. Nothing is saved.
func (s *Service) Stress(ctx context.Context, clientID, portfolioID, ownerID string, moves []domain.StressMove) (*domain.StressReport, error) {
	view, err := s.View(ctx, clientID, portfolioID, ownerID)
	if err != nil {
		return nil, err
	}
	rep, err := domain.StressPortfolio(*view, moves)
	if err != nil {
		return nil, err
	}
	return &rep, nil
}
