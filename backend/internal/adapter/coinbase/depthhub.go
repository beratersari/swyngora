package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultCoinbaseWSURL = "wss://ws-feed.exchange.coinbase.com"
	defaultDepthIdle     = 90 * time.Second
	maxLiveDepthSymbols  = 32
	depthReadIdle        = 8 * time.Second // heartbeats arrive ~1s
	heartbeatGrace       = 5 * time.Second
	checksumEvery        = 15 * time.Second
)

type wsDialer func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error)

type liveSymbol struct {
	symbol   string
	book     *domain.LocalDepthBook
	cancel   context.CancelFunc
	lastUsed time.Time
}

type DepthHub struct {
	wsBase   string
	dial     wsDialer
	idle     time.Duration
	checksum func(ctx context.Context, symbol string) (bid, ask domain.PriceLevel, ok bool, err error)
	now      func() time.Time

	mu     sync.Mutex
	books  map[string]*liveSymbol
	closed bool
	stop   context.CancelFunc
}

func newDepthHub(wsBase string, dial wsDialer, idle time.Duration, checksum func(context.Context, string) (domain.PriceLevel, domain.PriceLevel, bool, error)) *DepthHub {
	wsBase = strings.TrimRight(strings.TrimSpace(wsBase), "/")
	if wsBase == "" {
		wsBase = defaultCoinbaseWSURL
	}
	if dial == nil {
		dial = func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error) {
			d := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
			return d.DialContext(ctx, urlStr, header)
		}
	}
	if idle <= 0 {
		idle = defaultDepthIdle
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &DepthHub{
		wsBase: wsBase, dial: dial, idle: idle, checksum: checksum, now: time.Now,
		books: make(map[string]*liveSymbol), stop: cancel,
	}
	go h.reapLoop(ctx)
	return h
}

func (h *DepthHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	if h.stop != nil {
		h.stop()
	}
	for _, ls := range h.books {
		if ls.cancel != nil {
			ls.cancel()
		}
		ls.book.Invalidate()
	}
	h.books = map[string]*liveSymbol{}
}

func (h *DepthHub) Get(ctx context.Context, symbol string, limit int) (*domain.RawOrderBook, error) {
	if h == nil {
		return nil, fmt.Errorf("%w: depth hub not configured", domain.ErrUpstream)
	}
	ls, err := h.ensure(symbol)
	if err != nil {
		return nil, err
	}
	ticker := time.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	for {
		if snap, err := ls.book.Snapshot(limit); err == nil {
			return snap, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: waiting for live order book: %v", domain.ErrUpstream, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (h *DepthHub) ensure(symbol string) (*liveSymbol, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, fmt.Errorf("%w: depth hub closed", domain.ErrUpstream)
	}
	if ls, ok := h.books[symbol]; ok {
		ls.lastUsed = h.now()
		return ls, nil
	}
	if len(h.books) >= maxLiveDepthSymbols {
		return nil, fmt.Errorf("%w: too many live order books", domain.ErrInvalidArgument)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ls := &liveSymbol{symbol: symbol, book: domain.NewLocalDepthBook(symbol), cancel: cancel, lastUsed: h.now()}
	h.books[symbol] = ls
	go h.runSymbol(ctx, ls)
	return ls, nil
}

func (h *DepthHub) reapLoop(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.reapIdle()
		}
	}
}

func (h *DepthHub) reapIdle() {
	cutoff := h.now().Add(-h.idle)
	h.mu.Lock()
	defer h.mu.Unlock()
	for sym, ls := range h.books {
		if ls.lastUsed.After(cutoff) {
			continue
		}
		if ls.cancel != nil {
			ls.cancel()
		}
		ls.book.Invalidate()
		delete(h.books, sym)
	}
}

