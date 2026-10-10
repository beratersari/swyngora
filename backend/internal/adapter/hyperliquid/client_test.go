package hyperliquid

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

func TestCoinMapsUSDTNamesOnly(t *testing.T) {
	coin, ok := Coin("btcusdt")
	if !ok || coin != "BTC" {
		t.Fatalf("btc %q %v", coin, ok)
	}
	// NormalizeLiquidationSymbol folds a trailing USD into USDT, same as the other perp names.
	coin, ok = Coin("BTC-USD")
	if !ok || coin != "BTC" {
		t.Fatalf("usd fold %q %v", coin, ok)
	}
	if _, ok := Coin("BTCUSDC"); ok {
		t.Fatal("a USDC ticker is not the USDT name")
	}
	if _, ok := Coin("ETHBTC"); ok {
		t.Fatal("a BTC-quoted name is not the USDT name")
	}
}

func TestBookReadsMarkOpenInterestAndHourlyFunding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" || r.Method != http.MethodPost {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch body["type"] {
		case "metaAndAssetCtxs":
			_, _ = w.Write([]byte(`[
				{"universe":[{"name":"BTC"},{"name":"ETH"},{"name":"SKIP"}]},
				[
					{"funding":"-0.000005","openInterest":"2","markPx":"100"},
					{"funding":"0.00001","openInterest":"4","markPx":"50"},
					{"funding":"0","openInterest":"0","markPx":"1"}
				]
			]`))
		case "fundingHistory":
			_, _ = w.Write([]byte(`[
				{"coin":"ETH","fundingRate":"0.0000125","premium":"0","time":1700000000000},
				{"coin":"ETH","fundingRate":"-0.000001","premium":"0","time":1700003600000}
			]`))
		default:
			t.Fatalf("type %v", body["type"])
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})

	oi, err := c.GetOpenInterestSeries(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if oi.Exchange != domain.ExchangeHyperliquid || oi.Current.Contracts != 2 || oi.Current.Value != 200 {
		t.Fatalf("oi %+v", oi.Current)
	}
	if len(oi.History) != 0 {
		t.Fatal("open interest history is not published")
	}
	fund, err := c.GetFundingSeries(context.Background(), "BTCUSDT", 1)
	if err != nil {
		t.Fatal(err)
	}
	if fund.Current.Rate != -0.000005 || fund.Current.MarkPrice != 100 || !fund.Current.Predicted || fund.IntervalHours != 1 {
		t.Fatalf("funding %+v hours %d", fund.Current, fund.IntervalHours)
	}
	if _, err := c.GetOpenInterestSeries(context.Background(), "DOGEUSDT"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}

	from := time.UnixMilli(1700000000000)
	to := time.UnixMilli(1700003600000)
	hist, err := c.ListFundingHistory(context.Background(), "ETHUSDT", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 || hist[0].Rate != 0.0000125 || hist[1].Time.UnixMilli() != 1700003600000 {
		t.Fatalf("history %+v", hist)
	}
}

func TestFundingHistoryStopsOnAShortPage(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`[{"fundingRate":"0.0001","time":1000}]`))
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	hist, err := c.ListFundingHistory(context.Background(), "BTCUSDT", time.UnixMilli(0), time.UnixMilli(5000))
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 || calls != 1 {
		t.Fatalf("hist %d calls %d", len(hist), calls)
	}
}
