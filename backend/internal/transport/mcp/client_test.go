package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClient_GetTicker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/ticker/24h" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("symbol=%s", r.URL.Query().Get("symbol"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "lastPrice": "100", "exchange": "binance",
		})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetTicker(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_argument","message":"bad"}}`))
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	_, err := c.GetSupply(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPrettyJSON(t *testing.T) {
	raw := json.RawMessage(`{"a":1}`)
	out := PrettyJSON(raw)
	if out == "" || out[0] != '{' {
		t.Fatalf("%q", out)
	}
}

func TestNewServer_RegistersTools(t *testing.T) {
	s := NewServer(ServerOptions{APIBaseURL: "http://127.0.0.1:9"})
	if s == nil {
		t.Fatal("nil server")
	}
}

func TestInProcessServerAndHTTPHandler(t *testing.T) {
	// Smoke: construct in-process server without market services using HTTP fallback is already tested.
	// NewHTTPHandler must be a valid http.Handler.
	s := NewServer(ServerOptions{APIBaseURL: "http://127.0.0.1:9"})
	h := NewHTTPHandler(s)
	if h == nil {
		t.Fatal("nil handler")
	}
}
