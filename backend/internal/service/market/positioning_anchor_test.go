package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type positioningHist struct {
	queries []domain.FuturesHistoryQuery
}

func (p *positioningHist) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	p.queries = append(p.queries, q)
	anchor := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	switch q.Metric {
	case domain.FuturesMetricOpenInterest:
		return []domain.FuturesSnapshot{{
			Exchange: domain.ExchangeBinance, SampledAt: anchor.Add(-time.Hour), Value: 10,
		}, {
			Exchange: domain.ExchangeBinance, SampledAt: anchor.Add(time.Hour), Value: 16,
		}}, nil
	case domain.FuturesMetricFunding:
		return []domain.FuturesSnapshot{{
			Exchange: domain.ExchangeBybit, SampledAt: anchor.Add(time.Hour), FundingRate: 0.0001, Predicted: true,
		}}, nil
	case "liquidations":
		return []domain.LiquidationEvent{{
			Exchange: domain.ExchangeBinance, Time: anchor.Add(-time.Minute), Side: "short", Notional: 2,
		}, {
			Exchange: domain.ExchangeBinance, Time: anchor.Add(time.Minute), Side: "long", Notional: 1,
		}}, nil
	default:
		return nil, errors.New("unexpected metric")
	}
}

func TestGetPositioningAnchor_UsesTheChartWindow(t *testing.T) {
	hist := &positioningHist{}
	svc := New(&fakeMarket{}, &fakeSupply{})
	svc.SetFuturesHistory(hist)
	anchor := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	from := anchor.Add(-2 * time.Hour)
	got, err := svc.GetPositioningAnchor(context.Background(), "all", "btcusdt", anchor.Format(time.RFC3339), "5m", &from, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist.queries) != 3 || hist.queries[0].Bucket != 5*time.Minute || hist.queries[0].Symbol != "BTCUSDT" {
		t.Fatalf("queries %+v", hist.queries)
	}
	if len(got.OpenInterest) != 1 || got.OpenInterest[0].Change != "6" {
		t.Fatalf("oi %+v", got.OpenInterest)
	}
	if len(got.MissingFunding) != 3 {
		t.Fatalf("funding should stay off: %+v missing %v", got.Funding, got.MissingFunding)
	}
	if len(got.Liquidations) != 1 || !got.Liquidations[0].Complete || got.Liquidations[0].Change != "-1" {
		t.Fatalf("liq %+v", got.Liquidations)
	}
}

func TestGetPositioningAnchor_RequiresHistoryAndAnchor(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetPositioningAnchor(context.Background(), "all", "BTCUSDT", time.Now().UTC().Format(time.RFC3339), "5m", nil, nil); err == nil {
		t.Fatal("expected unconfigured history")
	}
	svc.SetFuturesHistory(&positioningHist{})
	if _, err := svc.GetPositioningAnchor(context.Background(), "all", "BTCUSDT", "", "5m", nil, nil); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty anchor: %v", err)
	}
}
