package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestRecentTradesSeedBuyAndSell(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch body["type"] {
		case "metaAndAssetCtxs":
			_, _ = w.Write([]byte(`[{"universe":[{"name":"ETH"}]},[{"funding":"0.0001","openInterest":"1","markPx":"2000"}]]`))
			return
		case "recentTrades":
		default:
			t.Fatalf("body %+v", body)
		}
		if body["coin"] != "ETH" {
			t.Fatalf("coin %+v", body)
		}
		buyAt := time.Now().UTC().Add(-2 * time.Second).UnixMilli()
		sellAt := time.Now().UTC().Add(-time.Second).UnixMilli()
		_, _ = w.Write([]byte(`[
			{"coin":"ETH","side":"B","px":"2000","sz":"0.5","time":` + strconv.FormatInt(buyAt, 10) + `,"tid":1},
			{"coin":"ETH","side":"A","px":"2000","sz":"0.25","time":` + strconv.FormatInt(sellAt, 10) + `,"tid":2},
			{"coin":"@1","side":"B","px":"1","sz":"1","time":` + strconv.FormatInt(sellAt, 10) + `,"tid":3},
			{"coin":"ETH","side":"X","px":"2000","sz":"1","time":` + strconv.FormatInt(sellAt, 10) + `,"tid":4}
		]`))
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	prints, err := c.GetRecentPrints(context.Background(), "ETHUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(prints) != 2 || prints[0].Side != domain.TakerSideBuy || prints[0].Notional != 1000 || prints[1].Side != domain.TakerSideSell || prints[1].Notional != 500 {
		t.Fatalf("%+v", prints)
	}
	if prints[0].Exchange != domain.ExchangeHyperliquid || prints[0].Symbol != "ETHUSDT" {
		t.Fatalf("name %+v", prints[0])
	}
	buckets, err := c.GetTakerBuckets(context.Background(), "ETHUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 || buckets[0].BuyNotional != 1000 || buckets[0].SellNotional != 500 || !buckets[0].SizeKnown {
		t.Fatalf("buckets %+v", buckets)
	}
	if calls != 2 {
		t.Fatalf("seeded twice, calls %d", calls)
	}
	flow, err := c.GetTakerFlow(context.Background(), "ETHUSDT")
	if err != nil || flow == nil || len(flow.Windows) == 0 || flow.Windows[0].Complete {
		t.Fatalf("a short tape must stay incomplete %+v %v", flow, err)
	}
}

func TestRecentTradesEmptyIsNotZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch body["type"] {
		case "metaAndAssetCtxs":
			_, _ = w.Write([]byte(`[{"universe":[{"name":"DOGE"}]},[{"funding":"0.0001","openInterest":"1","markPx":"1"}]]`))
		case "recentTrades":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("body %+v", body)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetRecentPrints(context.Background(), "DOGEUSDT")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if got := c.TakerBook().Buckets(domain.ExchangeHyperliquid, "DOGEUSDT"); len(got) != 0 {
		t.Fatalf("empty tape stored %+v", got)
	}
}

func TestTradeMessageRecordsTheAggressor(t *testing.T) {
	book := domain.NewTakerBook()
	tape := domain.NewPrintTape()
	h := NewTradeHub(TradeHubOptions{Book: book, Tape: tape})
	ts := time.Now().UTC().Add(-time.Second).UnixMilli()
	msg := []byte(`{"channel":"trades","data":[
		{"coin":"BTC","side":"A","px":"64000","sz":"0.01","time":` + strconv.FormatInt(ts, 10) + `,"tid":9},
		{"coin":"PURR/USDC","side":"B","px":"1","sz":"1","time":` + strconv.FormatInt(ts, 10) + `,"tid":10}
	]}`)
	for _, p := range parseTradeMessage(msg) {
		domain.RecordMeasuredPrint(book, tape, p)
	}
	for _, p := range parseTradeMessage(msg) {
		domain.RecordMeasuredPrint(book, tape, p)
	}
	got := book.Buckets(domain.ExchangeHyperliquid, "BTCUSDT")
	if len(got) != 1 || got[0].SellNotional != 640 {
		t.Fatalf("%+v", got)
	}
	if len(tape.Prints(domain.ExchangeHyperliquid, "BTCUSDT")) != 1 {
		t.Fatal("duplicate print stored")
	}
	if len(book.Buckets(domain.ExchangeHyperliquid, "PURRUSDT")) != 0 {
		t.Fatal("spot pair stored")
	}
	if h.wsBase == "" {
		t.Fatal("hub")
	}
}
