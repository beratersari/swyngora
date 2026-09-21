package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type stubOptionsChain struct{}

func (stubOptionsChain) GetOptionQuotes(_ context.Context, asset string) ([]domain.OptionQuote, error) {
	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Hour)
	return []domain.OptionQuote{{
		Asset: asset, Expiry: exp, Strike: 100000, Side: domain.OptionSideCall,
		OpenInterest: 10, Volume: 1, MarkIV: 50, Underlying: 100000,
	}, {
		Asset: asset, Expiry: exp, Strike: 100000, Side: domain.OptionSidePut,
		OpenInterest: 5, Volume: 2, MarkIV: 52, Underlying: 100000,
	}}, nil
}

func (stubOptionsChain) GetOptionsDVOL(context.Context, string) (domain.OptionsDVOL, error) {
	return domain.OptionsDVOL{Value: 49.5, Change24h: -1.1, HasChange: true}, nil
}

func (stubOptionsChain) GetOptionsIndex(context.Context, string) (float64, error) {
	return 100000, nil
}

func TestGetOptions_OK(t *testing.T) {
	h := NewMarketHandler(market.New(stubMarket{}, stubSupply{}).WithOptionsChain(stubOptionsChain{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/options?asset=BTC", nil)
	rr := httptest.NewRecorder()
	h.GetOptions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body optionsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Asset != "BTC" || body.Exchange != "deribit" || body.OpenInterest == "" || body.Dvol == "" {
		t.Fatalf("%+v", body)
	}
	if body.Atm == nil || len(body.Expiries) != 1 || len(body.TermStructure) != 1 {
		t.Fatalf("%+v", body)
	}
	if body.Skew == nil || body.Skew.AtmSkew == "" || body.TermStructure[0].Skew == "" {
		t.Fatalf("vol %+v", body)
	}
	if body.Expiries[0].ExpectedMovePct == "" || body.Expiries[0].MaxPain == "" {
		t.Fatalf("expiry move/pain %+v", body.Expiries[0])
	}
	if body.Gamma == nil || body.Gamma.NetUsd == "" || body.Gamma.Regime == "" {
		t.Fatalf("gamma %+v", body.Gamma)
	}
	if body.Expiries[0].GammaNetUsd == "" {
		t.Fatalf("expiry gamma %+v", body.Expiries[0])
	}
	if body.VsMarket != nil && len(body.VsMarket.Findings) > 0 {
		t.Fatalf("stub tape should not invent vs-market findings %+v", body.VsMarket)
	}
}

type dailyVolMarket struct{ stubMarket }

func (dailyVolMarket) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	px := 80000.0
	out := make([]domain.Candle, 0, 40)
	for i := 0; i < 40; i++ {
		if i%3 == 0 {
			px *= 1.02
		} else if i%3 == 1 {
			px *= 0.985
		} else {
			px *= 1.004
		}
		s := fmt.Sprintf("%.4f", px)
		t := now.AddDate(0, 0, i-39)
		out = append(out, domain.Candle{OpenTime: t, Open: s, High: s, Low: s, Close: s})
	}
	return out, nil
}

func TestGetOptions_VolCompare(t *testing.T) {
	h := NewMarketHandler(market.New(dailyVolMarket{}, stubSupply{}).WithOptionsChain(stubOptionsChain{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/options?asset=BTC", nil)
	rr := httptest.NewRecorder()
	h.GetOptions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body optionsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.VolCompare == nil || body.VolCompare.Dvol == "" || body.VolCompare.Realized == "" || body.VolCompare.Gap == "" {
		t.Fatalf("volCompare=%+v", body.VolCompare)
	}
	if body.VolCompare.Days != 30 {
		t.Fatalf("days=%d", body.VolCompare.Days)
	}
}

func TestGetOptions_BadAsset(t *testing.T) {
	h := NewMarketHandler(market.New(stubMarket{}, stubSupply{}).WithOptionsChain(stubOptionsChain{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/options?asset=SOL", nil)
	rr := httptest.NewRecorder()
	h.GetOptions(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
