package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type anchorFutHist struct{}

func (anchorFutHist) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	anchor := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	if q.Metric == "liquidations" {
		return []domain.LiquidationEvent{{
			Exchange: domain.ExchangeBinance, Time: anchor.Add(-time.Minute), Side: "short", Notional: 3,
		}}, nil
	}
	if q.Metric == domain.FuturesMetricOpenInterest {
		return []domain.FuturesSnapshot{{
			Exchange: domain.ExchangeBinance, SampledAt: anchor.Add(-time.Hour), Value: 20,
		}, {
			Exchange: domain.ExchangeBinance, SampledAt: anchor.Add(time.Hour), Value: 25,
		}}, nil
	}
	return []domain.FuturesSnapshot{}, nil
}

func TestGetPositioningAnchor_OK(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{})
	svc.SetFuturesHistory(anchorFutHist{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/positioning-anchor?symbol=BTCUSDT&anchor=2026-10-08T12:00:00Z&bucket=5m", nil)
	rr := httptest.NewRecorder()
	h.GetPositioningAnchor(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body domain.PositioningAnchor
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || len(body.OpenInterest) != 1 || body.OpenInterest[0].Change != "5" {
		t.Fatalf("%+v", body)
	}
	if len(body.Liquidations) != 1 || !body.Liquidations[0].Complete || body.Liquidations[0].Change != "0" {
		t.Fatalf("liq %+v", body.Liquidations)
	}
	if len(body.MissingFunding) != 3 {
		t.Fatalf("funding %v", body.MissingFunding)
	}
}

func TestGetPositioningAnchor_RequiresAnchor(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{})
	svc.SetFuturesHistory(anchorFutHist{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/positioning-anchor?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetPositioningAnchor(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
