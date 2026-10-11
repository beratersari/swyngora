package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestListLiquidationHistory_PerSymbol(t *testing.T) {
	from := time.Date(2026, 9, 13, 17, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Minute)
	at := from.Add(time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/public/liquidation-orders" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "filled" || r.URL.Query().Get("uly") != "BTC-USDT" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": []map[string]any{{
				"instId": "BTC-USDT-SWAP",
				"details": []map[string]any{{
					"bkPx": "77211.6", "sz": "48.9", "posSide": "long", "side": "sell",
					"ts": ms(at),
				}},
			}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	c.SetContractValue("BTC-USDT-SWAP", 0.01)
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeOKX, Symbols: []string{"BTCUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCoveredRange() || len(got.Events) != 1 {
		t.Fatalf("%+v", got)
	}
	ev := got.Events[0]
	if ev.Exchange != domain.ExchangeOKX || ev.Side != domain.LiquidationSideLong {
		t.Fatalf("event %+v", ev)
	}
	wantQty := 48.9 * 0.01
	if ev.Quantity < wantQty-0.001 || ev.Quantity > wantQty+0.001 {
		t.Fatalf("qty %v want %v", ev.Quantity, wantQty)
	}
}

func TestListLiquidationHistory_FullPageLeavesTheStart(t *testing.T) {
	from := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	to := from.Add(6 * time.Hour)
	at := to.Add(-time.Minute)
	details := make([]map[string]any, okxLiqHistLimit)
	for i := range details {
		details[i] = map[string]any{
			"bkPx": "100", "sz": "1", "posSide": "long", "side": "sell", "ts": ms(at),
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": []map[string]any{{"instId": "BTC-USDT-SWAP", "details": details}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	c.SetContractValue("BTC-USDT-SWAP", 0.01)
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeOKX, Symbols: []string{"BTCUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCoveredRange() || got.CoveredFrom.Equal(from) || !got.CoveredFrom.Equal(at.UTC()) {
		t.Fatalf("coverage %s events %d", got.CoveredFrom, len(got.Events))
	}
}

func TestListLiquidationHistory_EmptyPageUncovered(t *testing.T) {
	from := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "0", "data": []any{}})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeOKX, Symbols: []string{"BTCUSDT"}, From: from, To: to,
	})
	if err != nil || got.HasCoveredRange() {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestListLiquidationHistory_MissingContractDropsThePage(t *testing.T) {
	from := time.Date(2026, 9, 13, 17, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Minute)
	at := from.Add(time.Minute)
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": []map[string]any{{
				"instId": "BTC-USDT-SWAP",
				"details": []map[string]any{{
					"bkPx": "77211.6", "sz": "48.9", "posSide": "long", "side": "sell",
					"ts": ms(at),
				}},
			}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	c.SetContractValue("BTC-USDT-SWAP", 0)
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeOKX, Symbols: []string{"BTCUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HasCoveredRange() || len(got.Events) != 0 {
		t.Fatalf("missing ctVal stored a size: %+v", got)
	}
	if hits != 0 {
		t.Fatalf("liquidation page fetched without a contract size: %d", hits)
	}
}

func TestParseLiquidationWS(t *testing.T) {
	at := time.Date(2026, 9, 13, 18, 0, 0, 0, time.UTC)
	body, _ := json.Marshal(map[string]any{
		"arg": map[string]string{"channel": "liquidation-orders", "instType": "SWAP"},
		"data": []map[string]any{{
			"instId": "ETH-USDT-SWAP",
			"details": []map[string]any{{
				"bkPx": "2500", "sz": "10", "posSide": "short", "side": "buy",
				"ts": ms(at),
			}},
		}},
	})
	got := ParseLiquidationWS(body, func(string) float64 { return 0.1 })
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	if got[0].Symbol != "ETHUSDT" || got[0].Side != domain.LiquidationSideShort || got[0].Quantity != 1 {
		t.Fatalf("%+v", got[0])
	}
}

func TestParseLiquidationWS_SkipsUnknownContract(t *testing.T) {
	at := time.Date(2026, 9, 13, 18, 0, 0, 0, time.UTC)
	body, _ := json.Marshal(map[string]any{
		"arg": map[string]string{"channel": "liquidation-orders", "instType": "SWAP"},
		"data": []map[string]any{
			{
				"instId": "BTC-USDT-SWAP",
				"details": []map[string]any{{
					"bkPx": "77211.6", "sz": "48.9", "posSide": "long", "side": "sell",
					"ts": ms(at),
				}},
			},
			{
				"instId": "ETH-USDT-SWAP",
				"details": []map[string]any{{
					"bkPx": "2500", "sz": "10", "posSide": "short", "side": "buy",
					"ts": ms(at),
				}},
			},
		},
	})
	sizes := map[string]float64{"ETH-USDT-SWAP": 0.1}
	got := ParseLiquidationWS(body, func(inst string) float64 { return sizes[inst] })
	if len(got) != 1 || got[0].Symbol != "ETHUSDT" || got[0].Quantity != 1 {
		t.Fatalf("unknown BTC size should drop, ETH should stay: %+v", got)
	}
	if len(ParseLiquidationWS(body, nil)) != 0 {
		t.Fatal("nil contract lookup invented a size")
	}
}

func TestLiquidationHub_UnknownContractIsZero(t *testing.T) {
	c := NewClient(Options{})
	c.SetContractValue("BTC-USDT-SWAP", 0)
	h := NewLiquidationHub(LiquidationOptions{Client: c})
	if h.ctVal("BTC-USDT-SWAP") != 0 {
		t.Fatal("non-positive ctVal was treated as 1")
	}
	if NewLiquidationHub(LiquidationOptions{}).ctVal("BTC-USDT-SWAP") != 0 {
		t.Fatal("nil client invented a contract size")
	}
}
