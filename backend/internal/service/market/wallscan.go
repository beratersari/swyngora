package market

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	wallScanLimit = 20
	wallScanGap   = 250 * time.Millisecond
	// Near-mid walls fit in the top 100 levels. Binance charges 50 weight for 1,000 levels.
	wallScanDepth          = 100
	binanceWallScanGap     = time.Second
	wallScanUniverseEvery  = 2 * time.Minute
	wallScanRequestTimeout = 4 * time.Second
)

const wallScanNote = "Suspicious means the same wall was added and pulled at least 4 times inside 2 minutes. A flicker is a large wall that rested under about 20 seconds and then left. The desk samples the liquid USDT pairs on Binance, Coinbase, and Bybit while this process runs. Empty until those books have been watched. Not financial advice."

// StartWallScan samples liquid spot books so suspicious walls and brief pulls
// are visible without opening each coin.
func (s *Service) StartWallScan(ctx context.Context) {
	if s == nil {
		return
	}
	s.refreshWallScanUniverse(ctx)
	var wg sync.WaitGroup
	for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit} {
		if _, err := s.port(ex); err != nil {
			continue
		}
		ex := ex
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanVenueWalls(ctx, ex)
		}()
	}
	go func() {
		t := time.NewTicker(wallScanUniverseEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.refreshWallScanUniverse(ctx)
			}
		}
	}()
	<-ctx.Done()
	wg.Wait()
}

func wallScanGapFor(ex domain.Exchange) time.Duration {
	if ex == domain.ExchangeBinance {
		return binanceWallScanGap
	}
	return wallScanGap
}

func (s *Service) scanVenueWalls(ctx context.Context, ex domain.Exchange) {
	tick := time.NewTicker(wallScanGapFor(ex))
	defer tick.Stop()
	idx := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			pairs := s.wallScanList(ex)
			if len(pairs) == 0 {
				continue
			}
			if idx >= len(pairs) {
				idx = 0
			}
			symbol := pairs[idx]
			idx++
			if err := s.sampleWallBook(ctx, ex, symbol); errors.Is(err, domain.ErrRateLimited) {
				if !sleepMarket(ctx, 15*time.Second) {
					return
				}
			}
		}
	}
}

func (s *Service) sampleWallBook(ctx context.Context, ex domain.Exchange, symbol string) error {
	p, err := s.port(ex)
	if err != nil {
		return err
	}
	req, cancel := context.WithTimeout(ctx, wallScanRequestTimeout)
	defer cancel()
	raw, err := p.GetOrderBook(req, domain.OrderBookQuery{
		Symbol:       symbol,
		Limit:        wallScanDepth,
		SnapshotOnly: true,
	})
	if err != nil || raw == nil {
		return err
	}
	an := domain.AnalyzeOrderBook(*raw, domain.DefaultOrderBookRangePct)
	s.observeWalls(ex, symbol, an.Walls)
	return nil
}

func (s *Service) refreshWallScanUniverse(ctx context.Context) {
	if s == nil {
		return
	}
	next := map[domain.Exchange][]string{}
	for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit} {
		if _, err := s.port(ex); err != nil {
			continue
		}
		quote := "USDT"
		if ex == domain.ExchangeCoinbase {
			quote = "USD"
		}
		symbols, err := s.orderCountUniverse(ctx, string(ex), quote, wallScanLimit)
		if err != nil || len(symbols) == 0 {
			if prev := s.wallScanList(ex); len(prev) > 0 {
				next[ex] = prev
			}
			continue
		}
		rows := make([]string, 0, len(symbols))
		for _, symbol := range symbols {
			mapped := domain.CrossVenueSymbol(ex, symbol)
			if mapped == "" {
				continue
			}
			rows = append(rows, mapped)
		}
		if len(rows) > 0 {
			next[ex] = rows
		}
	}
	s.watchMu.Lock()
	s.wallScanPairs = next
	s.watchMu.Unlock()
}

func (s *Service) wallScanList(ex domain.Exchange) []string {
	if s == nil {
		return nil
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	src := s.wallScanPairs[ex]
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func (s *Service) wallScanUniverseCount(exchange string) int {
	if s == nil {
		return 0
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if exchange == "" || exchange == "all" {
		n := 0
		for _, rows := range s.wallScanPairs {
			n += len(rows)
		}
		return n
	}
	return len(s.wallScanPairs[domain.Exchange(exchange)])
}

// ScanSuspiciousWalls returns walls the desk has seen flicker or get pulled.
func (s *Service) ScanSuspiciousWalls(ctx context.Context, exchange, quote string, limit int) (*domain.WallScan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ex, err := domain.ParseWallScanExchange(exchange)
	if err != nil {
		return nil, err
	}
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if quote == "" {
		quote = "USDT"
	}
	now := time.Now().UTC()
	var rows []domain.WallScanHit
	if s != nil && s.walls != nil {
		rows = s.walls.Export(now)
	}
	out := domain.BuildWallScan(ex, quote, s.wallScanUniverseCount(ex), rows, now, limit)
	out.Note = wallScanNote
	return &out, nil
}
