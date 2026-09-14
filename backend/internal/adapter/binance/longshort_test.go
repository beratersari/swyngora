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

func TestGetLongShortSeries_OK(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("period") != "5m" || r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		switch r.URL.Path {
		case "/futures/data/globalLongShortAccountRatio":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"longAccount": "0.60", "shortAccount": "0.40", "longShortRatio": "1.5", "timestamp": now.Add(-5 * time.Minute).UnixMilli()},
				{"longAccount": "0.63", "shortAccount": "0.37", "longShortRatio": "1.7027", "timestamp": now.UnixMilli()},
			})
		case "/futures/data/topLongShortAccountRatio":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"longAccount": "0.55", "shortAccount": "0.45", "longShortRatio": "1.222", "timestamp": now.UnixMilli()},
			})
		case "/futures/data/topLongShortPositionRatio":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"longAccount": "0.32", "shortAccount": "0.68", "longShortRatio": "0.4706", "timestamp": now.UnixMilli()},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetLongShortSeries(context.Background(), "btc-usd", 24)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.LongShare != 0.63 || got.Current.Ratio < 1.70 {
		t.Fatalf("current %+v", got.Current)
	}
	if len(got.History) != 1 || got.History[0].LongShare != 0.60 {
		t.Fatalf("hist %+v", got.History)
	}
	if got.TopAccounts == nil || got.TopAccounts.Current.LongShare != 0.55 {
		t.Fatalf("top accounts %+v", got.TopAccounts)
	}
	if got.TopPositions == nil || got.TopPositions.Current.LongShare != 0.32 {
		t.Fatalf("top positions %+v", got.TopPositions)
	}
	if got.Split.Alignment != domain.AlignOpposite || !got.Split.Important {
		t.Fatalf("split %+v", got.Split)
	}
}

func TestGetLongShortSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{FuturesBaseURL: "http://example.invalid"})
	_, err := c.GetLongShortSeries(context.Background(), "  ", 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
