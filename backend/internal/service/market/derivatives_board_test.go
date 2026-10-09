package market

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type boardOI struct {
	series *domain.OpenInterestSeries
	err    error
}

func (f boardOI) GetOpenInterestSeries(context.Context, string) (*domain.OpenInterestSeries, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.series, nil
}

type boardFunding struct {
	series *domain.FundingSeries
}

func (f boardFunding) GetFundingSeries(context.Context, string, int) (*domain.FundingSeries, error) {
	return f.series, nil
}

func (f boardFunding) ListFundingHistory(context.Context, string, time.Time, time.Time) ([]domain.FundingPoint, error) {
	return nil, nil
}

func TestGetDerivativesBoard_FreshVenues(t *testing.T) {
	now := time.Now().UTC()
	svc := boardService(now, false)
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Universe != 5 || !got.Coins[0].OpenInterestDefinite || got.Coins[0].OpenInterest == "" {
		t.Fatalf("%+v", got)
	}
	if !got.Coins[0].FundingDefinite || got.Coins[0].Price == "" {
		t.Fatalf("funding/price %+v", got.Coins[0])
	}
}

func TestGetDerivativesBoard_MissingOKXWithholdsCombined(t *testing.T) {
	now := time.Now().UTC()
	svc := boardService(now, true)
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Coins[0]
	if row.OpenInterestDefinite || row.OpenInterest != "" || got.Definite {
		t.Fatalf("combined %+v coverage %+v", row.OpenInterest, got.Coverage)
	}
	var binance string
	for _, v := range row.Venues {
		if v.Exchange == "binance" {
			binance = v.OIValue
		}
		if v.Exchange == "okx" && v.OIStatus != domain.VenueReadMissing {
			t.Fatalf("okx %+v", v)
		}
	}
	if binance == "" {
		t.Fatalf("venues %+v", row.Venues)
	}
}

type boardLS struct {
	series *domain.LongShortSeries
}

func (f boardLS) GetLongShortSeries(context.Context, string, int) (*domain.LongShortSeries, error) {
	return f.series, nil
}

type boardTaker struct {
	flow *domain.TakerVenueFlow
}

func (f boardTaker) GetTakerFlow(context.Context, string) (*domain.TakerVenueFlow, error) {
	return f.flow, nil
}

func TestGetDerivativesBoard_AddsCrowdAndTaker(t *testing.T) {
	now := time.Now().UTC()
	ls := &domain.LongShortSeries{Current: domain.LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.22}}
	flow := &domain.TakerVenueFlow{Windows: []domain.TakerWindowFlow{
		domain.SummarizeTakerWindow(60, 40, domain.TakerWindow1h, true),
	}}
	portsLS := map[domain.Exchange]domain.LongShortRatioPort{
		domain.ExchangeBinance: boardLS{series: ls},
		domain.ExchangeBybit:   boardLS{series: ls},
		domain.ExchangeOKX:     boardLS{series: ls},
	}
	portsTaker := map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: boardTaker{flow: flow},
		domain.ExchangeBybit:   boardTaker{flow: flow},
		domain.ExchangeOKX:     boardTaker{flow: flow},
	}
	svc := boardService(now, false).WithLongShortRatio(portsLS).WithTakerFlow(portsTaker)
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Coins[0]
	if !row.LongDefinite || row.LongPct != "55" || !row.TakerDefinite || row.TakerBuyPct != "60" {
		t.Fatalf("crowd/taker %+v", row)
	}
}

func TestGetDerivativesBoard_IncompleteTakerStaysBlank(t *testing.T) {
	now := time.Now().UTC()
	flow := &domain.TakerVenueFlow{Windows: []domain.TakerWindowFlow{
		domain.SummarizeTakerWindow(60, 40, domain.TakerWindow1h, false),
	}}
	svc := boardService(now, false).WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: boardTaker{flow: flow},
		domain.ExchangeBybit:   boardTaker{flow: flow},
		domain.ExchangeOKX:     boardTaker{flow: flow},
	})
	got, err := svc.GetDerivativesBoard(context.Background(), "binance", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Coins[0]
	if row.TakerDefinite || row.TakerBuyPct != "" {
		t.Fatalf("incomplete taker leaked %+v", row.TakerBuyPct)
	}
}

type boardTape struct {
	flow    *domain.TakerVenueFlow
	buckets []domain.TakerBucket
}