func (h *DepthHub) runSymbol(ctx context.Context, ls *liveSymbol) {
	backoff := 200 * time.Millisecond
	for {
		if ctx.Err() != nil {
			return
		}
		_ = h.syncOnce(ctx, ls)
		ls.book.Invalidate()
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
}

func (h *DepthHub) syncOnce(ctx context.Context, ls *liveSymbol) error {
	conn, _, err := h.dial(ctx, h.wsBase, nil)
	if err != nil {
		return fmt.Errorf("coinbase depth ws dial: %w", err)
	}
	defer conn.Close()
	// level2 requires auth; level2_batch is the public snapshot + l2update feed (50ms).
	sub, _ := json.Marshal(map[string]any{
		"type":        "subscribe",
		"product_ids": []string{ls.symbol},
		"channels":    []string{"level2_batch", "heartbeat"},
	})
	if err := conn.WriteMessage(websocket.TextMessage, sub); err != nil {
		return err
	}

	lastBeat := h.now()
	nextCheck := h.now().Add(checksumEvery)
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(h.now().Add(depthReadIdle))
	})

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if h.now().Sub(lastBeat) > heartbeatGrace && ls.book.Synced() {
			return fmt.Errorf("coinbase heartbeat timeout")
		}
		if h.checksum != nil && ls.book.Synced() && !h.now().Before(nextCheck) {
			nextCheck = h.now().Add(checksumEvery)
			if err := h.verifyTop(ctx, ls); err != nil {
				return err
			}
		}
		_ = conn.SetReadDeadline(h.now().Add(depthReadIdle))
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		kind, bids, asks, at, err := parseCoinbaseFeed(payload, ls.symbol)
		if err != nil {
			return err
		}
		switch kind {
		case "snapshot":
			ls.book.ReplaceSnapshot(0, bids, asks)
			lastBeat = h.now()
		case "l2update":
			if err := ls.book.ApplyUnsequenced(bids, asks, at); err != nil {
				return err
			}
			lastBeat = h.now()
		case "heartbeat":
			lastBeat = h.now()
		}
	}
}

func (h *DepthHub) verifyTop(ctx context.Context, ls *liveSymbol) error {
	wantBid, wantAsk, ok, err := h.checksum(ctx, ls.symbol)
	if err != nil || !ok {
		// REST flakiness must not tear down a healthy stream.
		return nil
	}
	gotBid, gotAsk, ok := ls.book.BestBidAsk()
	if !ok {
		return fmt.Errorf("%w: local book empty during checksum", domain.ErrConflict)
	}
	// Compare prices only — BBO size moves faster than a REST round-trip.
	if math.Abs(gotBid.Price-wantBid.Price) > 1e-8 || math.Abs(gotAsk.Price-wantAsk.Price) > 1e-8 {
		return fmt.Errorf("%w: rest checksum mismatch bid %v/%v ask %v/%v",
			domain.ErrConflict, gotBid.Price, wantBid.Price, gotAsk.Price, wantAsk.Price)
	}
	return nil
}

type coinbaseFeedMsg struct {
	Type      string     `json:"type"`
	ProductID string     `json:"product_id"`
	Bids      [][]string `json:"bids"`
	Asks      [][]string `json:"asks"`
	Changes   [][]string `json:"changes"`
	Time      string     `json:"time"`
}

func parseCoinbaseFeed(raw []byte, want string) (kind string, bids, asks []domain.DepthLevel, at time.Time, err error) {
	var msg coinbaseFeedMsg
	if err = json.Unmarshal(raw, &msg); err != nil {
		return "", nil, nil, time.Time{}, err
	}
	if msg.ProductID != "" && !strings.EqualFold(msg.ProductID, want) {
		return "", nil, nil, time.Time{}, nil
	}
	if msg.Time != "" {
		if t, perr := time.Parse(time.RFC3339Nano, msg.Time); perr == nil {
			at = t.UTC()
		}
	}
	switch msg.Type {
	case "snapshot":
		return "snapshot", parseDepthRows(msg.Bids), parseDepthRows(msg.Asks), at, nil
	case "l2update":
		var b, a []domain.DepthLevel
		for _, ch := range msg.Changes {
			if len(ch) < 3 {
				continue
			}
			side := strings.ToLower(ch[0])
			lv, ok := domain.ParsePriceQty(ch[1], ch[2])
			d := domain.DepthLevel{Price: strings.TrimSpace(ch[1])}
			if ok {
				d.Quantity = lv.Quantity
			}
			if side == "buy" {
				b = append(b, d)
			} else if side == "sell" {
				a = append(a, d)
			}
		}
		return "l2update", b, a, at, nil
	case "heartbeat", "subscriptions", "error":
		if msg.Type == "error" {
			return "", nil, nil, time.Time{}, fmt.Errorf("coinbase ws error: %s", strings.TrimSpace(string(raw)))
		}
		return msg.Type, nil, nil, at, nil
	default:
		return "", nil, nil, time.Time{}, nil
	}
}

func parseDepthRows(rows [][]string) []domain.DepthLevel {
	out := make([]domain.DepthLevel, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		lv, ok := domain.ParsePriceQty(row[0], row[1])
		if !ok {
			continue
		}
		out = append(out, domain.DepthLevel{Price: strings.TrimSpace(row[0]), Quantity: lv.Quantity})
	}
	return out
}
