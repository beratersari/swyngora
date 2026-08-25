package market

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const aroundDisclaimer = "Before / during / after are adjacent windows around the time you pick: before is the lookback ending at that time, during is the move starting there, after is the same lookback after the move. Price, volume, buy/sell, VWAP, volume vs typical, and the volume-profile POC come from spot candles. Liquidity sweeps use 15-minute bars. Stored order-book and futures history are attached when those archives have a sample near the window. Informational only — not financial advice."

const aroundSweepLookback = 48 * time.Hour

// GetAround assembles what changed before, during, and after a chosen time.
func (s *Service) GetAround(ctx context.Context, exchange, symbol, window, during string, at time.Time) (*domain.AroundReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	plan, err := domain.ResolveAroundPlan(window, during, at, now)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	interval := domain.ProfileBarInterval(plan.To.Sub(plan.From.Add(-time.Duration(domain.AroundTypicalPriors) * plan.WindowDur)))

	out := &domain.AroundReport{
		Symbol: symbol, Exchange: ex, At: plan.At, Window: plan.Window, During: plan.During,
		From: plan.From, To: plan.To, AsOf: now, Clipped: plan.Clipped,
		Venues: make([]domain.AroundVenue, 0, len(want)),
		Note:   aroundDisclaimer,
	}

	type fetched struct {
		ex  domain.Exchange
		ven domain.AroundVenue
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			got[i] = fetched{ex: v, ven: s.aroundVenue(ctx, v, symbol, plan, interval)}
		}(i, v)
	}
	wg.Wait()
	for _, f := range got {
		out.Venues = append(out.Venues, f.ven)
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombineAroundVenues(symbol, out.Venues, plan, interval)
	}
	out.Summary = domain.ExplainAroundReport(*out)
	return out, nil
}

const aroundCompareDisclaimer = "Compares two moves of the same coin: each time is an around-the-move tape (before / during / after). Differences cover price level, the move itself (net %, range, volume, vs typical, takers, POC), stored order-book mid/liquidity, and stored futures (OI, funding, long %, liquidations) when those archives have samples. from and to are the two event times — they do not have to be in order. Informational only — not financial advice."

// CompareAround diffs two around-the-move tapes for the same coin.
func (s *Service) CompareAround(ctx context.Context, exchange, symbol, window, during string, from, to time.Time) (*domain.AroundCompareReport, error) {
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("%w: from and to times are required", domain.ErrInvalidArgument)
	}
	if from.UTC().Equal(to.UTC()) {
		return nil, fmt.Errorf("%w: from and to must be different times", domain.ErrInvalidArgument)
	}
	type got struct {
		rep *domain.AroundReport
		err error
	}
	var a, b got
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		rep, err := s.GetAround(ctx, exchange, symbol, window, during, from)
		a = got{rep, err}
	}()
	go func() {
		defer wg.Done()
		rep, err := s.GetAround(ctx, exchange, symbol, window, during, to)
		b = got{rep, err}
	}()
	wg.Wait()
	if a.err != nil {
		return nil, a.err
	}
	if b.err != nil {
		return nil, b.err
	}
	out := domain.CompareAroundReports(*a.rep, *b.rep)
	s.attachCompareBooks(ctx, exchange, symbol, from, to, &out)
	out.Note = aroundCompareDisclaimer
	if out.Summary == "" {
		out.Summary = domain.ExplainAroundCompare(out)
	}
	return &out, nil
}