func (f boardTape) GetTakerFlow(context.Context, string) (*domain.TakerVenueFlow, error) {
	return f.flow, nil
}

func (f boardTape) GetTakerBuckets(context.Context, string) ([]domain.TakerBucket, error) {
	return f.buckets, nil
}

type boardDepth struct{}

func (boardDepth) GetFuturesDepth(_ context.Context, symbol string) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{
		Symbol:    symbol,
		FetchedAt: time.Now().UTC(),
		Bids: []domain.PriceLevel{
			{Price: 100, Quantity: 10},
			{Price: 99, Quantity: 400},
		},
		Asks: []domain.PriceLevel{
			{Price: 100.2, Quantity: 8},
			{Price: 101.3, Quantity: 200},
		},
	}, nil
}

type boardMemPrints struct {
	now time.Time
}

func (boardMemPrints) GetRecentPrints(context.Context, string) ([]domain.TakerPrint, error) {
	return nil, nil
}

func (f boardMemPrints) MemoryPrints(symbol string, from, to time.Time) ([]domain.TakerPrint, bool) {
	return []domain.TakerPrint{
		{Exchange: domain.ExchangeBinance, Symbol: symbol, Side: domain.TakerSideBuy, Notional: 100, Price: 1, Quantity: 100, Time: f.now.Add(-20 * time.Minute)},
		{Exchange: domain.ExchangeBinance, Symbol: symbol, Side: domain.TakerSideBuy, Notional: 200, Price: 1, Quantity: 200, Time: f.now.Add(-30 * time.Second)},
	}, true
}

func TestGetDerivativesBoard_ScanColumnsFromFeeds(t *testing.T) {
	now := time.Now().UTC()
	day := now.Truncate(24 * time.Hour)
	crowd := domain.LongShortSeries{Current: domain.LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.22}}
	binanceLS := crowd
	binanceLS.TopPositions = &domain.LongShortSeries{Current: domain.LongShortPoint{Time: now, LongShare: 0.70, ShortShare: 0.30, Ratio: 2.3}}
	okxLS := crowd
	okxLS.TopPositions = &domain.LongShortSeries{Current: domain.LongShortPoint{Time: now, LongShare: 0.50, ShortShare: 0.50, Ratio: 1}}
	buckets := []domain.TakerBucket{
		{Start: day, BuyNotional: 30, SellNotional: 10},
		{Start: now.Add(-time.Minute), BuyNotional: 5, SellNotional: 1},
	}
	tape := boardTape{buckets: buckets}
	prints := boardMemPrints{now: now}
	depth := map[domain.Exchange]domain.FuturesDepthPort{
		domain.ExchangeBinance: boardDepth{},
		domain.ExchangeBybit:   boardDepth{},
		domain.ExchangeOKX:     boardDepth{},
	}
	svc := boardService(now, false).
		WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
			domain.ExchangeBinance: boardLS{series: &binanceLS},
			domain.ExchangeBybit:   boardLS{series: &crowd},
			domain.ExchangeOKX:     boardLS{series: &okxLS},
		}).
		WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
			domain.ExchangeBinance: tape,
			domain.ExchangeBybit:   tape,
			domain.ExchangeOKX:     tape,
		}).
		WithRecentPrints(map[domain.Exchange]domain.RecentPrintPort{
			domain.ExchangeBinance: prints,
			domain.ExchangeBybit:   prints,
			domain.ExchangeOKX:     prints,
		}).
		WithFuturesDepth(depth)
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Coins[0]
	if !row.GapDefinite || row.GapScope != "binance,okx" || row.GapPct != "+5" {
		t.Fatalf("gap %s %s", row.GapPct, row.GapScope)
	}
	var bybitGap string
	for _, v := range row.Venues {
		if v.Exchange == "bybit" {
			bybitGap = v.GapStatus
		}
	}
	if bybitGap != domain.VenueReadUnsupported {
		t.Fatalf("bybit %s", bybitGap)
	}
	if !row.CVDDayDefinite || row.CVDDay != "+72" {
		t.Fatalf("cvd %s definite %v", row.CVDDay, row.CVDDayDefinite)
	}
	if row.CVDSplitDefinite || row.CVDSplit != "" {
		t.Fatalf("split without a price leaked %q", row.CVDSplit)
	}
	if !row.PrintDefinite || row.PrintChangePct != "+100" {
		t.Fatalf("print %s", row.PrintChangePct)
	}
	if !row.OrderDefinite || row.OrderRatio != "+1.00" || row.OrderLean != domain.OrderCountLeanBuy || row.OrderBuy != 3 || row.OrderSell != 0 {
		t.Fatalf("order %s %s %d %d", row.OrderRatio, row.OrderLean, row.OrderBuy, row.OrderSell)
	}
	if !row.BidDefinite || row.BidPct == "" {
		t.Fatalf("bid %+v", row.BidPct)
	}
}

