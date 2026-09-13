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

func TestGetFundingSeries_OK(t *testing.T) {
	next := time.Date(2026, 9, 13, 16, 0, 0, 0, time.UTC)
	cur := next.Add(-8 * time.Hour)
	prev := cur.Add(-8 * time.Hour)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v5/public/funding-rate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0",
				"data": []map[string]any{{
					"fundingRate":     "0.0001",
					"fundingTime":     ms(cur),
					"nextFundingTime": ms(next),
					"prevFundingTime": ms(prev),
					"ts":              ms(cur.Add(time.Minute)),
				}},
			})
		case "/api/v5/public/funding-rate-history":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0",
				"data": []map[string]any{{
					"fundingRate": "0.00005", "realizedRate": "0.00005", "fundingTime": ms(cur),
				}},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetFundingSeries(context.Background(), "BTCUSDT", 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != domain.ExchangeOKX || got.Current.Rate != 0.0001 || !got.Current.Predicted {
		t.Fatalf("current %+v", got.Current)
	}
	if got.IntervalHours != 8 {
		t.Fatalf("interval %d", got.IntervalHours)
	}
	if len(got.History) != 1 || got.History[0].Rate != 0.00005 {
		t.Fatalf("hist %+v", got.History)
	}
}

func ms(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}
