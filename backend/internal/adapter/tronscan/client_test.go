package tronscan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFromContracts_JST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token_trc20" || r.URL.Query().Get("contract") != "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"trc20_tokens": []map[string]any{{
				"name": "JUST", "symbol": "JST", "holders_count": 441846,
				"contract_address": "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9",
			}},
		})
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FromContracts(context.Background(), "JSTUSDT", []domain.AssetContract{
		{Chain: "bsc", Address: "0xskip"},
		{Chain: "tron20", Address: "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HolderCount != 441846 || got.Source != "tronscan" || got.Asset != "JST" {
		t.Fatalf("%+v", got)
	}
}

func TestFromContracts_SkipsNonTron(t *testing.T) {
	c := New(Options{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{}})
	_, err := c.FromContracts(context.Background(), "UNI", []domain.AssetContract{
		{Chain: "ethereum", Address: "0xuni"},
	})
	if err == nil {
		t.Fatal("expected miss")
	}
}
