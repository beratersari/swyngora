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

func TestListAggTradeWindow_PagesWithoutCache(t *testing.T) {
	var starts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/aggTrades" {
			t.Errorf("path %s", r.URL.Path)
		}
		starts = append(starts, r.URL.Query().Get("startTime"))
		if len(starts) == 1 {
			t0 := int64(1_700_000_000_000)
			rows := make([]map[string]any, 1000)
			for i := range rows {
				rows[i] = map[string]any{"p": "100", "q": "1", "T": t0 + int64(i), "m": false}
			}
			_ = json.NewEncoder(w).Encode(rows)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"p": "100", "q": "2", "T": int64(1_700_000_002_000), "m": true},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	from := time.UnixMilli(1_700_000_000_000).UTC()
	to := from.Add(time.Minute)
	got, err := c.ListAggTradeWindow(context.Background(), "BTCUSDT", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1001 || got[len(got)-1].Side != domain.TakerSideSell {
		t.Fatalf("len %d last %+v", len(got), got[len(got)-1])
	}
	if _, err := c.ListAggTradeWindow(context.Background(), "BTCUSDT", from.Add(time.Hour), from.Add(time.Hour+time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(starts) < 3 || starts[2] == starts[0] {
		t.Fatalf("windows were cached: %v", starts)
	}
}

func TestListAggTradeWindow_RejectsLongWindow(t *testing.T) {
	c := NewClient(Options{})
	from := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	_, err := c.ListAggTradeWindow(context.Background(), "BTCUSDT", from, from.Add(time.Hour))
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err %v", err)
	}
}

func TestListAggTradeWindow_FullBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, _ := strconvParse(r.URL.Query().Get("startTime"))
		rows := make([]map[string]any, 1000)
		for i := range rows {
			rows[i] = map[string]any{"p": "100", "q": "1", "T": start + int64(i+1), "m": false}
		}
		_ = json.NewEncoder(w).Encode(rows)
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	from := time.UnixMilli(1_700_000_000_000).UTC()
	_, err := c.ListAggTradeWindow(context.Background(), "BTCUSDT", from, from.Add(time.Minute))
	if !errors.Is(err, domain.ErrTapeWindowFull) {
		t.Fatalf("err %v", err)
	}
}

func strconvParse(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("nan")
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