func TestGetDerivativesBoard_BadExchange(t *testing.T) {
	svc := New(nil, nil)
	if _, err := svc.GetDerivativesBoard(context.Background(), "coinbase", 5); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

type boardCandles struct {
	bars []domain.Candle
}

func (f boardCandles) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	return f.bars, nil
}

func (boardCandles) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return nil, errors.New("unused")
}

func (boardCandles) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return nil, errors.New("unused")
}

func (boardCandles) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	return nil, errors.New("unused")
}

func (boardCandles) ListProductTags(context.Context) ([]string, error) {
	return nil, errors.New("unused")
}

func (boardCandles) TagsByBase(context.Context) (map[string][]string, error) {
	return nil, errors.New("unused")
}

func TestGetDerivativesBoard_CVDSplitFromTape(t *testing.T) {
	now := time.Now().UTC()
	day := now.Truncate(24 * time.Hour)
	buckets := []domain.TakerBucket{
		{Start: now.Add(-domain.DefaultCVDLookback), BuyNotional: 1, SellNotional: 1},
		{Start: day, BuyNotional: 1, SellNotional: 1},
	}
	bars := []domain.Candle{
		{OpenTime: now.Add(-domain.DefaultCVDLookback), Close: "100"},
		{OpenTime: day, Close: "100"},
	}
	for i := 4; i >= 1; i-- {
		at := now.Add(-time.Duration(i) * 5 * time.Minute).Truncate(5 * time.Minute)
		buckets = append(buckets, domain.TakerBucket{Start: at, SellNotional: 40})
		bars = append(bars, domain.Candle{OpenTime: at, Close: strconv.FormatFloat(100+float64(5-i)*0.4, 'f', 2, 64)})
	}
	buckets = append(buckets, domain.TakerBucket{Start: now.Add(-30 * time.Second).Truncate(time.Minute), SellNotional: 20})
	tape := boardTape{buckets: buckets}
	ports := map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: tape,
		domain.ExchangeBybit:   tape,
		domain.ExchangeOKX:     tape,
	}
	svc := boardService(now, false).WithTakerFlow(ports)
	short, err := svc.GetDerivativesBoard(context.Background(), "binance", 5)
	if err != nil {
		t.Fatal(err)
	}
	if short.Coins[0].CVDSplitDefinite {
		t.Fatal("a price-less tape must not mark a split")
	}

	priced := New(boardCandles{bars: bars}, nil).
		WithOpenInterest(svc.oi).
		WithFundingRate(svc.funding).
		WithTakerFlow(ports)
	got, err := priced.GetDerivativesBoard(context.Background(), "binance", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Coins[0]
	if !row.CVDSplitDefinite || row.CVDSplit != domain.CVDDivPriceUpCVDDown || row.CVDSplitSeconds == "" || row.CVDSplitSeconds == "0" {
		t.Fatalf("binance split %s %s definite %v", row.CVDSplit, row.CVDSplitSeconds, row.CVDSplitDefinite)
	}

	all, err := priced.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	combined := all.Coins[0]
	if !combined.CVDSplitDefinite || combined.CVDSplit != domain.CVDDivPriceUpCVDDown {
		t.Fatalf("combined %s definite %v", combined.CVDSplit, combined.CVDSplitDefinite)
	}

	dayOnly := []domain.TakerBucket{{Start: day, BuyNotional: 1, SellNotional: 1}}
	for i := 4; i >= 1; i-- {
		at := now.Add(-time.Duration(i) * 5 * time.Minute).Truncate(5 * time.Minute)
		dayOnly = append(dayOnly, domain.TakerBucket{Start: at, SellNotional: 40})
	}
	dayOnly = append(dayOnly, domain.TakerBucket{Start: now.Add(-30 * time.Second).Truncate(time.Minute), SellNotional: 20})
	partial := boardTape{buckets: dayOnly}
	partialPorts := map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: partial,
		domain.ExchangeBybit:   partial,
		domain.ExchangeOKX:     partial,
	}
	held, err := New(boardCandles{bars: bars}, nil).
		WithOpenInterest(svc.oi).
		WithFundingRate(svc.funding).
		WithTakerFlow(partialPorts).
		GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	if held.Coins[0].CVDSplitDefinite || held.Coins[0].CVDSplit != "" {
		t.Fatalf("short combined leaked %q", held.Coins[0].CVDSplit)
	}
	var binanceKind, binanceStatus string
	for _, venue := range held.Coins[0].Venues {
		if venue.Exchange == "binance" {
			binanceKind = venue.CVDSplit
			binanceStatus = venue.CVDSplitStatus
		}
	}
	if binanceStatus != domain.VenueReadFresh || binanceKind != domain.CVDDivPriceUpCVDDown {
		t.Fatalf("binance row %s %s", binanceStatus, binanceKind)
	}
}

