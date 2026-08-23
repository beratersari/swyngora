package ethplorer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFromContracts_ParsesInfoAndTop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/getTokenInfo/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "Uniswap", "symbol": "UNI", "decimals": "18", "holdersCount": 12,
			})
		case strings.Contains(r.URL.Path, "/getTopTokenHolders/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"holders": []map[string]any{
					{"address": "0xabc", "balance": 1e20, "share": 26.7},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FromContracts(context.Background(), "UNI", []domain.AssetContract{
		{Chain: "ethereum", Address: "0xuni"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HolderCount != 12 || got.Source != "ethplorer" || len(got.TopHolders) != 1 {
		t.Fatalf("%+v", got)
	}
	if got.TopHolders[0].SharePct != 26.7 {
		t.Fatalf("share=%v", got.TopHolders[0].SharePct)
	}
}
