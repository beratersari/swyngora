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
