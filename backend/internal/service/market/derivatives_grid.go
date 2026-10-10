package market

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const derivativesGridCacheTTL = 20 * time.Second

// SetDerivativesGrid attaches the archive the time grid reads.
func (s *Service) SetDerivativesGrid(store domain.DerivativesGridStore) {
	if s != nil {
		s.gridStore = store
	}
}

// GetDerivativesGrid returns coins down the side and stored hours or days across.
// A cell stays blank until every expected venue has that column.
func (s *Service) GetDerivativesGrid(ctx context.Context, exchange, metric, rangeID string, limit int) (*domain.DerivativesGrid, error) {
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseDerivativesGrid(metric, rangeID)
	if err != nil {
		return nil, err
	}
	limit = domain.ParseDerivativesBoardLimit(limit)
	if s == nil || s.gridStore == nil {
		return nil, fmt.Errorf("%w: derivatives grid is not configured", domain.ErrUpstream)
	}
	key := ex + "|" + spec.Metric + "|" + spec.Range + "|" + strconv.Itoa(limit)
	if c := s.derivativesGridCache(); c != nil {
		if hit, ok := c.Get(key); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	symbols, err := s.LiquidPerpSymbols(ctx, limit)
	if err != nil && len(symbols) == 0 {
		return nil, err
	}
	now := time.Now().UTC()
	cols := domain.GridColumns(spec, now)
	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: range has no columns", domain.ErrInvalidArgument)
	}
	from := cols[0]
	if spec.Metric == domain.GridMetricOI {
		from = from.Add(-spec.Step)
	}
	in := domain.GridBuildInput{
		Exchange: ex,
		Spec:     spec,
		Now:      now,
		Symbols:  symbols,
	}
	read := domain.GridRead{Symbols: symbols, From: from, To: now, Step: spec.Step}
	switch spec.Metric {
	case domain.GridMetricFunding:
		in.Snapshots, err = s.gridSnapshots(ctx, read, domain.FuturesMetricFunding, domain.FuturesMetricOpenInterest)
	case domain.GridMetricOI:
		in.Snapshots, err = s.gridSnapshots(ctx, read, domain.FuturesMetricOpenInterest)
	case domain.GridMetricCVD:
		in.Taker, err = s.gridStore.ListGridTakerSums(ctx, read)
	case domain.GridMetricLiquidations:
		in.Liquidations, err = s.gridStore.ListGridLiquidations(ctx, read)
		if err != nil {
			return nil, err
		}
		in.Coverage, err = s.gridStore.ListLiquidationCoverage(ctx)
	default:
		return nil, fmt.Errorf("%w: metric %s", domain.ErrInvalidArgument, spec.Metric)
	}
	if err != nil {
		return nil, err
	}
	out := domain.BuildDerivativesGrid(in)
	if c := s.derivativesGridCache(); c != nil && out != nil {
		cp := *out
		c.Set(key, &cp)
	}
	return out, nil
}

func (s *Service) gridSnapshots(ctx context.Context, read domain.GridRead, metrics ...string) ([]domain.GridSnapshot, error) {
	var out []domain.GridSnapshot
	for _, metric := range metrics {
		read.Metric = metric
		rows, err := s.gridStore.ListGridSnapshots(ctx, read)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	return out, nil
}

func (s *Service) derivativesGridCache() *cache.TTL[*domain.DerivativesGrid] {
	if s == nil {
		return nil
	}
	if s.gridCache == nil {
		s.gridCache = cache.New[*domain.DerivativesGrid](derivativesGridCacheTTL)
	}
	return s.gridCache
}
