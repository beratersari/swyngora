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
		BaseURL:        srv.URL,
		ProductBaseURL: srv.URL,
		HTTPClient:     srv.Client(),
		APIKey:         "test-key",
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

func TestParseBinanceWillDelistTitle(t *testing.T) {
	tokens, when, ok := parseBinanceWillDelistTitle("Binance Will Delist ACX, HFT, PIVX, PYR, VANRY, VIC on 2026-08-17")
	if !ok {
		t.Fatal("expected parse")
	}
	if !when.Equal(time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("when=%v", when)
	}
	if len(tokens) != 6 || tokens[0] != "ACX" || tokens[5] != "VIC" {
		t.Fatalf("tokens=%v", tokens)
	}
	if _, _, ok := parseBinanceWillDelistTitle("Notice of Removal of Spot Trading Pairs - 2026-08-21"); ok {
		t.Fatal("pair-removal notice has no tokens")
	}
	if _, _, ok := parseBinanceWillDelistTitle("Binance Margin And Loan Will Delist BTTC & POWR on 2026-08-14"); ok {
		t.Fatal("margin/loan should skip")
	}
}

func TestFetchSpotDelistScheduleMergesCMSPastWillDelist(t *testing.T) {
	past := time.Now().UTC().Add(-10 * 24 * time.Hour).Format("2006-01-02")
	future := time.Now().UTC().Add(12 * 24 * time.Hour).Format("2006-01-02")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sapi/v1/spot/delist-schedule":
			_, _ = w.Write([]byte(`[{"delistTime":1786928400000,"symbols":["ICXUSDT"]}]`))
		case cmsDelistCatalogPath:
			body := `{"code":"000000","data":{"catalogs":[{"catalogId":161,"articles":[` +
				`{"title":"Binance Will Delist ICX, SCRT, STORJ on ` + future + `"},` +
				`{"title":"Binance Will Delist ACX, HFT on ` + past + `"},` +
				`{"title":"Notice of Removal of Spot Trading Pairs - 2026-08-21"}` +
				`]}]}}`
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{
		BaseURL:        srv.URL,
		ProductBaseURL: srv.URL,
		HTTPClient:     srv.Client(),
		APIKey:         "test-key",
	})
	entries, err := c.FetchSpotDelistSchedule(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]time.Time{}
	for _, e := range entries {
		found[e.Symbol] = e.DelistTime
	}
	if !found["ICXUSDT"].Equal(time.UnixMilli(1786928400000).UTC()) {
		t.Fatalf("official time should win for ICX: %v", found["ICXUSDT"])
	}
	if _, ok := found["ACXUSDT"]; !ok {
		t.Fatalf("expected CMS ACXUSDT in %v", found)
	}
	if _, ok := found["HFTUSDT"]; !ok {
		t.Fatalf("expected CMS HFTUSDT in %v", found)
	}
}
