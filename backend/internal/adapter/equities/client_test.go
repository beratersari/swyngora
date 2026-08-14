package equities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestNasdaqListAndTicker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/v7/finance/spark") {
			_, _ = w.Write([]byte(`{"spark":{"result":[{"symbol":"AAPL","response":[{"meta":{
				"symbol":"AAPL","instrumentType":"EQUITY","shortName":"Apple Inc.",
				"regularMarketPrice":190.5,"regularMarketDayHigh":191,"regularMarketDayLow":188.5,
				"regularMarketVolume":50000000,"regularMarketTime":1700000000,"chartPreviousClose":189.25
			}}]}]}}`))
			return
		}
		if strings.Contains(r.URL.RawQuery, "download=true") || strings.Contains(r.URL.Path, "screener") || r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"data":{"rows":[{"symbol":"AAPL","name":"Apple","lastsale":"$190.50","netchange":"1.25","pctchange":"0.66%","volume":"50000000","marketCap":"2,900,000,000,000","sector":"Technology"}]}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewNasdaq(Options{
		BaseURL:        srv.URL,
		NasdaqScreener: srv.URL + "/screener?download=true",
		Universe:       []string{"AAPL"},
	})
	rows, err := c.ListSpotMarkets(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("list: %+v %v", rows, err)
	}
	if rows[0].Symbol != "AAPL" || rows[0].QuoteAsset != "USD" || rows[0].LastPrice != "190.5" {
		t.Fatalf("row=%+v", rows[0])
	}
	if rows[0].MarketCapCirculating == nil || *rows[0].MarketCapCirculating < 1e12 {
		t.Fatalf("mcap=%v", rows[0].MarketCapCirculating)
	}
	tk, err := c.GetTicker24h(context.Background(), "aapl")
	if err != nil || tk.LastPrice != "190.5" || tk.Symbol != "AAPL" {
		t.Fatalf("ticker %+v %v", tk, err)
	}
}

func TestBistUniverseFromPublicList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "hisse/list") {
			_, _ = w.Write([]byte(`{"code":"0","data":[{"kod":"THYAO","tip":"Hisse"},{"kod":"GARAN","tip":"Hisse"}]}`))
			return
		}
		if strings.Contains(r.URL.Path, "/v7/finance/spark") {
			_, _ = w.Write([]byte(`{"spark":{"result":[
				{"symbol":"THYAO.IS","response":[{"meta":{"symbol":"THYAO.IS","regularMarketPrice":280.4,"regularMarketVolume":100,"chartPreviousClose":282.5}}]},
				{"symbol":"GARAN.IS","response":[{"meta":{"symbol":"GARAN.IS","regularMarketPrice":120,"regularMarketVolume":50,"chartPreviousClose":118}}]}
			]}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := NewBist(Options{BaseURL: srv.URL, BistListURL: srv.URL + "/hisse/list", BistScreener: srv.URL + "/no-screener", Universe: []string{"THYAO"}})
	rows, err := c.ListSpotMarkets(context.Background())
	if err != nil || len(rows) != 2 {
		t.Fatalf("list %+v %v", rows, err)
	}
}

func TestBistYahooSuffixAndLocalSymbol(t *testing.T) {
	var gotSymbols string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSymbols = r.URL.Query().Get("symbols")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"spark":{"result":[{"symbol":"THYAO.IS","response":[{"meta":{
			"symbol":"THYAO.IS","instrumentType":"EQUITY",
			"regularMarketPrice":280.4,"regularMarketVolume":12000000,"chartPreviousClose":282.5
		}}]}]}}`))
	}))
	defer srv.Close()

	c := NewBist(Options{BaseURL: srv.URL, BistListURL: srv.URL + "/no-list", BistScreener: srv.URL + "/no-screener", Universe: []string{"THYAO"}})
	rows, err := c.ListSpotMarkets(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("list: %+v %v", rows, err)
	}
	if gotSymbols != "THYAO.IS" {
		t.Fatalf("yahoo symbols=%q", gotSymbols)
	}
	if rows[0].Symbol != "THYAO" || rows[0].QuoteAsset != "TRY" {
		t.Fatalf("row=%+v", rows[0])
	}
}

func TestBistScreenerListUsesMarketCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalCount":2,"data":[
			{"s":"BIST:THYAO","d":["THYAO",305.25,-0.89,-2.75,33086500,425040000000,"Transportation","THY"]},
			{"s":"BIST:LINK","d":["LINK",5.94,3.66,0.21,46816417,5109755845,"Technology Services","LINK BILGISAYAR"]}
		]}`))
	}))
	defer srv.Close()
	c := NewBist(Options{BistScreener: srv.URL, Universe: []string{"THYAO"}})
	rows, err := c.ListSpotMarkets(context.Background())
	if err != nil || len(rows) != 2 {
		t.Fatalf("list %+v %v", rows, err)
	}
	by := map[string]domain.SpotMarket{}
	for _, r := range rows {
		by[r.Symbol] = r
	}
	if by["THYAO"].MarketCapCirculating == nil || *by["THYAO"].MarketCapCirculating < 4e11 {
		t.Fatalf("thyao mcap=%v", by["THYAO"].MarketCapCirculating)
	}
	if by["LINK"].MarketCapCirculating == nil || *by["LINK"].MarketCapCirculating > 6e9 {
		t.Fatalf("link should be BIST mcap not crypto: %v", by["LINK"].MarketCapCirculating)
	}
	if len(by["LINK"].Tags) != 1 || by["LINK"].Tags[0] != "Technology Services" {
		t.Fatalf("link tags=%v", by["LINK"].Tags)
	}
}

func TestGetOrderBookUnsupported(t *testing.T) {
	c := NewNasdaq(Options{BaseURL: "http://127.0.0.1:9", Universe: []string{"AAPL"}})
	_, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "AAPL"})
	if err == nil || !strings.Contains(err.Error(), "order book") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseChartTrimsLimit(t *testing.T) {
	body := []byte(`{"chart":{"result":[{"timestamp":[1,2,3],"indicators":{"quote":[{
		"open":[1,2,3],"high":[1,2,3],"low":[1,2,3],"close":[10,20,30],"volume":[1,1,1]
	}]}}]}}`)
	bars, err := parseChart(body, 2)
	if err != nil || len(bars) != 2 {
		t.Fatalf("%+v %v", bars, err)
	}
	if bars[0].Close != "20" || bars[1].Close != "30" {
		t.Fatalf("bars=%+v", bars)
	}
}
