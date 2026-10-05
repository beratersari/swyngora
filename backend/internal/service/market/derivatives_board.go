package market

import (
	"context"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	derivativesBoardCacheTTL = 20 * time.Second
	derivativesBoardWorkers  = 4
)

// GetDerivativesBoard returns the liquid USDT perpetual desk: mark, funding,
// open interest, crowd long, top-versus-crowd, day CVD, print size, 15m order
// count, the farthest unusual reading, liquidation heat, 1% bid share, taker
// buy share, and 24h liquidations.
func (s *Service) GetDerivativesBoard(ctx context.Context, exchange string, limit int) (*domain.DerivativesBoard, error) {
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	limit = domain.ParseDerivativesBoardLimit(limit)
	key := ex + "|" + strconv.Itoa(limit)
	if c := s.derivativesBoardCache(); c != nil {
		if hit, ok := c.Get(key); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	symbols, err := s.LiquidPerpSymbols(ctx, limit)
	if err != nil && len(symbols) == 0 {
		return nil, err
	}
	var (
		scan    *domain.UnusualScan
		scanErr error
		scanWG  sync.WaitGroup
	)
	scanWG.Add(1)
	go func() {
		defer scanWG.Done()
		scan, scanErr = s.ScanUnusual(ctx, ex, "all", limit)
	}()
	want := domain.ExpectedFuturesVenues(ex)
	coins := make([]domain.DerivativesCoinInput, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, derivativesBoardWorkers)
	for i, sym := range symbols {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				coins[i] = domain.DerivativesCoinInput{Symbol: sym}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			coins[i] = s.derivativesCoin(ctx, ex, sym, want)
		}(i, sym)
	}
	wg.Wait()
	scanWG.Wait()
	if scanErr != nil {
		return nil, scanErr
	}
	now := time.Now().UTC()
	out := domain.BuildDerivativesBoard(ex, coins, now)
	if out != nil {
		if spans, err := s.recordingBySymbol(ctx, symbols, now); err != nil {
			return nil, err
		} else {
			domain.AttachRecordedSpans(out, spans)
		}
		if scan != nil {
			domain.AttachBoardUnusual(out, scan.Hits)
		}
	}
	if c := s.derivativesBoardCache(); c != nil && out != nil {
		cp := *out
		c.Set(key, &cp)
	}
	return out, nil
}

func (s *Service) derivativesCoin(ctx context.Context, exchange, symbol string, want []domain.Exchange) domain.DerivativesCoinInput {
	s.noteFutures(symbol)
	venues := make([]domain.DerivativesVenueInput, len(want))
	var wg sync.WaitGroup
	for i, ex := range want {
		wg.Add(1)
		go func(i int, ex domain.Exchange) {
			defer wg.Done()
			venues[i] = s.derivativesVenue(ctx, ex, symbol)
		}(i, ex)
	}
	wg.Wait()
	longs, shorts := s.derivativesLiquidations(exchange, symbol, time.Now().UTC().Add(-24*time.Hour))
	for i := range venues {
		venues[i].LiqLong = longs[venues[i].Exchange]
		venues[i].LiqShort = shorts[venues[i].Exchange]
	}
	in := domain.DerivativesCoinInput{Symbol: symbol, Venues: venues}
	if s != nil && s.liq != nil {
		now := time.Now().UTC()
		snap := s.liq.Snapshot(exchange, symbol)
		in.LiqFeed = snap.Feed
		cover := map[domain.Exchange]int64{}
		for _, h := range snap.Feed.Venues {
			cover[domain.Exchange(h.Exchange)] = h.CoverageSeconds
		}
		events := s.liq.EventsSince(exchange, symbol, now.Add(-domain.DerivativesLiqHeatMinCoverage))
		minCover := int64(domain.DerivativesLiqHeatMinCoverage / time.Second)
		for i := range in.Venues {
			ex := in.Venues[i].Exchange
			if cover[ex] < minCover {
				continue
			}
			detected := domain.DetectCascadeVenue(ex, symbol, events, now)
			for _, w := range detected.Windows {
				if w.Window != domain.CascadeWindow5m || !w.Complete {
					continue
				}
				in.Venues[i].LiqHeatOK = true
				in.Venues[i].LiqGrade = w.Grade
				in.Venues[i].LiqRatio = w.MaxRatio
			}
		}
	}
	return in
}

func (s *Service) derivativesDayCVD(ctx context.Context, ex domain.Exchange, symbol string) (float64, time.Time, bool) {
	if s == nil {
		return 0, time.Time{}, false
	}
	now := time.Now().UTC()
	from := now.Truncate(24 * time.Hour)
	var live []domain.TakerBucket
	if p := s.bucketPort(ex); p != nil {
		got, err := p.GetTakerBuckets(ctx, symbol)
		if err == nil {
			live = got
		}
	}
	var stored []domain.TakerBucket
	if s.takerStore != nil {
		stored, _ = s.takerStore.ListTakerBuckets(ctx, string(ex), symbol, from, now)
	}
	merged := domain.MergeTakerBuckets(stored, live)
	if len(merged) == 0 {
		return 0, time.Time{}, false
	}
	started := now
	for _, b := range merged {
		if !b.Start.IsZero() && b.Start.Before(started) {
			started = b.Start
		}
	}
	return domain.DayCVDDelta(merged, now, started)
}

