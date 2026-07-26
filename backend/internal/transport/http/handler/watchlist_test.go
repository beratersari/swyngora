package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

func newWatchHandler() *WatchlistHandler {
	return NewWatchlistHandler(watchlist.New(watchliststore.NewMemory()))
}

func TestWatchlistHTTP_AddGetRemove(t *testing.T) {
	h := newWatchHandler()

	// Add 6 items
	syms := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT"}
	for _, s := range syms {
		body, _ := json.Marshal(map[string]string{
			"clientId": "test-client",
			"exchange": "binance",
			"symbol":   s,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.Add(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("add %s status=%d body=%s", s, rr.Code, rr.Body.String())
		}
	}

	// Get — must be 6
	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=test-client", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d", rr.Code)
	}
	var got watchlistDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 6 {
		t.Fatalf("want 6 items, got %d", len(got.Items))
	}

	// Remove one — 5 remain
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/watchlist/items?clientId=test-client&exchange=binance&symbol=DOGEUSDT", nil)
	rr = httptest.NewRecorder()
	h.Remove(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d", rr.Code)
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if len(got.Items) != 5 {
		t.Fatalf("want 5 after delete, got %d", len(got.Items))
	}
}

func TestWatchlistHTTP_ReplaceStable(t *testing.T) {
	h := newWatchHandler()
	payload := map[string]any{
		"clientId": "c2",
		"items": []map[string]string{
			{"exchange": "binance", "symbol": "BTCUSDT"},
			{"exchange": "bybit", "symbol": "ETHUSDT"},
			{"exchange": "binance", "symbol": "SOLUSDT"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watchlist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Replace(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d %s", rr.Code, rr.Body.String())
	}
	var got watchlistDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if len(got.Items) != 3 {
		t.Fatalf("want 3 got %d", len(got.Items))
	}

	// Second replace same set — still 3
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/watchlist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Replace(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if len(got.Items) != 3 {
		t.Fatalf("unstable replace: %d", len(got.Items))
	}
}

func TestWatchlistHTTP_HeaderClientID(t *testing.T) {
	h := newWatchHandler()
	body, _ := json.Marshal(map[string]string{"exchange": "binance", "symbol": "BTCUSDT"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "hdr-client")
	rr := httptest.NewRecorder()
	h.Add(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/watchlist", nil)
	req.Header.Set("X-Client-Id", "hdr-client")
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	var got watchlistDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.ClientID != "hdr-client" || len(got.Items) != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestWatchlistHTTP_BadJSON(t *testing.T) {
	h := newWatchHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/items", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Add(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestWatchlistHTTP_EmptyClientIDRejected(t *testing.T) {
	h := newWatchHandler()
	// No clientId in body or header.
	body, _ := json.Marshal(map[string]string{"exchange": "binance", "symbol": "BTCUSDT"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Add(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty clientId status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Explicit shared name "default" rejected.
	body, _ = json.Marshal(map[string]string{
		"clientId": "default", "exchange": "binance", "symbol": "BTCUSDT",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.Add(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("default clientId status=%d body=%s", rr.Code, rr.Body.String())
	}

	// GET without clientId rejected.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/watchlist", nil)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("get empty clientId status=%d", rr.Code)
	}
}

func TestWatchlistHTTP_CoinbaseSymbolNormalized(t *testing.T) {
	h := newWatchHandler()
	body, _ := json.Marshal(map[string]string{
		"clientId": "coin-client", "exchange": "coinbase", "symbol": "BTCUSD",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Add(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}
	var got watchlistDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if len(got.Items) != 1 || got.Items[0].Symbol != "BTC-USD" {
		t.Fatalf("want BTC-USD stored, got %+v", got.Items)
	}
}

func TestWatchlistHTTP_BodyTooLarge(t *testing.T) {
	h := newWatchHandler()
	// > 1 MiB body should fail decode
	big := bytes.Repeat([]byte("x"), DefaultMaxJSONBody+100)
	payload := append([]byte(`{"clientId":"c","items":[{"symbol":"`), big...)
	payload = append(payload, []byte(`"}]}`)...)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watchlist", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Replace(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("oversized body status=%d", rr.Code)
	}
}