func (s *Service) attachCompareBooks(ctx context.Context, exchange, symbol string, from, to time.Time, out *domain.AroundCompareReport) {
	if s == nil || s.bookHist == nil || out == nil {
		return
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if out.Exchange != "all" && out.Exchange != "" {
		want = []domain.Exchange{domain.Exchange(out.Exchange)}
	}
	for _, ex := range want {
		s.noteBook(ex, symbol)
		diff, err := s.bookHist.Compare(ctx, string(ex), symbol, from.UTC(), to.UTC())
		if err != nil || diff == nil {
			continue
		}
		book := domain.BookFromDiff(*diff)
		if book == nil {
			continue
		}
		for i := range out.Venues {
			if out.Venues[i].Exchange == ex {
				out.Venues[i].Book = book
				if mid, ok := findVenueState(&out.Venues[i], domain.AroundCompareMetricBookMid); !ok || mid.From == 0 {
					out.Venues[i].State = append(out.Venues[i].State, domain.AroundCompareDelta{
						Metric: domain.AroundCompareMetricBookMid,
						From:   book.FromMid, To: book.ToMid,
						Change: book.MidDelta, ChangePct: book.MidDeltaPct,
						Direction: domainDir(book.MidDeltaPct),
						Summary:   book.Summary,
					})
				}
				out.Venues[i].Summary = domain.ExplainAroundCompareVenue(out.Venues[i])
			}
		}
	}
	_ = exchange
}

func findVenueState(v *domain.AroundCompareVenue, metric string) (domain.AroundCompareDelta, bool) {
	if v == nil {
		return domain.AroundCompareDelta{}, false
	}
	for _, d := range v.State {
		if d.Metric == metric {
			return d, true
		}
	}
	return domain.AroundCompareDelta{}, false
}

func domainDir(pct float64) string {
	switch {
	case pct > 0.05:
		return domain.CVDDirUp
	case pct < -0.05:
		return domain.CVDDirDown
	default:
		return domain.CVDDirFlat
	}
}

func (s *Service) aroundVenue(ctx context.Context, ex domain.Exchange, symbol string, plan domain.AroundPlan, interval domain.CandleInterval) domain.AroundVenue {
	lookbackFrom := plan.BeforeFrom.Add(-time.Duration(domain.AroundTypicalPriors) * plan.WindowDur)
	var (
		candles []domain.Candle
		sweepC  []domain.Candle
		cerr    error
		wg      sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		candles, cerr = s.candlesForProfile(ctx, ex, symbol, interval, lookbackFrom, plan.To)
	}()
	go func() {
		defer wg.Done()
		sweepFrom := plan.From.Add(-aroundSweepLookback)
		if sweepFrom.After(lookbackFrom) {
			sweepFrom = lookbackFrom
		}
		var err error
		sweepC, err = s.GetCandles(ctx, string(ex), symbol, "15m", domain.SweepCandleLimit, &sweepFrom, &plan.To)
		if err != nil {
			sweepC = nil
		}
	}()
	wg.Wait()
	if cerr != nil && len(candles) == 0 {
		return domain.AroundVenue{
			Exchange: ex, Symbol: symbol, Interval: string(interval),
			Phases: []domain.AroundPhase{}, Error: cerr.Error(), Summary: cerr.Error(),
		}
	}
	bars := domain.AroundBarsFromCandles(filterCandlesRange(candles, lookbackFrom, plan.To, interval))
	ven := domain.BuildAroundVenue(ex, symbol, bars, plan, interval)
	if ven.Error != "" {
		return ven
	}

	if len(sweepC) > 0 {
		sweeps := domain.DetectLiquiditySweeps(domain.SweepBarsFromCandles(sweepC), 15*time.Minute)
		if evs := domain.AroundSweepsToEvents(sweeps, plan.From, plan.To); len(evs) > 0 {
			domain.AttachAroundEvents(&ven, evs)
		}
	}

	s.attachAroundBook(ctx, ex, symbol, plan, &ven)
	s.attachAroundFutures(ctx, ex, symbol, plan, &ven)
	ven.Summary = domain.ExplainAroundVenue(ven)
	return ven
}

func (s *Service) attachAroundBook(ctx context.Context, ex domain.Exchange, symbol string, plan domain.AroundPlan, ven *domain.AroundVenue) {
	if s == nil || s.bookHist == nil || ven == nil {
		return
	}
	s.noteBook(ex, symbol)
	for i := range ven.Phases {
		ph := &ven.Phases[i]
		if !ph.To.After(ph.From) {
			continue
		}
		fromSnap, err := s.bookHist.SnapshotAt(ctx, string(ex), symbol, ph.From)
		if err != nil || fromSnap == nil || fromSnap.SampledAt.After(ph.To) {
			continue
		}
		toSnap, err := s.bookHist.SnapshotAt(ctx, string(ex), symbol, ph.To)
		if err != nil || toSnap == nil || toSnap.SampledAt.After(ph.To) {
			continue
		}
		diff := domain.CompareBookHistory(*fromSnap, *toSnap)
		if b := domain.BookFromDiff(diff); b != nil {
			ph.Book = b
			ph.Summary = domain.ExplainAroundPhase(*ph)
		}
	}
}

func (s *Service) attachAroundFutures(ctx context.Context, ex domain.Exchange, symbol string, plan domain.AroundPlan, ven *domain.AroundVenue) {
	if s == nil || s.futHist == nil || ven == nil {
		return
	}
	pad := 15 * time.Minute
	from := plan.From.Add(-pad)
	to := plan.To.Add(pad)
	oi := s.aroundFuturesSnaps(ctx, domain.FuturesMetricOpenInterest, string(ex), symbol, from, to)
	fund := s.aroundFuturesSnaps(ctx, domain.FuturesMetricFunding, string(ex), symbol, from, to)
	ls := s.aroundFuturesSnaps(ctx, domain.FuturesMetricLongShort, string(ex), symbol, from, to)
	liqs := s.aroundFuturesLiqs(ctx, string(ex), symbol, plan.From, plan.To)
	if len(oi) == 0 && len(fund) == 0 && len(ls) == 0 && len(liqs) == 0 {
		return
	}
	for i := range ven.Phases {
		ph := &ven.Phases[i]
		if !ph.To.After(ph.From) {
			continue
		}
		var phaseLiqs []domain.LiquidationEvent
		for _, e := range liqs {
			if e.Time.Before(ph.From) || !e.Time.Before(ph.To) {
				continue
			}
			phaseLiqs = append(phaseLiqs, e)
		}
		got := domain.FuturesAcrossSamples(
			domain.LatestFuturesSnapshotAtOrBefore(oi, ph.From),
			domain.LatestFuturesSnapshotAtOrBefore(oi, ph.To),
			domain.LatestFuturesSnapshotAtOrBefore(fund, ph.From),
			domain.LatestFuturesSnapshotAtOrBefore(fund, ph.To),
			domain.LatestFuturesSnapshotAtOrBefore(ls, ph.From),
			domain.LatestFuturesSnapshotAtOrBefore(ls, ph.To),
			phaseLiqs,
		)
		if got != nil {
			ph.Futures = got
			ph.Summary = domain.ExplainAroundPhase(*ph)
		}
	}
}

func (s *Service) aroundFuturesSnaps(ctx context.Context, metric, exchange, symbol string, from, to time.Time) []domain.FuturesSnapshot {
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric: metric, Exchange: exchange, Symbol: symbol, From: from, To: to, Limit: 200,
	})
	if err != nil || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []domain.FuturesSnapshot:
		return v
	case *[]domain.FuturesSnapshot:
		if v == nil {
			return nil
		}
		return *v
	default:
		return nil
	}
}

func (s *Service) aroundFuturesLiqs(ctx context.Context, exchange, symbol string, from, to time.Time) []domain.LiquidationEvent {
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric: "liquidations", Exchange: exchange, Symbol: symbol, From: from, To: to, Limit: 500,
	})
	if err != nil || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []domain.LiquidationEvent:
		return v
	case *[]domain.LiquidationEvent:
		if v == nil {
			return nil
		}
		return *v
	default:
		return nil
	}
}
