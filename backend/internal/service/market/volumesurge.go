package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const volumeSurgeDisclaimer = "Volume surge compares the latest 5-minute, 15-minute, and 1-hour quote volume to that coin's own typical (median of the prior ~24 hours). Buy/sell uses taker-buy when the venue publishes it (Binance klines); Bybit rows are total-only. Scan ranks top 24h-volume USDT pairs by how far above typical they are. Informational only — not financial advice."

// GetVolumeSurge returns current vs typical volume for one coin (5m / 15m / 1h, buy/sell split).
func (s *Service) GetVolumeSurge(ctx context.Context, exchange, symbol string) (*domain.VolumeSurgeReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	out := &domain.VolumeSurgeReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.VolumeSurgeVenue, 0, len(want)),
		Note:     volumeSurgeDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.volumeSurgeOne(ctx, v, symbol)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	out.Summary = domain.ExplainVolumeSurgeReport(*out)
	return out, nil
}

// ScanVolumeSurges ranks top-volume coins by how far above their own typical they are.
func (s *Service) ScanVolumeSurges(ctx context.Context, exchange, quote string, minRatio float64, symbolLimit int) (*domain.VolumeSurgeScan, error) {
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	if quote == "" {
		quote = "USDT"
	}
	if minRatio <= 0 {
		minRatio = domain.DefaultVolumeSurgeMin
	}
	if symbolLimit <= 0 {
		symbolLimit = domain.VolumeSurgeScanDefault
	}
	if symbolLimit > domain.VolumeSurgeScanMax {
		symbolLimit = domain.VolumeSurgeScanMax
	}
	now := time.Now().UTC()
	out := &domain.VolumeSurgeScan{
		Exchange:    ex,
		Quote:       quote,
		MinRatio:    minRatio,
		SymbolLimit: symbolLimit,
		AsOf:        now,
		Hits:        []domain.VolumeSurgeHit{},
		Note:        volumeSurgeDisclaimer,
	}

	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}

	type job struct {
		ex     domain.Exchange
		symbol string
	}
	var jobs []job
	for _, v := range want {
		spot, err := s.ListSpotMarkets(ctx, string(v), domain.SpotListQuery{
			QuoteAsset: quote,
			SortBy:     domain.SpotSortQuoteVolume,
			Order:      domain.SortDesc,
			Limit:      symbolLimit,
		})
		if err != nil {
			if len(want) == 1 {
				return nil, err
			}
			continue
		}
		for _, m := range spot.Items {
			if m.Symbol == "" {
				continue
			}
			jobs = append(jobs, job{ex: v, symbol: m.Symbol})
		}
	}

	var (
		mu   sync.Mutex
		hits []domain.VolumeSurgeHit
		wg   sync.WaitGroup
		sem  = make(chan struct{}, 5)
	)
	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			ven := s.volumeSurgeOne(ctx, j.ex, j.symbol)
			if ven.Error != "" || ven.MaxRatio < minRatio {
				return
			}
			hit := domain.VolumeSurgeHit{
				Symbol: ven.Symbol, Exchange: ven.Exchange, BuySellKnown: ven.BuySellKnown,
				Windows: ven.Windows, MaxRatio: ven.MaxRatio, Hottest: ven.Hottest,
				Summary: ven.Summary,
			}
			for _, w := range ven.Windows {
				if w.Window == ven.Hottest {
					hit.Grade = w.Grade
					hit.Dominant = w.Dominant
					break
				}
			}
			mu.Lock()
			hits = append(hits, hit)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].MaxRatio != hits[j].MaxRatio {
			return hits[i].MaxRatio > hits[j].MaxRatio
		}
		return hits[i].Symbol < hits[j].Symbol
	})
	if len(hits) > domain.MaxVolumeSurgeHits {
		hits = hits[:domain.MaxVolumeSurgeHits]
	}
	out.Hits = hits
	out.Summary = domain.ExplainVolumeSurgeScan(*out)
	return out, nil
}

func (s *Service) volumeSurgeOne(ctx context.Context, ex domain.Exchange, symbol string) domain.VolumeSurgeVenue {
	out := domain.VolumeSurgeVenue{Exchange: ex, Symbol: symbol, Windows: []domain.VolumeSurgeWindow{}}
	candles, err := s.GetCandles(ctx, string(ex), symbol, "5m", domain.VolumeSurgeLookbackBars, nil, nil)
	if err != nil {
		out.Error = err.Error()
		out.Summary = err.Error()
		return out
	}
	bars := domain.VolumeBarsFromCandles(candles)
	return domain.BuildVolumeSurgeVenue(ex, symbol, bars)
}
