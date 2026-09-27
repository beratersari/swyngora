package market

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *Service) spreadMemory() *domain.SpreadMemory {
	if s == nil {
		return nil
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if s.spreads == nil {
		s.spreads = domain.NewSpreadMemory()
	}
	return s.spreads
}

func (s *Service) noteSpread(ex domain.Exchange, symbol string, raw *domain.RawOrderBook) {
	if s == nil || raw == nil {
		return
	}
	s.spreadMemory().Observe(time.Now().UTC(), ex, symbol, *raw)
}

// ScanWideSpreads ranks books whose latest spread is wide versus their own typical.
func (s *Service) ScanWideSpreads(ctx context.Context, exchange, quote string, limit int) (*domain.SpreadScan, error) {
	_ = ctx
	ex := strings.ToLower(strings.TrimSpace(exchange))
	if ex == "" || ex == "combined" {
		ex = "all"
	}
	if ex != "all" {
		if _, err := s.ResolveExchange(ex); err != nil {
			return nil, err
		}
	}
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if quote == "" {
		quote = "USDT"
	}
	mem := s.spreadMemory()
	var tracks []domain.SpreadTrack
	if mem != nil {
		tracks = mem.Tracks()
	}
	got := domain.BuildSpreadScan(ex, quote, tracks, limit)
	return &got, nil
}

// GetWideSpread reports one coin's spread versus its own recent typical.
func (s *Service) GetWideSpread(ctx context.Context, exchange, symbol string) (*domain.SpreadReport, error) {
	_ = ctx
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	ex := strings.ToLower(strings.TrimSpace(exchange))
	if ex == "" || ex == "combined" {
		ex = "all"
	}
	if ex != "all" {
		resolved, err := s.ResolveExchange(ex)
		if err != nil {
			return nil, err
		}
		ex = string(resolved)
		symbol = domain.NormalizeSymbol(resolved, symbol)
	}
	mem := s.spreadMemory()
	var tracks []domain.SpreadTrack
	if mem != nil {
		tracks = mem.Tracks()
	}
	got := domain.SpreadReportFor(symbol, ex, tracks)
	return &got, nil
}
