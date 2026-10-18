package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFuturesDepthReadsThePublicBook(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch body["type"] {
		case "metaAndAssetCtxs":
			_, _ = w.Write([]byte(`[{"universe":[{"name":"BTC"}]},[{"funding":"0.0001","openInterest":"1","markPx":"100"}]]`))
			return
		case "l2Book":
		default:
			t.Fatalf("body %+v", body)
		}
		if body["coin"] != "BTC" {
			t.Fatalf("coin %+v", body)
		}
		_, _ = w.Write([]byte(`{
			"coin":"BTC","time":1700000000000,
			"levels":[
				[{"px":"100","sz":"2","n":3},{"px":"99","sz":"1","n":1},{"px":"0","sz":"1","n":1}],
				[{"px":"101","sz":"4","n":2}]
			]
		}`))
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	book, err := c.GetFuturesDepth(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(book.Bids) != 2 || book.Bids[0].Price != 100 || book.Bids[0].Quantity != 2 || book.Bids[1].Price != 99 {
		t.Fatalf("bids %+v", book.Bids)
	}
	if len(book.Asks) != 1 || book.Asks[0].Price != 101 || book.Asks[0].Quantity != 4 {
		t.Fatalf("asks %+v", book.Asks)
	}
	if _, err := c.GetFuturesDepth(context.Background(), "BTCUSDT"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("cache missed, calls %d", calls)
	}
	if _, err := c.GetFuturesDepth(context.Background(), "ETHBTC"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
}

func TestFuturesDepthNamesAnEmptyBook(t *testing.T) {
	l2 := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch body["type"] {
		case "metaAndAssetCtxs":
			_, _ = w.Write([]byte(`[{"universe":[{"name":"DOGE"}]},[{"funding":"0.0001","openInterest":"1","markPx":"1"}]]`))
		case "l2Book":
			l2++
			_, _ = w.Write([]byte(`{"coin":"DOGE","time":1,"levels":[[],[]]}`))
		default:
			t.Fatalf("body %+v", body)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetFuturesDepth(context.Background(), "DOGEUSDT")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if l2 != 1 {
		t.Fatalf("l2 calls %d", l2)
	}
	_, err = c.GetFuturesDepth(context.Background(), "SOLUSDT")
	if !errors.Is(err, domain.ErrNotFound) || l2 != 1 {
		t.Fatalf("unlisted coin called the book: %v calls %d", err, l2)
	}
}

func TestFuturesDepthStopsOnRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetFuturesDepth(context.Background(), "BTCUSDT")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatal(err)
	}
}
