package fx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestUSD_ParsesRates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"base":"USD","date":"2026-08-14","rates":{"TRY":40.5,"EUR":0.92}}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.Client()).WithBaseURL(srv.URL)
	rates, asOf, err := c.LatestUSD(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rates["TRY"] != 40.5 || rates["EUR"] != 0.92 || rates["USD"] != 1 || rates["USDT"] != 1 {
		t.Fatalf("%v", rates)
	}
	if asOf.Format("2006-01-02") != "2026-08-14" {
		t.Fatalf("asOf=%v", asOf)
	}
}

func TestLatestUSD_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	c := New(srv.Client()).WithBaseURL(srv.URL)
	if _, _, err := c.LatestUSD(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
