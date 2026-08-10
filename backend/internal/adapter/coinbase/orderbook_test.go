package coinbase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOrderBook_LiveThenDropResync(t *testing.T) {
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
		_ = conn.WriteMessage(websocket.TextMessage, []byte(
			`{"type":"snapshot","product_id":"BTC-USD","bids":[["65000","0.4"]],"asks":[["65010","0.2"]]}`))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(
			`{"type":"heartbeat","product_id":"BTC-USD","sequence":1,"time":"2024-01-01T00:00:00.000000Z"}`))
		if n == 1 {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(
				`{"type":"l2update","product_id":"BTC-USD","changes":[["buy","65000","0.8"]],"time":"2024-01-01T00:00:01.000000Z"}`))
			time.Sleep(400 * time.Millisecond)
			return // drop connection → resync
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(
			`{"type":"l2update","product_id":"BTC-USD","changes":[["buy","65000","1.5"]],"time":"2024-01-01T00:00:02.000000Z"}`))
		for i := 0; i < 20; i++ {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(
				`{"type":"heartbeat","product_id":"BTC-USD","sequence":2,"time":"2024-01-01T00:00:03.000000Z"}`))
			time.Sleep(100 * time.Millisecond)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{
		WSURL:     "ws" + strings.TrimPrefix(srv.URL, "http"),
		DepthWait: 4 * time.Second,
		DepthIdle: time.Hour,
	})
	// Avoid REST checksum against the same httptest (no /products book).
	c.ensureDepth()
	c.depth.checksum = nil
	defer c.Close()

	var (
		got *domain.RawOrderBook
		err error
	)
	firstDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(firstDeadline) {
		got, err = c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "BTC-USD", Limit: 20})
		if err == nil && got != nil && got.Live && len(got.Bids) == 1 && got.Bids[0].Quantity == 0.8 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.Live || got.Source != domain.OrderBookSourceWebSocket {
		t.Fatalf("%+v", got)
	}
	if len(got.Bids) != 1 || got.Bids[0].Quantity != 0.8 {
		t.Fatalf("first live %+v", got.Bids)
	}

	resyncDeadline := time.Now().Add(3 * time.Second)
	var got2 *domain.RawOrderBook
	for time.Now().Before(resyncDeadline) {
		got2, err = c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "BTC-USD", Limit: 20})
		if err == nil && got2 != nil && len(got2.Bids) == 1 && got2.Bids[0].Quantity == 1.5 {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("after drop resync %+v err=%v", got2, err)
}

func TestParseCoinbaseFeed_L2Update(t *testing.T) {
	kind, bids, asks, _, err := parseCoinbaseFeed([]byte(
		`{"type":"l2update","product_id":"BTC-USD","changes":[["buy","1","0"],["sell","2","3"]]}`), "BTC-USD")
	if err != nil || kind != "l2update" {
		t.Fatalf("%s %v", kind, err)
	}
	if len(bids) != 1 || bids[0].Quantity != 0 || len(asks) != 1 || asks[0].Quantity != 3 {
		t.Fatalf("bids=%+v asks=%+v", bids, asks)
	}
}

func TestParseCoinbaseFeed_SnapshotAndError(t *testing.T) {
	kind, bids, asks, _, err := parseCoinbaseFeed([]byte(
		`{"type":"snapshot","product_id":"ETH-USD","bids":[["1","2"]],"asks":[["3","4"]]}`), "ETH-USD")
	if err != nil || kind != "snapshot" || len(bids) != 1 || len(asks) != 1 {
		t.Fatalf("%s bids=%+v asks=%+v err=%v", kind, bids, asks, err)
	}
	if _, _, _, _, err := parseCoinbaseFeed([]byte(`{"type":"error","message":"auth"}`), "ETH-USD"); err == nil {
		t.Fatal("want error")
	}
}