func boardService(now time.Time, skipOKX bool) *Service {
	oi := &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Value: 2e9, Contracts: 2e9},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-1 * time.Hour), Value: 2e9, Contracts: 2e9},
			{Time: now.Add(-24 * time.Hour), Value: 1e9, Contracts: 1e9},
		},
	}
	fund := &domain.FundingSeries{
		Current: domain.FundingPoint{Time: now, Rate: 0.0001, MarkPrice: 100000},
	}
	oiPorts := map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeBinance: boardOI{series: oi},
		domain.ExchangeBybit:   boardOI{series: oi},
	}
	fundPorts := map[domain.Exchange]domain.FundingRatePort{
		domain.ExchangeBinance: boardFunding{series: fund},
		domain.ExchangeBybit:   boardFunding{series: fund},
	}
	if !skipOKX {
		oiPorts[domain.ExchangeOKX] = boardOI{series: oi}
		fundPorts[domain.ExchangeOKX] = boardFunding{series: fund}
	}
	return New(nil, nil).WithOpenInterest(oiPorts).WithFundingRate(fundPorts)
}

type boardArchive struct {
	byEx map[domain.Exchange]domain.FuturesSnapshot
}

func (a boardArchive) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	if q.Metric != domain.FuturesMetricOpenInterest {
		return []domain.FuturesSnapshot{}, nil
	}
	row, ok := a.byEx[domain.Exchange(q.Exchange)]
	if !ok || row.SampledAt.IsZero() {
		return []domain.FuturesSnapshot{}, nil
	}
	if !q.From.IsZero() && row.SampledAt.Before(q.From) {
		return []domain.FuturesSnapshot{}, nil
	}
	if !q.To.IsZero() && row.SampledAt.After(q.To) {
		return []domain.FuturesSnapshot{}, nil
	}
	if q.Symbol != "" && row.Symbol != "" && row.Symbol != q.Symbol {
		return []domain.FuturesSnapshot{}, nil
	}
	return []domain.FuturesSnapshot{row}, nil
}

func boardCoinRow(got *domain.DerivativesBoard, symbol string) *domain.DerivativesCoin {
	if got == nil {
		return nil
	}
	for i := range got.Coins {
		if got.Coins[i].Symbol == symbol {
			return &got.Coins[i]
		}
	}
	return nil
}

func TestGetDerivativesBoard_ArchiveFills24hOpenInterest(t *testing.T) {
	now := time.Now().UTC()
	live := &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Value: 2e9, Contracts: 2e9},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Value: 2e9, Contracts: 2e9},
		},
	}
	fund := &domain.FundingSeries{
		Current: domain.FundingPoint{Time: now, Rate: 0.0001, MarkPrice: 100000},
	}
	oiPorts := map[domain.Exchange]domain.OpenInterestPort{}
	fundPorts := map[domain.Exchange]domain.FundingRatePort{}
	for _, ex := range domain.FuturesVenues() {
		oiPorts[ex] = boardOI{series: live}
		fundPorts[ex] = boardFunding{series: fund}
	}
	archiveAt := now.Add(-24 * time.Hour)
	points := map[domain.Exchange]domain.FuturesSnapshot{}
	for _, ex := range domain.FuturesVenues() {
		points[ex] = domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: "BTCUSDT",
			SampledAt: archiveAt, Value: 1e9, Contracts: 1e9,
		}
	}
	svc := New(nil, nil).WithOpenInterest(oiPorts).WithFundingRate(fundPorts)
	svc.SetFuturesHistory(boardArchive{byEx: points})
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(live.History) != 1 {
		t.Fatalf("live history mutated: %+v", live.History)
	}
	row := boardCoinRow(got, "BTCUSDT")
	if row == nil || !row.OpenInterestDefinite || !row.Change1hDefinite || row.Change1h != "0" {
		t.Fatalf("1h %+v", row)
	}
	if !row.Change24hDefinite || row.Change24h != "+3000000000" || row.Change24hPct != "+100" || len(row.Change24hMissing) != 0 {
		t.Fatalf("24h %+v missing %+v", row.Change24h, row.Change24hMissing)
	}
}

