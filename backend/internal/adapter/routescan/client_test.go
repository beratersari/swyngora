package routescan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFromContracts_CITY(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/evm/88888/") {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Query().Get("action") {
		case "tokenholdercount":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "1", "result": "1775"})
		case "tokeninfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1",
				"result": []map[string]any{{
					"tokenName": "Manchester City FC", "symbol": "CITY",
					"divisor": "18", "totalSupply": "100000000000000000000",
				}},
			})
		case "tokenholderlist":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1",
				"result": []map[string]any{{
					"TokenHolderAddress": "0xabc", "TokenHolderQuantity": "10000000000000000000",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FromContracts(context.Background(), "CITY", []domain.AssetContract{
		{Chain: "Chiliz", Address: "0x7bd6242d775faef1d50b2aa18c2fbf329bddf295"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HolderCount != 1775 || got.Source != "routescan" || got.Name != "Manchester City FC" {
		t.Fatalf("%+v", got)
	}
	if len(got.TopHolders) != 1 || got.TopHolders[0].SharePct != 10 {
		t.Fatalf("top=%+v", got.TopHolders)
	}
}
