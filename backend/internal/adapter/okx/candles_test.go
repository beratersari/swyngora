package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetFuturesCandles_SwapOHLC(t *testing.T) {
	newer := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	older := newer.Add(-15 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/candles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("instId") != "BTC-USDT-SWAP" || q.Get("bar") != "15m" || q.Get("limit") != "110" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		row := func(ts time.Time, close string) []string {
			return []string{strconv.FormatInt(ts.UnixMilli(), 10), "100", "110", "90", close, "1", "1", "1", "1"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": [][]string{row(newer, "105"), row(older, "101")},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetFuturesCandles(context.Background(), domain.CandleQuery{
		Symbol: "btcusdt", Interval: domain.Interval15m, Limit: 110,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Close != "101" || got[1].Close != "105" {
		t.Fatalf("%+v", got)
	}
	if !got[0].CloseTime.Equal(older.Add(15 * time.Minute)) {
		t.Fatalf("close time %v", got[0].CloseTime)
	}
}

func TestGetFuturesCandles_RejectsUnknownBar(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetFuturesCandles(context.Background(), domain.CandleQuery{
		Symbol: "BTCUSDT", Interval: "9h",
	})
	if err == nil {
		t.Fatal("expected invalid interval")
	}
}

func TestOKXCandleBar_UTCDay(t *testing.T) {
	bar, ok := okxCandleBar("1d")
	if !ok || bar != "1Dutc" {
		t.Fatalf("%s %v", bar, ok)
	}
	if _, ok := okxCandleBar("3d"); ok {
		t.Fatal("3d is not an OKX bar this adapter requests")
	}
}
