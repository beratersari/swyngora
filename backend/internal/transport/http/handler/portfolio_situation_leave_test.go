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

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type leavePx struct{ last string }

func (p *leavePx) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p.last}, nil
}

type leaveSituation struct {
	got *domain.CoinSituation
}

func (s *leaveSituation) GetCoinSituation(context.Context, string) (*domain.CoinSituation, error) {
	return s.got, nil
}

func TestPortfolioHTTP_SituationLeave(t *testing.T) {
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	px := &leavePx{last: "100"}
	src := &leaveSituation{got: &domain.CoinSituation{
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc := portfolio.New(st, px).WithPaperCosts(domain.ZeroTradingCosts)
	svc.SetSituationSource(src)
	h := NewPortfolioHandler(svc)

	post := func(path string, payload map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		switch path {
		case "/api/v1/portfolio":
			h.Create(rr, req)
		default:
			h.PlaceOrder(rr, req)
		}
		return rr
	}
	if rr := post("/api/v1/portfolio", map[string]any{"clientId": "http-leave", "startingBalance": 100000}); rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	if rr := post("/api/v1/portfolio/orders", map[string]any{
		"clientId": "http-leave", "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	}); rr.Code != http.StatusOK {
		t.Fatalf("buy %d %s", rr.Code, rr.Body.String())
	}
	px.last = "80"
	src.got = &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown}}
	if rr := post("/api/v1/portfolio/orders", map[string]any{
		"clientId": "http-leave", "symbol": "BTCUSDT", "side": "sell", "quantity": 1,
	}); rr.Code != http.StatusOK {
		t.Fatalf("sell %d %s", rr.Code, rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/situation-leave?clientId=http-leave", nil)
	rr := httptest.NewRecorder()
	h.SituationLeave(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("leave %d %s", rr.Code, rr.Body.String())
	}
	var got domain.SituationLeave
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Counted != 1 || got.Skipped != 0 || got.Unreadable != 0 || len(got.Rows) != 1 {
		t.Fatalf("%+v", got)
	}
	row := got.Rows[0]
	if row.Clause != "priceCvd" || row.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("row %+v", row)
	}
	if row.Still.Count != 0 || len(row.Still.Closes) != 0 {
		t.Fatalf("still %+v", row.Still)
	}
	if row.Left.Count != 1 || row.Left.Profitable != 0 || math.Abs(row.Left.RealizedPnL-(-20)) > 1e-9 {
		t.Fatalf("left %+v", row.Left)
	}
	if len(row.Left.Closes) != 1 || row.Left.Closes[0].Symbol != "BTCUSDT" {
		t.Fatalf("closes %+v", row.Left.Closes)
	}
}
