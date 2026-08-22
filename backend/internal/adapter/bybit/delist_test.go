package bybit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestParseDelistTimeAndSymbols(t *testing.T) {
	text := "Delisting 6 Spot trading pairs on May 19, 2026. Affected: ABCUSDT, XYZ/USDT and FOO-USDC."
	when, ok := parseDelistTime(text)
	if !ok {
		t.Fatal("expected date")
	}
	want := time.Date(2026, time.May, 19, 0, 0, 0, 0, time.UTC)
	if !when.Equal(want) {
		t.Fatalf("when=%v want %v", when, want)
	}
	syms := parseDelistSymbols(text)
	if len(syms) != 3 {
		t.Fatalf("syms=%v", syms)
	}
}

func TestParseDelistTimeWithClock(t *testing.T) {
	text := "Spot Trading of PUMPBTCUSDT will end at Jul 20, 2026, 9:00AM UTC"
	when, ok := parseDelistTime(text)
	if !ok {
		t.Fatal("expected date")
	}
	want := time.Date(2026, time.July, 20, 9, 0, 0, 0, time.UTC)
	if !when.Equal(want) {
		t.Fatalf("when=%v want %v", when, want)
	}
}

func TestParseDelistTimePrefersEndAfterOverPublishDate(t *testing.T) {
	text := "Aug 20, 2026 Aug 20, 2026 Delisted trading pairs: ELIZAOSUSDT,PRCLUSDT,TUSDUSDT " +
		"Spot : Trading of ELIZAOSUSDT,PRCLUSDT,TUSDUSDT pairs on Bybit’s Spot platform will end after Aug 27, 2026, 8:00AM UTC"
	when, ok := parseDelistTime(text)
	if !ok {
		t.Fatal("expected date")
	}
	want := time.Date(2026, time.August, 27, 8, 0, 0, 0, time.UTC)
	if !when.Equal(want) {
		t.Fatalf("when=%v want %v", when, want)
	}
	syms := parseDelistSymbols(text)
	if len(syms) != 3 {
		t.Fatalf("syms=%v", syms)
	}
}

func TestParseDelistTimeEightAMAndISO(t *testing.T) {
	when, ok := parseDelistTime("Trading of VANRY/USDT will end after Aug 18, 2026, 8AM UTC")
	if !ok || !when.Equal(time.Date(2026, time.August, 18, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("8AM when=%v ok=%v", when, ok)
	}
	iso, ok := parseDelistTime("we will delist the following Spot trading pairs on 2026-09-04 08:00:00 UTC: ETHMNT")
	if !ok || !iso.Equal(time.Date(2026, time.September, 4, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("iso when=%v ok=%v", iso, ok)
	}
}

func TestAnnouncementLooksLikeSpotDelist(t *testing.T) {
	if !announcementLooksLikeSpotDelist(announcementRow{
		Title: "Delisting 6 Spot trading pairs on May 19, 2026",
	}) {
		t.Fatal("spot delist title")
	}
	if announcementLooksLikeSpotDelist(announcementRow{
		Title: "Bybit will be delisting HIGHUSDT Perpetual Contract",
	}) {
		t.Fatal("perp-only should skip")
	}
	if announcementLooksLikeSpotDelist(announcementRow{
		Title: "Bybit Alpha to Delist 1 Token(s) on Aug 21, 2026, 11:00AM UTC",
	}) {
		t.Fatal("alpha should skip")
	}
}

func TestFetchSpotDelistSchedule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != announcementsPath {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "delistings" {
			t.Fatalf("type=%s", r.URL.Query().Get("type"))
		}
		_, _ = w.Write([]byte(`{"retCode":0,"retMsg":"OK","result":{"list":[
			{"title":"Delisting ABCUSDT spot pair on December 19, 2026","description":"Spot ABCUSDT","type":{"key":"delistings"},"tags":["Spot"]},
			{"title":"New Listing: FOO","description":"list FOOUSDT","type":{"key":"new_crypto"},"tags":["Spot"]}
		]}}`))
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FetchSpotDelistSchedule(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Symbol != "ABCUSDT" || got[0].Exchange != domain.ExchangeBybit {
		t.Fatalf("got=%+v", got)
	}
	want := time.Date(2026, time.December, 19, 0, 0, 0, 0, time.UTC)
	if !got[0].DelistTime.Equal(want) {
		t.Fatalf("time=%v", got[0].DelistTime)
	}
}

func TestFetchSpotDelistScheduleReadsPastDatedArticle(t *testing.T) {
	past := time.Now().UTC().Add(-12 * 24 * time.Hour)
	titleDate := past.Format("Jan 2, 2006")
	bodyDate := past.Format("Jan 2, 2006") + ", 8:00AM UTC"
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case announcementsPath:
			fmt.Fprintf(w, `{"retCode":0,"retMsg":"OK","result":{"list":[
				{"title":"Delisting of VANRY on %s","description":"","url":"%s/article/vanry","type":{"key":"delistings"},"tags":["Spot","Delistings"]}
			]}}`, titleDate, srv.URL)
		case "/article/vanry":
			fmt.Fprintf(w, `<html><body>Delisted Trading Pair VANRY/USDT Spot will end after %s</body></html>`, bodyDate)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FetchSpotDelistSchedule(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Symbol != "VANRYUSDT" {
		t.Fatalf("got=%+v", got)
	}
}

func TestFetchSpotDelistScheduleReadsArticleBody(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case announcementsPath:
			fmt.Fprintf(w, `{"retCode":0,"retMsg":"OK","result":{"list":[
				{"title":"Delisting of ELIZAOS,PRCL,TUSD","description":"Bybit is committed to maintaining a secure environment.","url":"%s/article/elizaos","type":{"key":"delistings"},"tags":["Delistings"]},
				{"title":"Delisting of VINEUSDT Perpetual Contract","description":"perp","url":"%s/article/vine","type":{"key":"delistings"},"tags":["Derivatives"]}
			]}}`, srv.URL, srv.URL)
		case "/article/elizaos":
			_, _ = w.Write([]byte(`<html><body><p>Aug 20, 2026</p>
				<p>Delisted trading pairs: ELIZAOSUSDT,PRCLUSDT,TUSDUSDT</p>
				<p>Spot : Trading will end after Aug 27, 2026, 8:00AM UTC</p></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.FetchSpotDelistSchedule(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got=%+v", got)
	}
	want := time.Date(2026, time.August, 27, 8, 0, 0, 0, time.UTC)
	for _, e := range got {
		if !e.DelistTime.Equal(want) {
			t.Fatalf("time=%v entry=%+v", e.DelistTime, e)
		}
	}
}
