package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetRecentPrints_TakerSide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/aggTrades" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"p": "100", "q": "10", "T": 1700000000000, "m": false},
			{"p": "99", "q": "5", "T": 1700000001000, "m": true},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
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
