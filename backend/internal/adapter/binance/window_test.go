package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetWindowChanges_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/ticker" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("windowSize") != "1h" {
			t.Fatalf("window %s", r.URL.Query().Get("windowSize"))
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"symbol": "BTCUSDT", "priceChangePercent": "1.25"},
			{"symbol": "ETHUSDT", "priceChangePercent": "-0.40"},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetWindowChanges(context.Background(), "1h", []string{"BTCUSDT", "ETHUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ChangePct != 1.25 || got[1].ChangePct != -0.40 {
		t.Fatalf("%+v", got)
	}
}

func TestGetWindowChanges_BadWindow(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetWindowChanges(context.Background(), "3d", []string{"BTCUSDT"})
	if err == nil {
		t.Fatal("expected error")
	}
}
