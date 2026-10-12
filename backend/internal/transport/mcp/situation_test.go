package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

func TestAPIClient_GetCoinSituation_SendsAtOnlyForAPastRead(t *testing.T) {
	var gotAt string
	var sawAt bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/situation" {
			http.NotFound(w, r)
			return
		}
		gotAt = r.URL.Query().Get("at")
		_, sawAt = r.URL.Query()["at"]
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": r.URL.Query().Get("symbol")})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL, 0)
	if _, err := c.GetCoinSituation(context.Background(), "BTCUSDT", "  "); err != nil {
		t.Fatal(err)
	}
	if sawAt {
		t.Fatalf("live query sent at=%q", gotAt)
	}
	if _, err := c.GetCoinSituation(context.Background(), "BTCUSDT", " 2026-06-01T16:00:00Z "); err != nil {
		t.Fatal(err)
	}
	if gotAt != "2026-06-01T16:00:00Z" {
		t.Fatalf("at=%q", gotAt)
	}
}

func TestBackend_GetCoinSituation_PastClockLeavesThePocketOff(t *testing.T) {
	b := &Backend{Market: market.New(nil, nil)}
	raw, err := b.GetCoinSituation(context.Background(), "BTCUSDT", "2026-06-01T16:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["liquidation"]; ok {
		t.Fatalf("liquidation present: %s", body["liquidation"])
	}
	if _, ok := body["liquidationMissing"]; ok {
		t.Fatalf("liquidationMissing present: %s", body["liquidationMissing"])
	}
	if !strings.Contains(string(body["note"]), "not part of this read") {
		t.Fatalf("note=%s", body["note"])
	}
	if _, err := b.GetCoinSituation(context.Background(), "BTCUSDT", "not-a-clock"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
