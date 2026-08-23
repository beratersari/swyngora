package coingecko

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLookupContracts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/api/v3/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"coins": []map[string]any{{"id": "pepe", "symbol": "pepe", "name": "Pepe"}},
			})
		case strings.Contains(r.URL.Path, "/api/v3/coins/pepe"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "Pepe",
				"platforms": map[string]string{
					"ethereum":            "0x6982",
					"binance-smart-chain": "0x25d8",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.LookupContracts(context.Background(), "PEPEUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
}
