package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFetchDepthSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/depth" || r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("req %s %v", r.URL.Path, r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lastUpdateId": 42,
			"bids":         [][]string{{"100.00", "1.5"}},
			"asks":         [][]string{{"100.10", "0.8"}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	defer c.Close()
	id, bids, asks, err := c.fetchDepthSnapshot(context.Background(), "btcusdt", 100)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 || len(bids) != 1 || bids[0].Price != "100.00" || asks[0].Quantity != 0.8 {
		t.Fatalf("id=%d bids=%+v asks=%+v", id, bids, asks)
	}
}

func TestDepthHub_LiveThenGapResync(t *testing.T) {
	var snapID atomic.Int64
	snapID.Store(10)
	var visits atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/depth", func(w http.ResponseWriter, r *http.Request) {
		id := snapID.Load()
		qty := "1"
		if id > 10 {
			qty = "9"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lastUpdateId": id,
			"bids":         [][]string{{"100.00", qty}},
			"asks":         [][]string{{"101.00", "1"}},
		})
	})
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		n := visits.Add(1)
		if n == 1 {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(
				`{"e":"depthUpdate","E":1,"s":"BTCUSDT","U":11,"u":11,"b":[["100.00","2"]],"a":[["101.00","1"]]}`))
			// Leave time for Get to observe qty=2, then send a gap so the hub must resync.
			time.Sleep(200 * time.Millisecond)
			snapID.Store(50)
			_ = conn.WriteMessage(websocket.TextMessage, []byte(
				`{"e":"depthUpdate","E":2,"s":"BTCUSDT","U":20,"u":20,"b":[["100.00","2"]],"a":[["101.00","1"]]}`))
			time.Sleep(2 * time.Second)
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(
			`{"e":"depthUpdate","E":3,"s":"BTCUSDT","U":51,"u":51,"b":[["100.00","9"]],"a":[["101.00","1"]]}`))
		time.Sleep(2 * time.Second)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	wsBase := "ws" + strings.TrimPrefix(srv.URL, "http")

	c := NewClient(Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		WSURL:      wsBase,
		DepthWait:  4 * time.Second,
		DepthIdle:  time.Hour,
	})
	defer c.Close()

	got, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "BTCUSDT", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Live || got.Source != domain.OrderBookSourceWebSocket {
		t.Fatalf("want live ws, got %+v", got)
	}
	if len(got.Bids) != 1 || got.Bids[0].Quantity != 2 {
		t.Fatalf("after first diff: %+v", got.Bids)
	}

	deadline := time.Now().Add(3 * time.Second)
	var got2 *domain.RawOrderBook
	for time.Now().Before(deadline) {
		got2, err = c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "BTCUSDT", Limit: 20})
		if err == nil && len(got2.Bids) == 1 && got2.Bids[0].Quantity == 9 {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got2 == nil || len(got2.Bids) != 1 || got2.Bids[0].Quantity != 9 {
		t.Fatalf("after resync want qty 9, got %+v err=%v", got2, err)
	}
}
