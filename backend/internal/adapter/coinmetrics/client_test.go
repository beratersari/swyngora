package coinmetrics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetHolders_ParsesAdrBalCnt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("assets") != "eth" || r.URL.Query().Get("metrics") != "AdrBalCnt" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"asset": "eth", "time": "2026-08-22T00:00:00.000000000Z", "AdrBalCnt": "204136337",
			}},
		})
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetHolders(context.Background(), "ETHUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.HolderCount != 204136337 || got.Source != "coinmetrics" || got.Asset != "ETH" {
		t.Fatalf("%+v", got)
	}
}

func TestGetHolders_ForbiddenIsUnpublished(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"type":"forbidden"}}`))
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetHolders(context.Background(), "SOL")
	if err == nil {
		t.Fatal("expected miss")
	}
}

func TestGetHolders_BadRequestIsUnpublished(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"bad_request","message":"Unknown assets"}}`))
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetHolders(context.Background(), "CITY")
	if err == nil {
		t.Fatal("expected miss")
	}
	if !errors.Is(err, domain.ErrHoldersUnpublished) {
		t.Fatalf("err=%v", err)
	}
}
