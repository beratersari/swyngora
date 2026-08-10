package bybit

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

func TestGetOrderBook_LiveThenGapResync(t *testing.T) {
	var visits atomic.Int32
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_, _, _ = conn.ReadMessage() // subscribe
		n := visits.Add(1)
		if n == 1 {
			_ = conn.WriteMessage(websocket.TextMessage, mustJSON(map[string]any{
				"topic": "orderbook.200.ETHUSDT", "type": "snapshot", "ts": 1,
				"data": map[string]any{"s": "ETHUSDT", "u": 10,
					"b": [][]string{{"3000", "2"}}, "a": [][]string{{"3001", "1"}}},
			}))
			time.Sleep(200 * time.Millisecond)
			_ = conn.WriteMessage(websocket.TextMessage, mustJSON(map[string]any{
				"topic": "orderbook.200.ETHUSDT", "type": "delta", "ts": 2,
				"data": map[string]any{"s": "ETHUSDT", "u": 20,
					"b": [][]string{{"3000", "2"}}, "a": [][]string{{"3001", "1"}}},
			}))
			time.Sleep(2 * time.Second)
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, mustJSON(map[string]any{
			"topic": "orderbook.200.ETHUSDT", "type": "snapshot", "ts": 3,
			"data": map[string]any{"s": "ETHUSDT", "u": 50,
				"b": [][]string{{"3000", "9"}}, "a": [][]string{{"3001", "1"}}},
		}))
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()
	c := NewClient(Options{
		WSURL:     "ws" + strings.TrimPrefix(srv.URL, "http"),
		DepthWait: 4 * time.Second,
		DepthIdle: time.Hour,
	})
	defer c.Close()

	got, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "ETHUSDT", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Live || got.Bids[0].Quantity != 2 {
		t.Fatalf("first %+v", got)
	}
	deadline := time.Now().Add(3 * time.Second)
	var got2 *domain.RawOrderBook
	for time.Now().Before(deadline) {
		got2, err = c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "ETHUSDT", Limit: 20})
		if err == nil && len(got2.Bids) == 1 && got2.Bids[0].Quantity == 9 {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("after resync %+v err=%v", got2, err)
}

func TestParseBybitBook(t *testing.T) {
	kind, u, bids, asks, ts, err := parseBybitBook(mustJSON(map[string]any{
		"topic": "orderbook.200.BTCUSDT", "type": "snapshot", "ts": 9,
		"data": map[string]any{"s": "BTCUSDT", "u": 3, "b": [][]string{{"1", "2"}}, "a": [][]string{{"3", "4"}}},
	}))
	if err != nil || kind != "snapshot" || u != 3 || ts != 9 || len(bids) != 1 || asks[0].Quantity != 4 {
		t.Fatalf("%s u=%d ts=%d bids=%+v asks=%+v err=%v", kind, u, ts, bids, asks, err)
	}
	kind, u, _, _, _, err = parseBybitBook([]byte(`{"topic":"orderbook.200.BTCUSDT","type":"delta","data":{"u":4,"b":[],"a":[]}}`))
	if err != nil || kind != "delta" || u != 4 {
		t.Fatalf("delta %s u=%d err=%v", kind, u, err)
	}
	if _, _, _, _, _, err := parseBybitBook([]byte(`{"op":"subscribe","success":true}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := parseBybitBook([]byte(`{"op":"subscribe","success":false,"ret_msg":"bad symbol"}`)); err == nil {
		t.Fatal("want subscribe failure")
	}
	if _, _, _, _, _, err := parseBybitBook([]byte(`{"op":"pong"}`)); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
