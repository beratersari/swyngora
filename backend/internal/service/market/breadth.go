package market

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const breadthDisclaimer = "Breadth is how many of the most liquid spot pairs we follow are up or down, not a prediction. Volume share uses 24h quote volume as size. Informational only — not financial advice."

// GetBreadth counts how many followed coins are up or down over 1h, 4h, and 24h.
func (s *Service) GetBreadth(ctx context.Context, exchange string, limit int) (*domain.BreadthReport, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	limit = domain.ParseBreadthLimit(limit)
	quote := "USDT"
	if ex == domain.ExchangeCoinbase {
		quote = "USD"
	}
	spot, err := s.ListSpotMarkets(ctx, string(ex), domain.SpotListQuery{
		QuoteAsset: quote,
		Status:     "TRADING",
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      limit + 20, // extra so leveraged tokens can be dropped
	})
	if err != nil {
		return nil, err
	}

	type row struct {
		symbol string
		base   string
		vol    float64
		chg24  float64
		ok24   bool
	}
	seen := map[string]struct{}{}
	rows := make([]row, 0, limit)
	ensure := func(base, symbol string) {
		base = strings.ToUpper(base)
		if _, ok := seen[base]; ok || !domain.IsBreadthEligible(base) {
			return
		}
		if len(rows) >= limit && base != "BTC" && base != "ETH" {
			return
		}
		seen[base] = struct{}{}
		rows = append(rows, row{symbol: symbol, base: base})
	}
	for _, m := range spot.Items {
		base := m.BaseAsset
		if base == "" {
			base, _ = domain.SplitBaseQuote(ex, m.Symbol)
		}
		if !domain.IsBreadthEligible(base) {
			continue
		}
		if _, ok := seen[strings.ToUpper(base)]; ok {
			continue
		}
		if len(rows) >= limit {
			continue
		}
		r := row{symbol: m.Symbol, base: strings.ToUpper(base), vol: parseSpotQty(m.QuoteVolume)}
		r.chg24, r.ok24 = domain.ParseChangePct(m.PriceChangePercent)
		seen[r.base] = struct{}{}
		rows = append(rows, r)
	}
	// Always include BTC and ETH when they exist on the venue.
	btcSym, ethSym := domain.CorrelationRefs(ex, domain.PairSymbol(ex, "SOL", quote))
	if _, ok := seen["BTC"]; !ok {
		ensure("BTC", btcSym)
	}
	if _, ok := seen["ETH"]; !ok {
		ensure("ETH", ethSym)
	}

	symbols := make([]string, 0, len(rows))
	bySym := make(map[string]row, len(rows))
	for _, r := range rows {
		symbols = append(symbols, r.symbol)
		bySym[r.symbol] = r
	}

	chg1h := s.breadthWindowChanges(ctx, ex, domain.BreadthWindow1h, symbols)
	chg4h := s.breadthWindowChanges(ctx, ex, domain.BreadthWindow4h, symbols)

	toMoves := func(chgs map[string]float64, use24 bool) []domain.CoinMove {
		out := make([]domain.CoinMove, 0, len(rows))
		for _, r := range rows {
			m := domain.CoinMove{Symbol: r.symbol, Base: r.base, QuoteVolume: r.vol}
			if use24 {
				m.ChangePct, m.Known = r.chg24, r.ok24
			} else if v, ok := chgs[r.symbol]; ok {
				m.ChangePct, m.Known = v, true
			}
			out = append(out, m)
		}
		return out
	}

	now := time.Now().UTC()
	out := &domain.BreadthReport{
		Exchange: string(ex),
		Quote:    quote,
		Universe: len(rows),
		AsOf:     now,
		Windows: []domain.BreadthWindow{
			domain.BuildBreadthWindow(domain.BreadthWindow1h, toMoves(chg1h, false)),
			domain.BuildBreadthWindow(domain.BreadthWindow4h, toMoves(chg4h, false)),
			domain.BuildBreadthWindow(domain.BreadthWindow24h, toMoves(nil, true)),
		},
		Note: breadthDisclaimer,
	}
	out.Summary = domain.ExplainBreadthReport(out.Windows)
	return out, nil
}

func (s *Service) breadthWindowChanges(ctx context.Context, ex domain.Exchange, window string, symbols []string) map[string]float64 {
	if p := s.windowPort(ex); p != nil {
		got, err := p.GetWindowChanges(ctx, window, symbols)
		if err == nil && len(got) > 0 {
			out := make(map[string]float64, len(got))
			for _, w := range got {
				out[w.Symbol] = w.ChangePct
			}
			return out
		}
	}
	return s.breadthFromCandles(ctx, string(ex), window, symbols)
}

func (s *Service) windowPort(ex domain.Exchange) domain.WindowChangePort {
	if s == nil || s.windows == nil {
		return nil
	}
	return s.windows[ex]
}

func (s *Service) breadthFromCandles(ctx context.Context, exchange, window string, symbols []string) map[string]float64 {
	interval, limit := "15m", 6
	if window == domain.BreadthWindow4h {
		interval, limit = "1h", 6
	} else if window == domain.BreadthWindow24h {
		interval, limit = "1h", 26
	}
	var (
		mu  sync.Mutex
		out = map[string]float64{}
		wg  sync.WaitGroup
		sem = make(chan struct{}, 6)
	)
	for _, sym := range symbols {
		sym := sym
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			candles, err := s.GetCandles(ctx, exchange, sym, interval, limit, nil, nil)
			if err != nil {
				return
			}
			pts := domain.PricePointsFromCandles(candles)
			if len(pts) < 2 {
				return
			}
			closes := make([]float64, len(pts))
			for i, p := range pts {
				closes[i] = p.Close
			}
			mu.Lock()
			out[sym] = domain.MovePct(closes)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func parseSpotQty(raw string) float64 {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
