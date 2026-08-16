package market

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	whalesDisclaimer = "Whale prints are clustered aggressive futures fills from the newest ~1000 trades per coin (minutes of tape, not a 24h tape). Buy = taker long; sell = taker short. Liquidations are forced closes from the live book (last hour). Unusual means the size is large versus circulating market cap. Informational only — not financial advice."
	whaleLiqLookback = time.Hour
	whaleScanConc    = 6
)

// GetWhales returns the largest recent aggressive trades and liquidations.
// Empty symbol scans the top liquid USDT pairs (biggest first).
func (s *Service) GetWhales(ctx context.Context, exchange, symbol string, minNotional float64, limit int) (*domain.WhaleReport, error) {
	ex, err := domain.ParseLiquidationExchange(exchange)
	if err != nil {
		return nil, err
	}
	minNotional = domain.ParseWhaleMinNotional(minNotional)
	limit = domain.ParseWhaleLimit(limit)

	symbol = strings.TrimSpace(symbol)
	var symbols []string
	if symbol != "" {
		symbol, err = domain.ValidateOpenInterestSymbol(symbol)
		if err != nil {
			return nil, err
		}
		symbols = []string{symbol}
	} else if scanned, scanErr := s.whaleScanSymbols(ctx, ex); scanErr == nil {
		symbols = scanned
	}

	for _, sy := range symbols {
		s.noteFutures(sy)
		if s.liqWatch != nil {
			s.liqWatch.Watch(sy)
		}
	}

	prints := s.fetchWhalePrints(ctx, ex, symbols)
	events := domain.ClusterWhalePrints(prints, minNotional, 0)

	since := time.Now().UTC().Add(-whaleLiqLookback)
	if s.liq != nil {
		for _, e := range s.liq.RecentLarge(since, minNotional) {
			if symbol != "" && domain.NormalizeLiquidationSymbol(e.Symbol) != symbol {
				continue
			}
			if ex != "all" && string(e.Exchange) != ex {
				continue
			}
			if ev, ok := domain.WhaleFromLiquidation(e, minNotional); ok {
				events = append(events, ev)
			}
		}
	}

	mcapBySym := map[string]float64{}
	for i := range events {
		sy := events[i].Symbol
		if _, ok := mcapBySym[sy]; !ok {
			mcapBySym[sy] = s.whaleMcap(ctx, sy)
		}
		domain.AnnotateWhaleMcap(&events[i], mcapBySym[sy])
	}

	domain.SortWhalesBiggestFirst(events)
	if len(events) > limit {
		events = events[:limit]
	}

	now := time.Now().UTC()
	out := &domain.WhaleReport{
		Symbol:      symbol,
		Exchange:    ex,
		AsOf:        now,
		MinNotional: minNotional,
		Events:      events,
		Summary:     domain.ExplainWhales(events, symbol),
		Note:        whalesDisclaimer,
	}
	return out, nil
}

func (s *Service) fetchWhalePrints(ctx context.Context, exchange string, symbols []string) []domain.TakerPrint {
	venues := whaleVenues(exchange)
	if len(venues) == 0 || len(symbols) == 0 {
		return nil
	}
	var (
		mu  sync.Mutex
		out []domain.TakerPrint
		wg  sync.WaitGroup
		sem = make(chan struct{}, whaleScanConc)
	)
	for _, v := range venues {
		p := s.printPort(v)
		if p == nil {
			continue
		}
		for _, sy := range symbols {
			sy, p := sy, p
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				case sem <- struct{}{}:
				}
				defer func() { <-sem }()
				got, err := p.GetRecentPrints(ctx, sy)
				if err != nil || len(got) == 0 {
					return
				}
				mu.Lock()
				out = append(out, got...)
				mu.Unlock()
			}()
		}
	}
	wg.Wait()
	return out
}

func (s *Service) whaleScanSymbols(ctx context.Context, exchange string) ([]string, error) {
	catalog := domain.ExchangeBinance
	if exchange == string(domain.ExchangeBybit) {
		catalog = domain.ExchangeBybit
	}
	if _, err := s.port(catalog); err != nil && exchange == "all" {
		catalog = domain.ExchangeBybit
	}
	limit := domain.ClampWhaleScanSymbols(0)
	spot, err := s.ListSpotMarkets(ctx, string(catalog), domain.SpotListQuery{
		QuoteAsset: "USDT",
		Status:     "TRADING",
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      limit + 10,
	})
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, limit)
	for _, m := range spot.Items {
		sym, err := domain.ValidateOpenInterestSymbol(m.Symbol)
		if err != nil {
			continue
		}
		base := m.BaseAsset
		if base == "" {
			base, _ = domain.SplitBaseQuote(catalog, sym)
		}
		if !domain.IsBreadthEligible(base) {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		out = append(out, sym)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Service) whaleMcap(ctx context.Context, symbol string) float64 {
	base, _ := domain.SplitBaseQuote(domain.ExchangeBinance, symbol)
	if base == "" {
		return 0
	}
	sup, err := s.GetSupply(ctx, base)
	if err != nil || sup == nil || sup.CirculatingSupply == nil || *sup.CirculatingSupply <= 0 {
		return 0
	}
	last := 0.0
	if tkr, err := s.GetTicker24h(ctx, string(domain.ExchangeBinance), symbol); err == nil && tkr != nil {
		last, _ = strconv.ParseFloat(tkr.LastPrice, 64)
	}
	if last <= 0 && sup.CurrentPriceUSD != nil {
		last = *sup.CurrentPriceUSD
	}
	if last <= 0 {
		return 0
	}
	return *sup.CirculatingSupply * last
}

func (s *Service) printPort(ex domain.Exchange) domain.RecentPrintPort {
	p, err := s.port(ex)
	if err != nil || p == nil {
		return nil
	}
	rp, ok := p.(domain.RecentPrintPort)
	if !ok {
		return nil
	}
	return rp
}

func whaleVenues(exchange string) []domain.Exchange {
	switch exchange {
	case string(domain.ExchangeBinance):
		return []domain.Exchange{domain.ExchangeBinance}
	case string(domain.ExchangeBybit):
		return []domain.Exchange{domain.ExchangeBybit}
	default:
		return []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	}
}
