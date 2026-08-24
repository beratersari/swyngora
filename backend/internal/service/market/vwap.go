package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const vwapDisclaimer = "VWAP is the volume-weighted average price from the start time to now: each candle's typical price (high+low+close)/3 is weighted by quote (USDT) volume. More trading at a price pulls VWAP toward that price. Combined weights Binance and Bybit by their volume. Distance is last price versus VWAP. Informational only — not financial advice."

// GetVWAP returns volume-weighted average price from a start time (or named window) to now.
func (s *Service) GetVWAP(ctx context.Context, exchange, symbol, window string, start, end *time.Time) (*domain.VWAPReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	from, to, windowID, err := domain.ResolveVolumeProfileRange(window, start, end, now)
	if err != nil {
		return nil, err
	}
	interval := domain.ProfileBarInterval(to.Sub(from))
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}

	type fetched struct {
		ex   domain.Exchange
		bars []domain.VolumeProfileBar
		last float64
		err  error
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			bars, last, ferr := s.volumeProfileVenue(ctx, v, symbol, interval, from, to)
			got[i] = fetched{ex: v, bars: bars, last: last, err: ferr}
		}(i, v)
	}
	wg.Wait()

	out := &domain.VWAPReport{
		Symbol:   symbol,
		Exchange: ex,
		Window:   windowID,
		From:     from,
		To:       to,
		AsOf:     now,
		Venues:   make([]domain.VWAPVenue, 0, len(want)),
		Note:     vwapDisclaimer,
	}
	for _, f := range got {
		if f.err != nil && len(f.bars) == 0 {
			out.Venues = append(out.Venues, domain.VWAPVenue{
				Exchange: f.ex, Symbol: symbol, From: from, To: to,
				Interval: string(interval), LastPrice: f.last,
				Error: f.err.Error(), Summary: f.err.Error(),
			})
			continue
		}
		out.Venues = append(out.Venues, domain.ComputeVWAP(f.ex, symbol, f.bars, f.last, from, to, interval))
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombineVWAP(symbol, out.Venues, from, to, interval)
	}
	out.Summary = domain.ExplainVWAPReport(*out)
	return out, nil
}
