package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// PumpQuery is the application input for single-symbol pump detection.
type PumpQuery struct {
	Exchange       string
	Symbol         string
	Interval       string
	LookbackHours  float64 // if >0, derive candle limit from interval
	Limit          int     // explicit bars (used when LookbackHours==0); default 100
	StartTime      *time.Time
	EndTime        *time.Time
	MinReturnPct   float64
	WindowBars     int
	Mode           domain.PumpDetectMode
	Direction      domain.PumpDirection
	MinVolumeRatio float64
	MaxEvents      int
}

// PumpScanQuery scans top spot symbols for recent pump events.
type PumpScanQuery struct {
	Exchange       string
	QuoteAsset     string // e.g. USDT
	Interval       string
	LookbackHours  float64
	LimitBars      int
	MinReturnPct   float64
	WindowBars     int
	Mode           domain.PumpDetectMode
	Direction      domain.PumpDirection
	MinVolumeRatio float64
	// SymbolLimit caps how many top-volume symbols to scan (default 15, max 40).
	SymbolLimit int
	// MaxEventsPerSymbol caps events kept per symbol (default 3).
	MaxEventsPerSymbol int
	// MaxTotalEvents caps aggregate result size (default 30).
	MaxTotalEvents int
}

// PumpDetectResult is the service response for one symbol.
type PumpDetectResult struct {
	Exchange      domain.Exchange
	Symbol        string
	Interval      string
	LookbackHours float64
	BarsAnalyzed  int
	MinReturnPct  float64
	WindowBars    int
	Mode          domain.PumpDetectMode
	Direction     domain.PumpDirection
	Events        []domain.PumpEvent
	Note          string
}

// PumpScanHit is one symbol's strongest recent pump in a scan.
type PumpScanHit struct {
	Symbol       string
	Exchange     domain.Exchange
	Interval     string
	Events       []domain.PumpEvent
	BestReturnPct float64
}

