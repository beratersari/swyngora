package binance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestRestGate_418StopsTheNextCall(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	q := url.Values{"symbol": []string{"BTCUSDT"}}
	if _, err := c.get(context.Background(), "/api/v3/ticker/24hr", q); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("first err=%v", err)
	}
	if !c.gate.paused() {
		t.Fatal("expected pause after 418")
	}
	if _, err := c.get(context.Background(), "/api/v3/ticker/24hr", q); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second err=%v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits=%d, banned IP was called again", hits.Load())
	}
}

func TestRestGate_WeightHeaderPauses(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("X-MBX-USED-WEIGHT-1M", "2400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","lastPrice":"1","priceChange":"0","priceChangePercent":"0","volume":"1","quoteVolume":"1","count":1}`))
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	q := url.Values{"symbol": []string{"BTCUSDT"}}
	if _, err := c.get(context.Background(), "/api/v3/ticker/24hr", q); err != nil {
		t.Fatal(err)
	}
	if !c.gate.paused() {
		t.Fatal("expected pause after high used weight")
	}
	if _, err := c.get(context.Background(), "/api/v3/ticker/24hr", q); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second err=%v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestRetryAfter_UsesHeaderWhenLonger(t *testing.T) {
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Retry-After", "120")
	if d := retryAfter(resp, time.Minute); d != 2*time.Minute {
		t.Fatalf("%s", d)
	}
	resp.Header.Set("Retry-After", "1")
	if d := retryAfter(resp, time.Minute); d != time.Minute {
		t.Fatalf("fallback %s", d)
	}
}
