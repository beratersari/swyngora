package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchSpotDelistScheduleRequiresAPIKey(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.FetchSpotDelistSchedule(context.Background())
	if err == nil {
		t.Fatal("expected error without API key")
	}
}

func TestFetchSpotDelistScheduleParsesRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sapi/v1/spot/delist-schedule" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-MBX-APIKEY") != "test-key" {
			http.Error(w, `{"code":-2014,"msg":"API-key format invalid."}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Live Binance uses "symbols" (plural).
		_, _ = w.Write([]byte(`[
			{"delistTime":1786928400000,"symbols":["ACXUSDT","HFTUSDT"]},
			{"delistTime":1787014800000,"symbols":["PIVXUSDT"]}
		]`))
	}))
	defer srv.Close()

	c := NewClient(Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		APIKey:     "test-key",
	})
	entries, err := c.FetchSpotDelistSchedule(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("len=%d entries=%+v", len(entries), entries)
	}
	want := time.UnixMilli(1786928400000).UTC()
	if !entries[0].DelistTime.Equal(want) {
		t.Fatalf("time=%v want %v", entries[0].DelistTime, want)
	}
	found := map[string]bool{}
	for _, e := range entries {
		found[e.Symbol] = true
	}
	if !found["ACXUSDT"] || !found["HFTUSDT"] || !found["PIVXUSDT"] {
		t.Fatalf("symbols=%v", found)
	}
}
