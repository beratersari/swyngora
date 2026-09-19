package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestListPrints_PagesAggTrades(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		start := r.URL.Query().Get("startTime")
		if start == "" {
			t.Fatal("expected startTime")
		}
		t0 := int64(1_700_000_000_000)
		if n == 1 {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"p": "100", "q": "1", "T": t0, "m": false},
				{"p": "100", "q": "1", "T": t0 + 1000, "m": true},
			})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	from := time.UnixMilli(1_700_000_000_000).UTC()
	to := from.Add(time.Minute)
	got, err := c.ListPrints(context.Background(), "BTCUSDT", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%+v pages=%d", got, n)
	}
}
