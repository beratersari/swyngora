package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetRecentPrints_TakerSide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/recent-trade" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"list": []map[string]any{
					{"side": "Buy", "price": "100", "size": "10", "time": "1700000000000"},
					{"side": "Sell", "price": "99", "size": "5", "time": "1700000001000"},
				},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetRecentPrints(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Side != domain.TakerSideBuy || got[1].Side != domain.TakerSideSell {
		t.Fatalf("%+v", got)
	}
	if got[0].Notional != 1000 {
		t.Fatalf("notional %+v", got[0])
	}
}

func TestGetRecentPrints_BadSymbol(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetRecentPrints(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
