package market

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const postDelistNoteGecko = "This venue no longer trades this pair. Price and candles below are a public global USD reference after delist — not this exchange. Informational only, not financial advice."
const postDelistNoteVenue = "This venue has delisted the pair. It still trades on %s. Informational only, not financial advice."
const postDelistNoteListed = "This pair is still listed here. Use the main chart for this venue."
const postDelistNoteNone = "No public off-venue price was found after delist."

// WithOffVenuePrice attaches CoinGecko (or similar) last/OHLC for post-delist info.
func (s *Service) WithOffVenuePrice(p domain.OffVenuePricePort) *Service {
	if s != nil {
		s.offVenue = p
	}
	return s
}

// GetPostDelist returns off-venue movement after this exchange halted the pair.
func (s *Service) GetPostDelist(ctx context.Context, exchange, symbol, interval string, limit int) (*domain.PostDelistView, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	if interval == "" {
		interval = "1d"
	}
	base, _ := inferDelistBaseQuote(ex, symbol)
	view := &domain.PostDelistView{
		Exchange: string(ex),
		Symbol:   symbol,
		Base:     base,
		Interval: interval,
		Quote:    "USD",
	}
	e, ok := s.delistEntry(ex, symbol)
	if !ok || e.DelistTime.IsZero() {
		view.Note = postDelistNoteListed
		return view, nil
	}
	view.DelistTime = e.DelistTime.UTC()
	if e.DelistTime.After(time.Now().UTC()) {
		view.Note = postDelistNoteListed
		return view, nil
	}

	if other := s.postDelistOtherVenue(ctx, ex, symbol, interval, limit, view.DelistTime); other != nil {
		other.DelistTime = view.DelistTime
		other.Base = view.Base
		return other, nil
	}
	if s.offVenue == nil {
		view.Note = postDelistNoteNone
		return view, nil
	}
	q, err := s.offVenue.QuoteByBase(ctx, base)
	if err != nil || q == nil {
		view.Note = postDelistNoteNone
		return view, nil
	}
	view.Available = true
	view.Source = "coingecko"
	view.SourceLabel = "CoinGecko"
	view.Note = postDelistNoteGecko
	view.LastPrice = strconv.FormatFloat(q.LastUSD, 'f', -1, 64)
	view.AsOf = q.AsOf
	if q.ChangePct != nil {
		view.PriceChangePercent = strconv.FormatFloat(*q.ChangePct, 'f', 3, 64)
	}
	days := 30
	if strings.HasSuffix(interval, "h") || strings.HasSuffix(interval, "m") {
		days = 7
	}
	bars, err := s.offVenue.OHLCByBase(ctx, base, days)
	if err == nil {
		bars = barsAfter(bars, view.DelistTime)
		if len(bars) > limit {
			bars = bars[len(bars)-limit:]
		}
		view.Candles = bars
	}
	return view, nil
}

func (s *Service) postDelistOtherVenue(ctx context.Context, home domain.Exchange, symbol, interval string, limit int, halt time.Time) *domain.PostDelistView {
	order := []domain.Exchange{domain.ExchangeBybit, domain.ExchangeCoinbase, domain.ExchangeBinance}
	for _, other := range order {
		if other == home {
			continue
		}
		p, err := s.port(other)
		if err != nil {
			continue
		}
		sym := normalizeSymbolForExchange(other, symbol)
		if other == domain.ExchangeCoinbase && !strings.Contains(sym, "-") {
			base, quote := inferDelistBaseQuote(home, symbol)
			if quote == "USDT" || quote == "USDC" || quote == "" {
				quote = "USD"
			}
			if base != "" {
				sym = base + "-" + quote
			}
		}
		if e, ok := s.delistEntry(other, sym); ok && !e.DelistTime.IsZero() && !e.DelistTime.After(time.Now().UTC()) {
			continue
		}
		tkr, err := p.GetTicker24h(ctx, sym)
		if err != nil || tkr == nil || strings.TrimSpace(tkr.LastPrice) == "" || tkr.Halted {
			continue
		}
		view := &domain.PostDelistView{
			Exchange:           string(home),
			Symbol:             symbol,
			Available:          true,
			Source:             string(other),
			SourceLabel:        venueLabel(other),
			Note:               fmt.Sprintf(postDelistNoteVenue, venueLabel(other)),
			LastPrice:          tkr.LastPrice,
			PriceChangePercent: tkr.PriceChangePercent,
			Quote:              "USDT",
			AsOf:               time.Now().UTC(),
			Interval:           interval,
		}
		if other == domain.ExchangeCoinbase {
			view.Quote = "USD"
		}
		q := domain.CandleQuery{
			Symbol:   sym,
			Interval: domain.CandleInterval(interval),
			Limit:    limit,
		}
		if !halt.IsZero() {
			q.StartTime = halt.UTC()
		}
		bars, berr := p.GetCandles(ctx, q)
		if berr == nil {
			view.Candles = barsAfter(bars, halt)
		}
		return view
	}
	return nil
}

