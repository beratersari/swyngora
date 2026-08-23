package geckoterminal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFromContracts_ParsesHolders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/networks/eth/tokens/0xuni/info") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"attributes": map[string]any{
					"name": "Uniswap",
					"holders": map[string]any{
						"count":                   388607,
						"distribution_percentage": map[string]any{"top_10": "58.31"},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FromContracts(context.Background(), "UNI", []domain.AssetContract{
		{Chain: "solana", Address: "skip"},
		{Chain: "ethereum", Address: "0xuni"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HolderCount != 388607 || got.Source != "geckoterminal" {
		t.Fatalf("%+v", got)
	}
	if got.TopTenSharePct == nil || *got.TopTenSharePct != 58.31 {
		t.Fatalf("top10=%v", got.TopTenSharePct)
	}
}

func TestGeckoNetwork(t *testing.T) {
	if geckoNetwork("Binance Smart Chain") != "bsc" {
		t.Fatal(geckoNetwork("Binance Smart Chain"))
	}
	if geckoNetwork("BNB Smart Chain (BEP20)") != "bsc" {
		t.Fatal(geckoNetwork("BNB Smart Chain (BEP20)"))
	}
	if geckoNetwork("ethereum") != "eth" {
		t.Fatal("eth")
	}
	if geckoNetwork("Optimism") != "optimism" {
		t.Fatal(geckoNetwork("Optimism"))
	}
	if geckoNetwork("tron20") != "tron" {
		t.Fatal(geckoNetwork("tron20"))
	}
	if geckoNetwork("zkSync Era") != "zksync" {
		t.Fatal(geckoNetwork("zkSync Era"))
	}
}