type derivativesTapeRead struct {
	printNotional   float64
	printCount      int
	printPrior      float64
	printPriorCount int
	printAt         time.Time
	printOK         bool
	orderBuy        int
	orderSell       int
	orderAt         time.Time
	orderOK         bool
}

// derivativesTape reads the 15-minute print-size change and the 15-minute
// order-count ratio from one print slice. A window the tape does not cover
// stays unset so the board does not treat it as zero.
func (s *Service) derivativesTape(ctx context.Context, ex domain.Exchange, symbol string) derivativesTapeRead {
	var out derivativesTapeRead
	if s == nil {
		return out
	}
	now := time.Now().UTC()
	from := now.Add(-domain.PrintSizeLookback())
	prints, covered := s.countedPrints(ctx, ex, symbol, from, now)
	if !covered {
		return out
	}
	sized := domain.BuildPrintSizeVenue(ex, symbol, prints, now)
	for _, w := range sized.Windows {
		if w.Window != domain.PrintSizeWindow15m || !w.Complete {
			continue
		}
		count := w.Buy.Count + w.Sell.Count
		prior := w.Buy.PriorCount + w.Sell.PriorCount
		if count <= 0 || prior <= 0 {
			break
		}
		out.printNotional = w.Buy.Notional + w.Sell.Notional
		out.printCount = count
		out.printPrior = w.Buy.PriorNotional + w.Sell.PriorNotional
		out.printPriorCount = prior
		out.printAt = sized.Newest
		out.printOK = !sized.Newest.IsZero()
		break
	}
	counted := domain.BuildOrderCount(ex, symbol, prints, now)
	for _, w := range counted.Windows {
		if w.Window != domain.OrderCount15m || !w.Complete || w.Total <= 0 || counted.Newest.IsZero() {
			continue
		}
		out.orderBuy = w.BuyCount
		out.orderSell = w.SellCount
		out.orderAt = counted.Newest
		out.orderOK = true
		break
	}
	return out
}

func (s *Service) derivativesBidShare(ctx context.Context, ex domain.Exchange, symbol string) (float64, float64, time.Time, bool) {
	if s == nil {
		return 0, 0, time.Time{}, false
	}
	p := s.futuresDepthPort(ex)
	if p == nil {
		return 0, 0, time.Time{}, false
	}
	book, err := p.GetFuturesDepth(ctx, symbol)
	if err != nil || book == nil {
		return 0, 0, time.Time{}, false
	}
	ven := domain.BuildPerpDepthVenue(ex, symbol, book, domain.DefaultPerpDepthRangePct)
	if ven.Error != "" || !ven.Headline.Complete || ven.Headline.Below.Notional+ven.Headline.Above.Notional <= 0 {
		return 0, 0, time.Time{}, false
	}
	at := book.FetchedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return ven.Headline.Below.Notional, ven.Headline.Above.Notional, at, true
}

