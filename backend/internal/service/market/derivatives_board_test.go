package market

import (
	"context"
	"errors"
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
