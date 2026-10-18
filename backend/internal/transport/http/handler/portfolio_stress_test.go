package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

func newStressHandler(t *testing.T, src portfolio.CandleSource) (*PortfolioHandler, *portfolio.Service) {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := portfolio.New(st, pfPx{}).WithPaperCosts(domain.ZeroTradingCosts)
	svc.SetCandleSource(src)
	return NewPortfolioHandler(svc), svc
}

func TestPortfolioHTTP_StressReplay(t *testing.T) {
	src := &stressHTTPCandles{bars: []domain.Candle{
		{OpenTime: time.Date(2024, 8, 4, 0, 0, 0, 0, time.UTC), Close: "100", CloseTime: time.Date(2024, 8, 4, 23, 59, 59, 0, time.UTC)},
		{OpenTime: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), Close: "80", CloseTime: time.Date(2024, 8, 6, 23, 59, 59, 0, time.UTC)},
	}}
	h, svc := newStressHandler(t, src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, portfolio.CreateInput{ClientID: "stress-http", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, portfolio.OrderInput{ClientID: "stress-http", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{"from": "2024-08-05", "to": "2024-08-06"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/stress", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "stress-http")
	rr := httptest.NewRecorder()
	h.StressPortfolio(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("replay %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["replay"]; !ok {
		t.Fatalf("missing replay %s", rr.Body.String())
	}
	replay := got["replay"].(map[string]any)
	moves := replay["moves"].([]any)
	row := moves[0].(map[string]any)
	if row["asset"] != "BTC" || math.Abs(row["movePct"].(float64)+20) > 1e-6 {
		t.Fatalf("move %+v", row)
	}
	if math.Abs(got["equityDelta"].(float64)+20) > 0.01 {
		t.Fatalf("delta %v", got["equityDelta"])
	}

	body, _ = json.Marshal(map[string]any{
		"from":  "2024-08-05",
		"to":    "2024-08-06",
		"moves": []map[string]any{{"asset": "BTC", "movePct": -10}},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/stress", bytes.NewReader(body))
	req.Header.Set("X-Client-Id", "stress-http")
	rr = httptest.NewRecorder()
	h.StressPortfolio(rr, req)
	if rr.Code != http.StatusBadRequest || !bytes.Contains(rr.Body.Bytes(), []byte("not both")) {
		t.Fatalf("both %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"moves": []map[string]any{{"asset": "BTC", "movePct": -10}}})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/stress", bytes.NewReader(body))
	req.Header.Set("X-Client-Id", "stress-http")
	rr = httptest.NewRecorder()
	h.StressPortfolio(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("typed %d %s", rr.Code, rr.Body.String())
	}
	got = map[string]any{}
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if _, ok := got["replay"]; ok {
		t.Fatalf("typed replay %s", rr.Body.String())
	}
	if math.Abs(got["equityDelta"].(float64)+10) > 0.01 {
		t.Fatalf("typed delta %v", got["equityDelta"])
	}
}

type stressHTTPCandles struct {
	bars []domain.Candle
}

func (s *stressHTTPCandles) GetCandles(context.Context, string, string, string, int, *time.Time, *time.Time) ([]domain.Candle, error) {
	return append([]domain.Candle(nil), s.bars...), nil
}