func (s *Service) derivativesVenue(ctx context.Context, ex domain.Exchange, symbol string) domain.DerivativesVenueInput {
	out := domain.DerivativesVenueInput{Exchange: ex}
	if s == nil {
		return out
	}
	var (
		oi             *domain.OpenInterestSeries
		oiOK           bool
		fundOK         bool
		fundRate       float64
		fundAt         time.Time
		mark           float64
		longShare      float64
		longAt         time.Time
		longOK         bool
		topShare       float64
		topOK          bool
		topUnsupported bool
		buy            float64
		sell           float64
		takerOK        bool
		cvdDay         float64
		cvdAt          time.Time
		cvdOK          bool
		printNotional  float64
		printCount     int
		printPrior     float64
		printPriorN    int
		printAt        time.Time
		printOK        bool
		orderBuy       int
		orderSell      int
		orderAt        time.Time
		orderOK        bool
		bidNotional    float64
		askNotional    float64
		bookAt         time.Time
		bookOK         bool
		wg             sync.WaitGroup
	)
	wg.Add(5)
	go func() {
		defer wg.Done()
		if p := s.oiPort(ex); p != nil {
			ser, err := p.GetOpenInterestSeries(ctx, symbol)
			if err == nil && ser != nil && derivativesSeriesPresent(ser) {
				oi = ser
				oiOK = true
			}
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.fundingPort(ex); p != nil {
			ser, err := p.GetFundingSeries(ctx, symbol, 1)
			if err == nil && ser != nil && !ser.Current.Time.IsZero() {
				fundOK = true
				fundRate = ser.Current.Rate
				fundAt = ser.Current.Time
				mark = ser.Current.MarkPrice
			}
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.longShortPort(ex); p != nil {
			ser, err := p.GetLongShortSeries(ctx, symbol, 1)
			share, at, ok, top, hasTop, unsupported := derivativesLongShortRead(ser, err)
			if ok {
				longShare = share
				longAt = at
				longOK = true
				topShare = top
				topOK = hasTop
				topUnsupported = unsupported
			}
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.takerPort(ex); p != nil {
			flow, err := p.GetTakerFlow(ctx, symbol)
			if b, sNotional, ok := derivativesTakerHour(flow, err); ok {
				buy = b
				sell = sNotional
				takerOK = true
			}
		}
		cvdDay, cvdAt, cvdOK = s.derivativesDayCVD(ctx, ex, symbol)
		tape := s.derivativesTape(ctx, ex, symbol)
		printNotional, printCount, printPrior, printPriorN, printAt, printOK = tape.printNotional, tape.printCount, tape.printPrior, tape.printPriorCount, tape.printAt, tape.printOK
		if tape.orderOK {
			orderBuy, orderSell, orderAt = tape.orderBuy, tape.orderSell, tape.orderAt
			orderOK = true
		}
	}()
	go func() {
		defer wg.Done()
		bidNotional, askNotional, bookAt, bookOK = s.derivativesBidShare(ctx, ex, symbol)
	}()
	wg.Wait()
	if oiOK {
		out.OI = oi
		out.OIValue = oi.Current.Value
		out.OIAt = oi.Current.Time
		out.OIOK = true
	}
	if fundOK {
		out.FundingOK = true
		out.FundingRate = fundRate
		out.FundingAt = fundAt
		out.MarkPrice = mark
	}
	if longOK {
		out.LongShortOK = true
		out.LongShare = longShare
		out.LongShortAt = longAt
	}
	if takerOK {
		out.TakerOK = true
		out.TakerBuy = buy
		out.TakerSell = sell
	}
	if topOK {
		out.TopLongOK = true
		out.TopLongShare = topShare
	} else if topUnsupported {
		out.TopUnsupported = true
	}
	if cvdOK {
		out.CVDDayOK = true
		out.CVDDay = cvdDay
		out.CVDAt = cvdAt
	}
	if printOK {
		out.PrintOK = true
		out.PrintNotional = printNotional
		out.PrintCount = printCount
		out.PrintPriorNotional = printPrior
		out.PrintPriorCount = printPriorN
		out.PrintAt = printAt
	}
	if orderOK {
		out.OrderOK = true
		out.OrderBuy = orderBuy
		out.OrderSell = orderSell
		out.OrderAt = orderAt
	}
	if bookOK {
		out.BookOK = true
		out.BidNotional = bidNotional
		out.AskNotional = askNotional
		out.BookAt = bookAt
	}
	return out
}

func derivativesLongShortRead(ser *domain.LongShortSeries, err error) (crowd float64, at time.Time, crowdOK bool, top float64, topOK, unsupported bool) {
	if err != nil || ser == nil || ser.Current.Time.IsZero() {
		return 0, time.Time{}, false, 0, false, false
	}
	share := ser.Current.LongShare
	if share < 0 || share > 1 || share != share {
		return 0, time.Time{}, false, 0, false, false
	}
	topSer := ser.TopPositions
	if topSer == nil {
		topSer = ser.TopAccounts
	}
	if topSer != nil && !topSer.Current.Time.IsZero() {
		ts := topSer.Current.LongShare
		if ts >= 0 && ts <= 1 && ts == ts {
			return share, ser.Current.Time.UTC(), true, ts, true, false
		}
	}
	return share, ser.Current.Time.UTC(), true, 0, false, true
}

func derivativesTakerHour(flow *domain.TakerVenueFlow, err error) (float64, float64, bool) {
	if err != nil || flow == nil {
		return 0, 0, false
	}
	for _, w := range flow.Windows {
		if w.Window != domain.TakerWindow1h || !w.Complete {
			continue
		}
		if w.BuyNotional+w.SellNotional <= 0 {
			return 0, 0, false
		}
		return w.BuyNotional, w.SellNotional, true
	}
	return 0, 0, false
}

func derivativesSeriesPresent(ser *domain.OpenInterestSeries) bool {
	if ser == nil {
		return false
	}
	cur := ser.Current
	return !cur.Time.IsZero() || cur.Value != 0 || cur.Contracts != 0
}

func (s *Service) derivativesLiquidations(exchange, symbol string, since time.Time) (map[domain.Exchange]float64, map[domain.Exchange]float64) {
	longs := map[domain.Exchange]float64{}
	shorts := map[domain.Exchange]float64{}
	if s == nil || s.liq == nil {
		return longs, shorts
	}
	for _, e := range s.liq.EventsSince(exchange, symbol, since) {
		switch e.Side {
		case domain.LiquidationSideLong:
			longs[e.Exchange] += e.Notional
		case domain.LiquidationSideShort:
			shorts[e.Exchange] += e.Notional
		}
	}
	return longs, shorts
}

func (s *Service) derivativesBoardCache() *cache.TTL[*domain.DerivativesBoard] {
	if s == nil {
		return nil
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if s.boardCache == nil {
		s.boardCache = cache.New[*domain.DerivativesBoard](derivativesBoardCacheTTL)
	}
	return s.boardCache
}
