// Package swing evaluates watchlist (or single-symbol) swing setups.
package swing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

const (
	primaryLimit = 200
	higherLimit  = 220
	maxScan      = 25
)

// Service orchestrates candle fetch + domain.EvaluateSwing.
type Service struct {
	market *market.Service
	watch  *watchlist.Service
}

// New constructs a swing service.
func New(m *market.Service, w *watchlist.Service) *Service {
	return &Service{market: m, watch: w}
}

// Analyze evaluates one symbol (public market data).
func (s *Service) Analyze(ctx context.Context, exchange, symbol string) (*domain.SwingDecision, error) {
	if s == nil || s.market == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	ex, err := s.market.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return s.analyzeResolved(ctx, ex, symbol)
}

// ScanWatchlist evaluates up to maxScan watchlist pairs (optional exchange filter).
func (s *Service) ScanWatchlist(ctx context.Context, clientID, exchange string, limit int) ([]domain.SwingDecision, error) {
	if s == nil || s.market == nil || s.watch == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > maxScan {
		limit = maxScan
	}
	wl, err := s.watch.Get(ctx, clientID, "")
	if err != nil {
		return nil, err
	}
	var items []domain.WatchlistItem
	wantEx := strings.TrimSpace(strings.ToLower(exchange))
	for _, it := range wl.Items {
		if wantEx != "" && string(it.Exchange) != wantEx {
			continue
		}
		items = append(items, it)
		if len(items) >= limit {
			break
		}
	}
	out := make([]domain.SwingDecision, 0, len(items))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for _, it := range items {
		it := it
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			dec, err := s.analyzeResolved(ctx, it.Exchange, it.Symbol)
			if err != nil {
				return
			}
			mu.Lock()
			out = append(out, *dec)
			mu.Unlock()
		}()
	}
	wg.Wait()
	// accepted first, then score desc
	sortDecisions(out)
	return out, ctx.Err()
}

func (s *Service) analyzeResolved(ctx context.Context, ex domain.Exchange, symbol string) (*domain.SwingDecision, error) {
	primary, err := s.market.GetCandles(ctx, string(ex), symbol, string(domain.Interval4h), primaryLimit, nil, nil)
	if err != nil {
		return nil, err
	}
	higher, err := s.market.GetCandles(ctx, string(ex), symbol, string(domain.Interval1d), higherLimit, nil, nil)
	if err != nil {
		return nil, err
	}
	btcSym := "BTCUSDT"
	if ex == domain.ExchangeCoinbase {
		btcSym = "BTC-USD"
	}
	var btcP, btcH []domain.Candle
	if !isBTC(symbol) {
		btcP, _ = s.market.GetCandles(ctx, string(ex), btcSym, string(domain.Interval4h), primaryLimit, nil, nil)
		btcH, _ = s.market.GetCandles(ctx, string(ex), btcSym, string(domain.Interval1d), higherLimit, nil, nil)
	} else {
		btcP, btcH = primary, higher
	}

	pBars, err := domain.ParseOHLC(primary)
	if err != nil {
		return nil, err
	}
	hBars, err := domain.ParseOHLC(higher)
	if err != nil {
		return nil, err
	}
	var btcPB, btcHB []domain.OHLC
	if len(btcP) > 0 {
		btcPB, _ = domain.ParseOHLC(btcP)
	}
	if len(btcH) > 0 {
		btcHB, _ = domain.ParseOHLC(btcH)
	}

	var qv, mcap float64
	if tkr, terr := s.market.GetTicker24h(ctx, string(ex), symbol); terr == nil && tkr != nil {
		qv, _ = strconv.ParseFloat(strings.TrimSpace(tkr.QuoteVolume), 64)
		if qv <= 0 {
			qv, _ = strconv.ParseFloat(strings.TrimSpace(tkr.Volume), 64)
		}
		last, _ := strconv.ParseFloat(strings.TrimSpace(tkr.LastPrice), 64)
		base, _ := domain.SplitBaseQuote(ex, symbol)
		if last > 0 && base != "" {
			if sup, serr := s.market.GetSupply(ctx, base); serr == nil && sup != nil && sup.CirculatingSupply != nil {
				mcap = *sup.CirculatingSupply * last
			}
		}
	}

	return domain.EvaluateSwing(domain.SwingScanInput{
		Exchange:    ex,
		Symbol:      symbol,
		Primary:     pBars,
		Higher:      hBars,
		BTCPrimary:  btcPB,
		BTCHigher:   btcHB,
		QuoteVolume: qv,
		MarketCap:   mcap,
		PrimaryTF:   string(domain.Interval4h),
		HigherTF:    string(domain.Interval1d),
		Now:         time.Now().UTC(),
	})
}

func isBTC(symbol string) bool {
	s := strings.ToUpper(strings.ReplaceAll(symbol, "-", ""))
	return strings.HasPrefix(s, "BTC") && (strings.HasSuffix(s, "USDT") || strings.HasSuffix(s, "USD") || strings.HasSuffix(s, "USDC"))
}

func sortDecisions(in []domain.SwingDecision) {
	for i := 1; i < len(in); i++ {
		for j := i; j > 0 && lessDecision(in[j], in[j-1]); j-- {
			in[j], in[j-1] = in[j-1], in[j]
		}
	}
}

func lessDecision(a, b domain.SwingDecision) bool {
	// sort desc accepted, then score
	if a.Accepted != b.Accepted {
		return a.Accepted && !b.Accepted
	}
	if a.SwingScore != b.SwingScore {
		return a.SwingScore > b.SwingScore
	}
	return a.Symbol < b.Symbol
}
