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

func TestGetLongShortSeries_CrowdAndTop(t *testing.T) {
	rubikGap = 0
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	t0 := now.UnixMilli()
	t1 := now.Add(-5 * time.Minute).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("instId") != "BTC-USDT-SWAP" || r.URL.Query().Get("period") != "5m" {
			t.Fatalf("query %s %s", r.URL.Path, r.URL.RawQuery)
		}
		var ratio string
		switch r.URL.Path {
		case okxLSAccounts:
			ratio = "1"
		case okxLSTopAcc:
			ratio = "1.5"
		case okxLSTopPos:
			ratio = "3"
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": [][]string{
				{itoa(t0), ratio},
				{itoa(t1), ratio},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetLongShortSeries(context.Background(), "BTCUSDT", 24)
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != domain.ExchangeOKX || got.Current.LongShare < 0.49 || got.Current.LongShare > 0.51 {
		t.Fatalf("crowd %+v", got.Current)
	}
	if got.TopPositions == nil || got.TopPositions.Current.LongShare < 0.74 {
		t.Fatalf("top size %+v", got.TopPositions)
	}
	if got.TopAccounts == nil || got.TopAccounts.Current.LongShare < 0.59 {
		t.Fatalf("top accounts %+v", got.TopAccounts)
	}
	if len(got.History) != 1 {
		t.Fatalf("history %d", len(got.History))
	}
}

func TestListLongShortHistory_MergesTopSize(t *testing.T) {
	rubikGap = 0
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	hour := now.Add(-time.Hour).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("period") != "1H" {
			t.Fatalf("period %s", r.URL.RawQuery)
		}
		ratio := "1"
		if r.URL.Path == okxLSTopPos {
			ratio = "3"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": [][]string{{itoa(hour), ratio}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	from := now.Add(-2 * time.Hour)
	got, err := c.ListLongShortHistory(context.Background(), "btcusdt", from, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("rows %d", len(got))
	}
	if got[0].LongShare < 0.49 || got[0].LongShare > 0.51 {
		t.Fatalf("crowd %+v", got[0])
	}
	if got[0].SizeLongShare < 0.74 {
		t.Fatalf("size share %v", got[0].SizeLongShare)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