func barsAfter(bars []domain.Candle, halt time.Time) []domain.Candle {
	if halt.IsZero() || len(bars) == 0 {
		return bars
	}
	cut := halt.UTC()
	out := make([]domain.Candle, 0, len(bars))
	for _, c := range bars {
		if c.OpenTime.IsZero() || !c.OpenTime.Before(cut) {
			out = append(out, c)
		}
	}
	return out
}

// offVenueTape is the same 24h last/change coin detail shows after halt:
// another live venue first, then CoinGecko price_change_24h + percentage.
type offVenueTape struct {
	Change  string
	Percent string
}

func (s *Service) offVenueTape(ctx context.Context, home domain.Exchange, symbol string) offVenueTape {
	if s == nil {
		return offVenueTape{}
	}
	order := []domain.Exchange{domain.ExchangeBybit, domain.ExchangeCoinbase, domain.ExchangeBinance}
	for _, other := range order {
		if other == home {
			continue
		}
		p, err := s.port(other)
		if err != nil {
			continue
		}
		sym := normalizeSymbolForExchange(other, symbol)
		if other == domain.ExchangeCoinbase && !strings.Contains(sym, "-") {
			base, quote := inferDelistBaseQuote(home, symbol)
			if quote == "USDT" || quote == "USDC" || quote == "" {
				quote = "USD"
			}
			if base != "" {
				sym = base + "-" + quote
			}
		}
		if e, ok := s.delistEntry(other, sym); ok && !e.DelistTime.IsZero() && !e.DelistTime.After(time.Now().UTC()) {
			continue
		}
		tkr, err := p.GetTicker24h(ctx, sym)
		if err != nil || tkr == nil || strings.TrimSpace(tkr.LastPrice) == "" || tkr.Halted {
			continue
		}
		tape := offVenueTape{
			Change:  strings.TrimSpace(tkr.PriceChange),
			Percent: strings.TrimSpace(tkr.PriceChangePercent),
		}
		if tape.Change == "" && tape.Percent != "" && strings.TrimSpace(tkr.LastPrice) != "" {
			tape.Change = absChangeFromLastPct(tkr.LastPrice, tape.Percent)
		}
		if tape.Percent != "" || tape.Change != "" {
			return tape
		}
	}
	if s.offVenue == nil {
		return offVenueTape{}
	}
	base, _ := inferDelistBaseQuote(home, symbol)
	q, err := s.offVenue.QuoteByBase(ctx, base)
	if err != nil || q == nil {
		return offVenueTape{}
	}
	q.FillChangeAbs()
	tape := offVenueTape{}
	if q.ChangePct != nil {
		tape.Percent = strconv.FormatFloat(*q.ChangePct, 'f', 3, 64)
	}
	if q.ChangeAbs != nil {
		tape.Change = strconv.FormatFloat(*q.ChangeAbs, 'f', -1, 64)
	}
	return tape
}

func absChangeFromLastPct(last, pct string) string {
	l, err1 := strconv.ParseFloat(strings.TrimSpace(last), 64)
	p, err2 := strconv.ParseFloat(strings.TrimSpace(pct), 64)
	if err1 != nil || err2 != nil {
		return ""
	}
	frac := p / 100
	if frac <= -1 {
		return ""
	}
	return strconv.FormatFloat(l*frac/(1+frac), 'f', -1, 64)
}

func (s *Service) fillHaltedOffVenueChange(ctx context.Context, ex domain.Exchange, items []domain.SpotMarket) {
	if s == nil || len(items) == 0 {
		return
	}
	now := time.Now().UTC()
	var idxs []int
	for i := range items {
		if items[i].DelistTime == nil || items[i].DelistTime.After(now) {
			continue
		}
		if strings.TrimSpace(items[i].PriceChangePercent) != "" && strings.TrimSpace(items[i].PriceChange) != "" {
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
			tape := s.offVenueTape(ctx, ex, items[i].Symbol)
			if tape.Percent != "" {
				items[i].PriceChangePercent = tape.Percent
			}
			if tape.Change != "" {
				items[i].PriceChange = tape.Change
			}
		}()
	}
	wg.Wait()
}

func venueLabel(ex domain.Exchange) string {
	switch ex {
	case domain.ExchangeBybit:
		return "Bybit"
	case domain.ExchangeCoinbase:
		return "Coinbase"
	case domain.ExchangeBinance:
		return "Binance"
	default:
		return string(ex)
	}
}
