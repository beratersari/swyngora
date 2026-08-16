package market

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	heatmapWarmLimit          = 100
	heatmapWarmGap            = 80 * time.Millisecond
	heatmapUniverseEvery      = 2 * time.Minute
	heatmapWarmRequestTimeout = 4 * time.Second
)

// StartHeatmapWarmer REST-samples every live crypto pair so coin-detail
// heatmaps already have history when first opened (no websocket attach).
func (s *Service) StartHeatmapWarmer(ctx context.Context) {
	if s == nil {
		return
	}
	s.refreshHeatmapUniverse(ctx)
	var wg sync.WaitGroup
	for _, ex := range domain.SupportedExchanges {
		if domain.IsEquityExchange(ex) {
			continue
		}
		if _, err := s.port(ex); err != nil {
			continue
		}
		ex := ex
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.warmVenueHeatmaps(ctx, ex)
		}()
	}
	go func() {
		t := time.NewTicker(heatmapUniverseEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.refreshHeatmapUniverse(ctx)
			}
		}
	}()
	<-ctx.Done()
	wg.Wait()
}

func (s *Service) warmVenueHeatmaps(ctx context.Context, ex domain.Exchange) {
	tick := time.NewTicker(heatmapWarmGap)
	defer tick.Stop()
	idx := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			pairs := s.heatmapUniverse(ex)
			if len(pairs) == 0 {
				continue
			}
			if idx >= len(pairs) {
				idx = 0
			}
			sym := pairs[idx]
			idx++
			s.sampleHeatSnapshot(ctx, ex, sym)
		}
	}
}

func (s *Service) sampleHeatSnapshot(ctx context.Context, ex domain.Exchange, symbol string) {
	p, err := s.port(ex)
	if err != nil {
		return
	}
	req, cancel := context.WithTimeout(ctx, heatmapWarmRequestTimeout)
	defer cancel()
	raw, err := p.GetOrderBook(req, domain.OrderBookQuery{
		Symbol:       symbol,
		Limit:        heatmapWarmLimit,
		SnapshotOnly: true,
	})
	if err != nil || raw == nil {
		return
	}
	s.recordHeatFromRaw(ex, symbol, raw, 0)
}

func (s *Service) refreshHeatmapUniverse(ctx context.Context) {
	next := map[domain.Exchange][]string{}
	for _, ex := range domain.SupportedExchanges {
		if domain.IsEquityExchange(ex) {
			continue
		}
		p, err := s.port(ex)
		if err != nil {
			continue
		}
		list, err := p.ListSpotMarkets(ctx)
		if err != nil {
			continue
		}
		type row struct {
			symbol string
			vol    float64
		}
		var rows []row
		for _, m := range list {
			if !heatmapWarmCandidate(m) {
				continue
			}
			vol, _ := strconv.ParseFloat(strings.TrimSpace(m.QuoteVolume), 64)
			rows = append(rows, row{symbol: strings.TrimSpace(m.Symbol), vol: vol})
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].vol > rows[j].vol })
		syms := make([]string, 0, len(rows))
		for _, r := range rows {
			syms = append(syms, r.symbol)
		}
		if len(syms) > 0 {
			next[ex] = syms
		}
	}
	s.watchMu.Lock()
	s.heatUniverse = next
	s.watchMu.Unlock()
}

func (s *Service) heatmapUniverse(ex domain.Exchange) []string {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	out := s.heatUniverse[ex]
	if len(out) == 0 {
		return nil
	}
	cp := make([]string, len(out))
	copy(cp, out)
	return cp
}

func heatmapWarmCandidate(m domain.SpotMarket) bool {
	if strings.TrimSpace(m.Symbol) == "" {
		return false
	}
	st := strings.ToUpper(strings.TrimSpace(m.Status))
	if st != "" && st != "TRADING" && st != "ONLINE" && st != "SPOT" {
		return false
	}
	return true
}
