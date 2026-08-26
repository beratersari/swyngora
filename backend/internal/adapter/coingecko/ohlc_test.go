package coingecko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuoteByBase_UsesMarketsPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == marketsPath:
			_, _ = w.Write([]byte(`[{"id":"viction","symbol":"vic","name":"Viction","current_price":0.12,"price_change_24h":-0.00435,"price_change_percentage_24h":-3.5,"circulating_supply":97000000,"total_supply":210000000,"max_supply":210000000}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	q, err := c.QuoteByBase(context.Background(), "VIC")
	if err != nil {
		t.Fatal(err)
	}
	if q.LastUSD != 0.12 || q.ChangePct == nil || *q.ChangePct != -3.5 {
		t.Fatalf("quote=%+v", q)
	}
	if q.ChangeAbs == nil || *q.ChangeAbs != -0.00435 {
		t.Fatalf("delta=%+v", q.ChangeAbs)
	}
}

func TestQuoteByBase_PriceWithoutSupply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == marketsPath:
			_, _ = w.Write([]byte(`[{"id":"ghost","symbol":"xyz","name":"Ghost","current_price":1.5,"price_change_percentage_24h":1.1}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	q, err := c.QuoteByBase(context.Background(), "XYZ")
	if err != nil {
		t.Fatal(err)
	}
	if q.LastUSD != 1.5 {
		t.Fatalf("quote=%+v", q)
	}
}

func TestOHLCByBase_MapsRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == marketsPath:
			_, _ = w.Write([]byte(`[{"id":"viction","symbol":"vic","name":"Viction","current_price":0.12,"circulating_supply":1,"total_supply":1}]`))
		case strings.Contains(r.URL.Path, "/ohlc"):
			_, _ = w.Write([]byte(`[[1723852800000,0.10,0.14,0.09,0.12]]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	bars, err := c.OHLCByBase(context.Background(), "VIC", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 1 || bars[0].Open != "0.1" || bars[0].Close != "0.12" {
		t.Fatalf("bars=%+v", bars)
	}
}
