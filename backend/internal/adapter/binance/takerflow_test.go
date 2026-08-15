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

func TestGetTakerFlow_OK(t *testing.T) {
	now := time.Now().UTC()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/futures/data/takerlongshortRatio":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"buyVol": "2", "sellVol": "1", "timestamp": now.Add(-2 * time.Minute).UnixMilli()},
				{"buyVol": "3", "sellVol": "8", "timestamp": now.Add(-20 * time.Minute).UnixMilli()},
			})
		case "/fapi/v1/premiumIndex":
			_ = json.NewEncoder(w).Encode(map[string]any{"markPrice": "100"})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetTakerFlow(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Windows[0].BuyNotional != 200 { // 2 * 100
		t.Fatalf("5m buy %+v", got.Windows[0])
	}
	if got.Windows[0].Dominant != domain.TakerSideBuy {
		t.Fatalf("dom %s", got.Windows[0].Dominant)
	}
}

func TestGetTakerFlow_BadSymbol(t *testing.T) {
	c := NewClient(Options{FuturesBaseURL: "http://example.invalid"})
	_, err := c.GetTakerFlow(context.Background(), " ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
