package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/exportstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

// HTTP reproduction of finding 1: concurrent POST /portfolio/margin/orders.
func TestVerifyHTTP_ConcurrentMarginOpensCash(t *testing.T) {
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	px := &mutPx{prices: map[string]string{"BTCUSDT": "100"}}
	svc := portfolio.New(st, px).WithPaperCosts(domain.ZeroTradingCosts)
	h := NewPortfolioHandler(svc)

	post := func(path string, body map[string]any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Client-Id", "http-race")
		rr := httptest.NewRecorder()
		switch path {
		case "/api/v1/portfolio":
			h.Create(rr, req)
		default:
			h.PlaceMarginOrder(rr, req)
		}
		return rr
	}

	rr := post("/api/v1/portfolio", map[string]any{"clientId": "http-race", "startingBalance": 35})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	var start sync.WaitGroup
	start.Add(1)
	var wg sync.WaitGroup
	var opened atomic.Int64
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start.Wait()
			res := post("/api/v1/portfolio/margin/orders", map[string]any{
				"clientId": "http-race", "symbol": "BTCUSDT", "side": "long", "type": "market",
				"quantity": 1, "leverage": 5,
			})
			if res.Code == http.StatusCreated || res.Code == http.StatusOK {
				opened.Add(1)
			}
		}()
	}
	start.Done()
	wg.Wait()

	view, err := svc.View(context.Background(), "http-race")
	if err != nil {
		t.Fatal(err)
	}
	n := int(opened.Load())
	if n > 1 {
		t.Errorf("CONFIRMED finding 1 (HTTP): %d margin opens from 35 cash (cash now %v)", n, view.CashBalance)
		return
	}
	t.Logf("NOT REPRODUCED finding 1 via HTTP: opened=%d cash=%v", n, view.CashBalance)
}

// HTTP reproduction of finding 3: X-Client-Id: .. + POST /export writes outside FileDir.
func TestVerifyHTTP_ExportDotDotClientEscapesSandbox(t *testing.T) {
	root := t.TempDir()
	fileDir := filepath.Join(root, "exports")
	wl := watchliststore.NewMemory()
	svc, err := exportsvc.New(exportstore.NewMemory(), exportsvc.DataSources{
		Watchlist: wl, Alerts: emptyAlerts{}, Scanner: emptyScanner{},
	}, exportsvc.Options{FileDir: fileDir, FileTTL: 0})
	if err != nil {
		t.Fatal(err)
	}
	h := NewExportHandler(svc)

	body, _ := json.Marshal(map[string]any{"format": "json", "sections": []string{"watchlist"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "..")
	rr := httptest.NewRecorder()
	h.Start(rr, req)
	if rr.Code == http.StatusAccepted {
		t.Fatalf("export Start must reject clientId .., got %d %s", rr.Code, rr.Body.String())
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("start %d %s", rr.Code, rr.Body.String())
	}
}