func TestGetDerivativesBoard_MissingArchiveNamesTheVenue(t *testing.T) {
	now := time.Now().UTC()
	live := &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Value: 2e9, Contracts: 2e9},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Value: 2e9, Contracts: 2e9},
		},
	}
	fund := &domain.FundingSeries{
		Current: domain.FundingPoint{Time: now, Rate: 0.0001, MarkPrice: 100000},
	}
	oiPorts := map[domain.Exchange]domain.OpenInterestPort{}
	fundPorts := map[domain.Exchange]domain.FundingRatePort{}
	for _, ex := range domain.FuturesVenues() {
		oiPorts[ex] = boardOI{series: live}
		fundPorts[ex] = boardFunding{series: fund}
	}
	archiveAt := now.Add(-24 * time.Hour)
	points := map[domain.Exchange]domain.FuturesSnapshot{
		domain.ExchangeBinance: {
			Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			SampledAt: archiveAt, Value: 1e9, Contracts: 1e9,
		},
		domain.ExchangeBybit: {
			Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT",
			SampledAt: archiveAt, Value: 1e9, Contracts: 1e9,
		},
	}
	svc := New(nil, nil).WithOpenInterest(oiPorts).WithFundingRate(fundPorts)
	svc.SetFuturesHistory(boardArchive{byEx: points})
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := boardCoinRow(got, "BTCUSDT")
	if row == nil || !row.OpenInterestDefinite || !row.Change1hDefinite {
		t.Fatalf("current %+v", row)
	}
	if row.Change24hDefinite || row.Change24h != "" || len(row.Change24hMissing) != 1 || row.Change24hMissing[0] != string(domain.ExchangeOKX) {
		t.Fatalf("24h %q missing %+v", row.Change24h, row.Change24hMissing)
	}
}

func TestGetDerivativesBoard_ArchiveOutsideSlackStaysBlank(t *testing.T) {
	now := time.Now().UTC()
	live := &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Value: 2e9, Contracts: 2e9},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Value: 2e9, Contracts: 2e9},
		},
	}
	fund := &domain.FundingSeries{
		Current: domain.FundingPoint{Time: now, Rate: 0.0001, MarkPrice: 100000},
	}
	oiPorts := map[domain.Exchange]domain.OpenInterestPort{}
	fundPorts := map[domain.Exchange]domain.FundingRatePort{}
	for _, ex := range domain.FuturesVenues() {
		oiPorts[ex] = boardOI{series: live}
		fundPorts[ex] = boardFunding{series: fund}
	}
	tooOld := now.Add(-26 * time.Hour)
	points := map[domain.Exchange]domain.FuturesSnapshot{}
	for _, ex := range domain.FuturesVenues() {
		points[ex] = domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: "BTCUSDT",
			SampledAt: tooOld, Value: 1e9, Contracts: 1e9,
		}
	}
	svc := New(nil, nil).WithOpenInterest(oiPorts).WithFundingRate(fundPorts)
	svc.SetFuturesHistory(boardArchive{byEx: points})
	got, err := svc.GetDerivativesBoard(context.Background(), "all", 5)
	if err != nil {
		t.Fatal(err)
	}
	row := boardCoinRow(got, "BTCUSDT")
	if row == nil || row.Change24hDefinite || row.Change24h != "" || len(row.Change24hMissing) != 3 {
		t.Fatalf("24h %q missing %+v", row.Change24h, row.Change24hMissing)
	}
	if !row.Change1hDefinite {
		t.Fatal("1h should still use the live sample")
	}
}
