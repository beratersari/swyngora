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
