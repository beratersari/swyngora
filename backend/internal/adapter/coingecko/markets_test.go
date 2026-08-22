package coingecko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupplyBySymbolsExactMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != marketsPath {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbols") != "hft,vanry" {
			t.Fatalf("symbols=%s", r.URL.Query().Get("symbols"))
		}
		_, _ = w.Write([]byte(`[
			{"id":"hashflow","symbol":"hft","name":"Hashflow","current_price":0.0065,"circulating_supply":900000000,"total_supply":1000000000,"max_supply":1000000000,"market_cap":5850000},
			{"id":"vanar-chain-2","symbol":"vanry","name":"Vanar","current_price":0.0008,"circulating_supply":3800000000,"total_supply":10000000000,"max_supply":10000000000,"market_cap":3040000}
		]`))
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.SupplyBySymbols(context.Background(), []string{"HFT", "vanry", "HFT"})
	if err != nil {
		t.Fatal(err)
	}
	if got["HFT"] == nil || got["HFT"].CirculatingSupply == nil || *got["HFT"].CirculatingSupply != 900000000 {
		t.Fatalf("HFT %+v", got["HFT"])
	}
	if got["VANRY"] == nil || got["VANRY"].Source != "coingecko" {
		t.Fatalf("VANRY %+v", got["VANRY"])
	}
}
