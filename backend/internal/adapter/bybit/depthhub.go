package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBybitWSURL   = "wss://stream.bybit.com/v5/public/spot"
	defaultDepthIdle    = 90 * time.Second
	maxLiveDepthSymbols = 32
	depthBufferSize     = 2048
	depthReadIdle       = 30 * time.Second
	bybitBookDepth      = 200
)

type wsDialer func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error)

type liveSymbol struct {
	symbol   string
	book     *domain.LocalDepthBook
	cancel   context.CancelFunc
	lastUsed time.Time
}

type DepthHub struct {
	wsBase string
	dial   wsDialer
	idle   time.Duration
	now    func() time.Time

	mu     sync.Mutex
	books  map[string]*liveSymbol
	closed bool
	stop   context.CancelFunc
}

func newDepthHub(wsBase string, dial wsDialer, idle time.Duration) *DepthHub {
	wsBase = strings.TrimRight(strings.TrimSpace(wsBase), "/")
	if wsBase == "" {
		wsBase = defaultBybitWSURL
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
		wsBase: wsBase, dial: dial, idle: idle, now: time.Now,
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
		return fmt.Errorf("bybit depth ws dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(h.now().Add(depthReadIdle))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(h.now().Add(depthReadIdle))
	})
	sub := fmt.Sprintf(`{"op":"subscribe","args":["orderbook.%d.%s"]}`, bybitBookDepth, ls.symbol)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(sub)); err != nil {
		return err
	}

	pingDone := make(chan struct{})
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-pingDone:
				return
			case <-t.C:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":"ping"}`))
			}
		}
	}()
	defer close(pingDone)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(h.now().Add(depthReadIdle))
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		kind, u, bids, asks, ts, err := parseBybitBook(payload)
		if err != nil {
			return err
		}
		if kind == "" {
			continue // subscribe ack / pong
		}
		at := time.Time{}
		if ts > 0 {
			at = time.UnixMilli(ts).UTC()
		}
		switch kind {
		case "snapshot":
			ls.book.ReplaceSnapshot(u, bids, asks)
		case "delta":
			if u == 1 {
				// Venue restart: payload is a full snapshot — overwrite, do not apply as a gap.
				ls.book.ReplaceSnapshot(u, bids, asks)
				continue
			}
			if err := ls.book.ApplySequential(u, bids, asks, at); err != nil {
				return err
			}
		}
	}
}

type bybitWSBook struct {
	Success *bool  `json:"success"`
	RetMsg  string `json:"ret_msg"`
	Op      string `json:"op"`
	Topic   string `json:"topic"`
	Type    string `json:"type"`
	TS      int64  `json:"ts"`
	Data    struct {
		Symbol string     `json:"s"`
		Bids   [][]string `json:"b"`
		Asks   [][]string `json:"a"`
		U      int64      `json:"u"`
	} `json:"data"`
}

func parseBybitBook(raw []byte) (kind string, u int64, bids, asks []domain.DepthLevel, ts int64, err error) {
	var msg bybitWSBook
	if err = json.Unmarshal(raw, &msg); err != nil {
		return "", 0, nil, nil, 0, err
	}
	if msg.Op == "pong" || msg.Op == "ping" {
		return "", 0, nil, nil, 0, nil
	}
	if msg.Op == "subscribe" {
		if msg.Success != nil && !*msg.Success {
			return "", 0, nil, nil, 0, fmt.Errorf("bybit subscribe failed: %s", strings.TrimSpace(msg.RetMsg))
		}
		return "", 0, nil, nil, 0, nil
	}
	if msg.Type != "snapshot" && msg.Type != "delta" {
		return "", 0, nil, nil, 0, nil
	}
	return msg.Type, msg.Data.U, parseDepthSide(msg.Data.Bids), parseDepthSide(msg.Data.Asks), msg.TS, nil
}

func parseDepthSide(rows [][]string) []domain.DepthLevel {
	out := make([]domain.DepthLevel, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		lv, ok := domain.ParsePriceQty(row[0], row[1])
		if !ok {
			if p := strings.TrimSpace(row[0]); p != "" {
				out = append(out, domain.DepthLevel{Price: p, Quantity: 0})
			}
			continue
		}
		out = append(out, domain.DepthLevel{Price: strings.TrimSpace(row[0]), Quantity: lv.Quantity})
	}
	return out
}