// DetectPumpEvents loads candles and runs domain pump detection.
func (s *Service) DetectPumpEvents(ctx context.Context, q PumpQuery) (*PumpDetectResult, error) {
	ex, err := s.ResolveExchange(q.Exchange)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(q.Interval)
	if interval == "" {
		interval = "1h"
	}
	if !domain.IsValidIntervalFor(ex, interval) {
		return nil, fmt.Errorf("%w: interval must be one of %v for %s", domain.ErrInvalidArgument, domain.SupportedIntervalsFor(ex), ex)
	}
	if q.MinReturnPct <= 0 {
		q.MinReturnPct = 5
	}
	if q.WindowBars <= 0 {
		q.WindowBars = 1
	}
	if q.Mode == "" {
		q.Mode = domain.PumpModeCloseReturn
	}
	if q.Direction == "" {
		q.Direction = domain.PumpDirectionUp
	}
	if q.MaxEvents <= 0 {
		q.MaxEvents = 20
	}

	limit := q.Limit
	if q.LookbackHours > 0 {
		n, err := domain.BarsForLookbackHours(domain.CandleInterval(interval), q.LookbackHours)
		if err != nil {
			return nil, err
		}
		limit = n
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	candles, err := s.GetCandles(ctx, string(ex), q.Symbol, interval, limit, q.StartTime, q.EndTime)
	if err != nil {
		return nil, err
	}
	events, err := domain.DetectPumpEvents(candles, domain.PumpDetectOptions{
		MinReturnPct:   q.MinReturnPct,
		WindowBars:     q.WindowBars,
		Mode:           q.Mode,
		Direction:      q.Direction,
		MinVolumeRatio: q.MinVolumeRatio,
		MaxEvents:      q.MaxEvents,
	})
	if err != nil {
		return nil, err
	}

	return &PumpDetectResult{
		Exchange:      ex,
		Symbol:        normalizeSymbolForExchange(ex, q.Symbol),
		Interval:      interval,
		LookbackHours: q.LookbackHours,
		BarsAnalyzed:  len(candles),
		MinReturnPct:  q.MinReturnPct,
		WindowBars:    q.WindowBars,
		Mode:          q.Mode,
		Direction:     q.Direction,
		Events:        events,
		Note:          "Informational only — not financial advice. Pump detection is mechanical threshold matching, not a trade signal.",
	}, nil
}

// ScanPumpEvents scans top quote-volume symbols for pumps (rate-limited concurrency).
func (s *Service) ScanPumpEvents(ctx context.Context, q PumpScanQuery) ([]PumpScanHit, error) {
	ex, err := s.ResolveExchange(q.Exchange)
	if err != nil {
		return nil, err
	}
	if q.Interval == "" {
		q.Interval = "15m"
	}
	if q.LookbackHours <= 0 {
		q.LookbackHours = 24
	}
	if q.MinReturnPct <= 0 {
		q.MinReturnPct = 8
	}
	if q.WindowBars <= 0 {
		q.WindowBars = 1
	}
	if q.Mode == "" {
		q.Mode = domain.PumpModeCloseReturn
	}
	if q.Direction == "" {
		q.Direction = domain.PumpDirectionUp
	}
	if q.SymbolLimit <= 0 {
		q.SymbolLimit = 15
	}
	if q.SymbolLimit > 40 {
		q.SymbolLimit = 40
	}
	if q.MaxEventsPerSymbol <= 0 {
		q.MaxEventsPerSymbol = 3
	}
	if q.MaxTotalEvents <= 0 {
		q.MaxTotalEvents = 30
	}
	quote := strings.TrimSpace(q.QuoteAsset)
	if quote == "" {
		if ex == domain.ExchangeCoinbase {
			quote = "USD"
		} else {
			quote = "USDT"
		}
	}

	spot, err := s.ListSpotMarkets(ctx, string(ex), domain.SpotListQuery{
		QuoteAsset: quote,
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      q.SymbolLimit,
		Offset:     0,
	})
	if err != nil {
		return nil, err
	}

	type job struct {
		symbol string
	}
	jobs := make([]job, 0, len(spot.Items))
	for _, m := range spot.Items {
		if m.Symbol == "" {
			continue
		}
		jobs = append(jobs, job{symbol: m.Symbol})
	}

	var (
		mu   sync.Mutex
		hits []PumpScanHit
		wg   sync.WaitGroup
		sem  = make(chan struct{}, 5) // limit concurrent exchange calls
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

			res, err := s.DetectPumpEvents(ctx, PumpQuery{
				Exchange:       string(ex),
				Symbol:         j.symbol,
				Interval:       q.Interval,
				LookbackHours:  q.LookbackHours,
				Limit:          q.LimitBars,
				MinReturnPct:   q.MinReturnPct,
				WindowBars:     q.WindowBars,
				Mode:           q.Mode,
				Direction:      q.Direction,
				MinVolumeRatio: q.MinVolumeRatio,
				MaxEvents:      q.MaxEventsPerSymbol,
			})
			if err != nil || res == nil || len(res.Events) == 0 {
				return
			}
			best := res.Events[0].ReturnPct
			mu.Lock()
			hits = append(hits, PumpScanHit{
				Symbol:        res.Symbol,
				Exchange:      ex,
				Interval:      q.Interval,
				Events:        res.Events,
				BestReturnPct: best,
			})
			mu.Unlock()
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return hits, err
	}

	sort.Slice(hits, func(i, j int) bool {
		ai, aj := hits[i].BestReturnPct, hits[j].BestReturnPct
		if ai < 0 {
			ai = -ai
		}
		if aj < 0 {
			aj = -aj
		}
		return ai > aj
	})
	if len(hits) > q.MaxTotalEvents {
		hits = hits[:q.MaxTotalEvents]
	}
	return hits, nil
}

// FormatPumpReturn is a helper for tests/DTOs.
func FormatPumpReturn(p float64) string {
	return strconv.FormatFloat(p, 'f', 4, 64)
}
