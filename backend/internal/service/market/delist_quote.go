package market

import (
	"context"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *Service) delistEntry(ex domain.Exchange, symbol string) (domain.SpotDelistEntry, bool) {
	if s == nil || s.delist == nil {
		return domain.SpotDelistEntry{}, false
	}
	return s.delist.Get(ex, symbol)
}

// lastDelistTicker returns the last venue print at/before halt when the live
// ticker is gone (pair already removed from the book).
func (s *Service) lastDelistTicker(ctx context.Context, ex domain.Exchange, p domain.MarketDataPort, symbol string) (*domain.Ticker24h, bool) {
	if p == nil {
		return nil, false
	}
	e, ok := s.delistEntry(ex, symbol)
	if !ok || e.DelistTime.IsZero() {
		return nil, false
	}
	key := string(ex) + "|" + strings.ToUpper(symbol) + "|" + e.DelistTime.UTC().Format("20060102150405")
	if s.delistQuote != nil {
		if hit, ok := s.delistQuote.Get(key); ok && hit != nil && hit.LastPrice != "" {
			cp := *hit
			return &cp, true
		}
	}
	bar, ok := lastHaltCandle(ctx, p, symbol, e.DelistTime)
	if !ok {
		return nil, false
	}
	t := domain.TickerFromLastCandle(symbol, bar)
	if s.delistQuote != nil {
		cp := t
		s.delistQuote.Set(key, &cp)
	}
	return &t, true
}

func lastHaltCandle(ctx context.Context, p domain.MarketDataPort, symbol string, halt time.Time) (domain.Candle, bool) {
	end := halt.UTC()
	for _, iv := range []domain.CandleInterval{domain.Interval1d, domain.Interval1h} {
		bars, err := p.GetCandles(ctx, domain.CandleQuery{
			Symbol:   symbol,
			Interval: iv,
			Limit:    5,
			EndTime:  end,
		})
		if err != nil || len(bars) == 0 {
			continue
		}
		return bars[len(bars)-1], true
	}
	return domain.Candle{}, false
}

// hydrateDelistQuotes fills last/high/low/volume on scheduled rows that have
// no live book print (already-halted stubs).
func (s *Service) hydrateDelistQuotes(ctx context.Context, ex domain.Exchange, items []domain.SpotMarket) {
	if s == nil || len(items) == 0 {
		return
	}
	p, err := s.port(ex)
	if err != nil {
		return
	}
	var idxs []int
	for i := range items {
		if items[i].DelistTime == nil || strings.TrimSpace(items[i].LastPrice) != "" {
			continue
		}
		idxs = append(idxs, i)
	}
	if len(idxs) == 0 {
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, i := range idxs {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			tkr, ok := s.lastDelistTicker(ctx, ex, p, items[i].Symbol)
			if !ok || tkr == nil {
				return
			}
			domain.ApplyTickerToSpot(&items[i], *tkr)
			if s.supply != nil {
				base := items[i].BaseAsset
				if base == "" {
					base, _ = inferDelistBaseQuote(ex, items[i].Symbol)
					items[i].BaseAsset = base
				}
				if base != "" {
					if sup, err := s.supply.GetSupply(ctx, base); err == nil {
						applySupplyAndMcap(&items[i], sup)
					}
				}
			}
		}()
	}
	wg.Wait()
}

// enrichDelistMcap attaches circulating supply and mcap to every scheduled
// delist row. Binance snapshot first; CoinGecko public markets for gaps.
func (s *Service) enrichDelistMcap(ctx context.Context, ex domain.Exchange, items []domain.SpotMarket) {
	if s == nil || len(items) == 0 {
		return
	}
	var needFB []string
	seenFB := map[string]struct{}{}
	for i := range items {
		if items[i].DelistTime == nil {
			continue
		}
		if items[i].MarketCapCirculating != nil {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(items[i].BaseAsset))
		if base == "" {
			base, _ = inferDelistBaseQuote(ex, items[i].Symbol)
			items[i].BaseAsset = base
		}
		if base == "" {
			continue
		}
		if s.supply != nil {
			if sup, err := s.supply.GetSupply(ctx, base); err == nil && sup != nil {
				applySupplyAndMcap(&items[i], sup)
				if items[i].MarketCapCirculating != nil {
					continue
				}
			}
		}
		if _, ok := seenFB[base]; !ok {
			seenFB[base] = struct{}{}
			needFB = append(needFB, base)
		}
	}
	if s.delistSupplyFB == nil || len(needFB) == 0 {
		return
	}
	got, err := s.delistSupplyFB.SupplyBySymbols(ctx, needFB)
	if err != nil || len(got) == 0 {
		return
	}
	for i := range items {
		if items[i].DelistTime == nil || items[i].MarketCapCirculating != nil {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(items[i].BaseAsset))
		sup := got[base]
		if sup == nil {
			continue
		}
		applySupplyAndMcap(&items[i], sup)
	}
}

// dropUnquotedDelistStubs removes halt stubs that still have no last print
// after historical kline fill (venue deleted the pair and the candles).
func dropUnquotedDelistStubs(items []domain.SpotMarket) []domain.SpotMarket {
	out := items[:0]
	for _, m := range items {
		if m.DelistTime != nil && strings.TrimSpace(m.LastPrice) == "" {
			continue
		}
		out = append(out, m)
	}
	return out
}
