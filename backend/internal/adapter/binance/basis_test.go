package binance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetBasisQuote_OK(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fapi/v1/premiumIndex":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"markPrice": "64080", "indexPrice": "64000", "time": now.UnixMilli(),
			})
		case "/fapi/v1/ticker/price":
			_ = json.NewEncoder(w).Encode(map[string]any{"price": "64100"})
		case "/api/v3/ticker/price":
			_ = json.NewEncoder(w).Encode(map[string]any{"price": "64010"})
		case "/fapi/v1/markPriceKlines":
			_ = json.NewEncoder(w).Encode([][]any{
				{float64(now.Add(-time.Hour).UnixMilli()), "0", "0", "0", "64020"},
			})
		case "/fapi/v1/indexPriceKlines":
			if r.URL.Query().Get("pair") == "" {
				http.Error(w, `{"code":-1102,"msg":"Mandatory parameter 'pair' was not sent"}`, http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode([][]any{
				{float64(now.Add(-time.Hour).UnixMilli()), "0", "0", "0", "64000"},
			})
		case "/fapi/v1/exchangeInfo":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetBasisQuote(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.FuturesLast != 64100 || got.SpotIndex != 64000 || got.FuturesMark != 64080 {
		t.Fatalf("%+v", got)
	}
	if len(got.History) != 1 || got.History[0].BasisPct <= 0 {
		t.Fatalf("hist %+v", got.History)
	}
}

func TestGetBasisQuote_Curve(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	near := now.Add(40 * 24 * time.Hour)
	far := now.Add(130 * 24 * time.Hour)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fapi/v1/premiumIndex":
			if r.URL.Query().Get("symbol") != "" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"markPrice": "64000", "indexPrice": "64000", "time": now.UnixMilli(),
				})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"symbol": "BTCUSDT", "markPrice": "64000", "indexPrice": "64000"},
				{"symbol": "BTCUSDT_260925", "markPrice": "64200", "indexPrice": "64000"},
				{"symbol": "BTCUSDT_261225", "markPrice": "64600", "indexPrice": "64000"},
			})
		case "/fapi/v1/ticker/price":
			if r.URL.Query().Get("symbol") != "" {
				_ = json.NewEncoder(w).Encode(map[string]any{"price": "64000"})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"symbol": "BTCUSDT", "price": "64000"},
				{"symbol": "BTCUSDT_260925", "price": "64200"},
				{"symbol": "BTCUSDT_261225", "price": "64600"},
			})
		case "/api/v3/ticker/price":
			_ = json.NewEncoder(w).Encode(map[string]any{"price": "64000"})
		case "/fapi/v1/markPriceKlines", "/fapi/v1/indexPriceKlines":
			_ = json.NewEncoder(w).Encode([][]any{})
		case "/fapi/v1/exchangeInfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"symbols": []map[string]any{
					{"symbol": "BTCUSDT", "pair": "BTCUSDT", "contractType": "PERPETUAL", "status": "TRADING", "deliveryDate": 4133404800000},
					{"symbol": "BTCUSDT_260925", "pair": "BTCUSDT", "contractType": "CURRENT_QUARTER", "status": "TRADING", "deliveryDate": near.UnixMilli()},
					{"symbol": "BTCUSDT_261225", "pair": "BTCUSDT", "contractType": "NEXT_QUARTER", "status": "TRADING", "deliveryDate": far.UnixMilli()},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetBasisQuote(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Contracts) < 3 {
		t.Fatalf("contracts %+v", got.Contracts)
	}
}

func TestGetBasisQuote_BadSymbol(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetBasisQuote(context.Background(), " ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
