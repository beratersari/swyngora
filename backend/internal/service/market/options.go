package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOptions returns the BTC/ETH options desk snapshot (Deribit).
func (s *Service) GetOptions(ctx context.Context, exchange, asset string) (*domain.OptionsSnapshot, error) {
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOptionsExchange(exchange)
	if err != nil {
		return nil, err
	}
	if s == nil || s.options == nil {
		return nil, fmt.Errorf("%w: options source not configured", domain.ErrUpstream)
	}
	quotes, err := s.options.GetOptionQuotes(ctx, asset)
	if err != nil {
		return nil, err
	}
	var (
		dvol    domain.OptionsDVOL
		index   float64
		wg      sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		got, err := s.options.GetOptionsDVOL(ctx, asset)
		if err == nil {
			dvol = got
		}
	}()
	go func() {
		defer wg.Done()
		got, err := s.options.GetOptionsIndex(ctx, asset)
		if err == nil {
			index = got
		}
	}()
	wg.Wait()
	snap := domain.BuildOptionsSnapshot(asset, quotes, index, dvol, time.Now().UTC())
	snap.Exchange = ex
	return snap, nil
}
