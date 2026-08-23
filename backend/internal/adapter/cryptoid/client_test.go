package cryptoid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetHolders_ParsesAddressesAndRich(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "q=addresses"):
			_ = json.NewEncoder(w).Encode(map[string]any{"known": 1_700_000, "nonzero": 120_664})
		case strings.Contains(r.URL.RawQuery, "q=rich"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 100_000_000,
				"rich1000": []map[string]any{
					{"addr": "Anonymous", "amount": 999_999_999},
					{"addr": "DU8gPC5mh4KxWJARQRxoESFark2jAguBr5", "amount": 12_913_466},
					{"addr": "DHGuebakhGk4DEUMpEtHndPf2W3XW6VY2L", "amount": 1_185_547},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetHolders(context.Background(), "PIVXUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "PIVX" || got.HolderCount != 120_664 || got.Source != "cryptoid" {
		t.Fatalf("%+v", got)
	}
	if len(got.TopHolders) != 2 || got.TopHolders[0].Address == "Anonymous" {
		t.Fatalf("list=%+v", got.TopHolders)
	}
	if got.TopHolders[0].SharePct < 12 || got.TopHolders[0].SharePct > 13 {
		t.Fatalf("share=%v", got.TopHolders[0].SharePct)
	}
}

func TestGetHolders_MissingCoin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetHolders(context.Background(), "NOCOIN")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
