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
		default:
			t.Fatalf("path %s", r.URL.Path)
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

func TestGetBasisQuote_BadSymbol(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetBasisQuote(context.Background(), " ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
