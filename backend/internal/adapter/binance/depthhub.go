package binance

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
	defaultBinanceWSURL     = "wss://stream.binance.com:9443"
	defaultDepthIdle        = 90 * time.Second
	defaultDepthSnapshotLim = 1000
	maxLiveDepthSymbols     = 32
	depthBufferSize         = 2048
	depthReadIdle           = 30 * time.Second
)

// wsDialer opens a Binance depth websocket.
type wsDialer func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error)

type liveSymbol struct {
	symbol   string
	book     *domain.LocalDepthBook
	cancel   context.CancelFunc
	lastUsed time.Time
}

// DepthHub keeps one local Binance spot book per subscribed symbol.
type depthSnapshotFn func(ctx context.Context, symbol string, limit int) (lastID int64, bids, asks []domain.DepthLevel, err error)

type DepthHub struct {
	fetch  depthSnapshotFn
	wsBase string
	dial   wsDialer
	idle   time.Duration
	now    func() time.Time

	mu     sync.Mutex
	books  map[string]*liveSymbol
	closed bool
	stop   context.CancelFunc
}

func newDepthHub(fetch depthSnapshotFn, wsBase string, dial wsDialer, idle time.Duration) *DepthHub {
	wsBase = strings.TrimRight(strings.TrimSpace(wsBase), "/")
	if wsBase == "" {
		wsBase = defaultBinanceWSURL
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
		fetch:  fetch,
		wsBase: wsBase,
		dial:   dial,
		idle:   idle,
		now:    time.Now,
		books:  make(map[string]*liveSymbol),
		stop:   cancel,
	}
	go h.reapLoop(ctx)
	return h
}

// Close stops every live stream.
func (h *DepthHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
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
	h.mu.Unlock()
}

// Get returns a synced local snapshot. It starts the stream on first use and
// waits until the book is contiguous. A gap or drop invalidates the book and
// this call waits for the next successful resync (or ctx timeout).
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
	ls := &liveSymbol{
		symbol:   symbol,
		book:     domain.NewLocalDepthBook(symbol),
		cancel:   cancel,
		lastUsed: h.now(),
	}
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
		err := h.syncOnce(ctx, ls)
		ls.book.Invalidate()
		if ctx.Err() != nil {
			return
		}
		if err != nil && ctx.Err() == nil {
			// reconnect / resync; do not keep the broken book
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
	urlStr := h.wsBase + "/ws/" + strings.ToLower(ls.symbol) + "@depth@100ms"
	conn, _, err := h.dial(ctx, urlStr, nil)
	if err != nil {
		return fmt.Errorf("depth ws dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(h.now().Add(depthReadIdle))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(h.now().Add(depthReadIdle))
	})

	buf := make(chan domain.DepthDiff, depthBufferSize)
	readErr := make(chan error, 1)
	go func() {
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			_ = conn.SetReadDeadline(h.now().Add(depthReadIdle))
			diff, err := parseDepthDiff(payload)
			if err != nil {
				readErr <- err
				return
			}
			select {
			case buf <- diff:
			default:
				readErr <- fmt.Errorf("depth buffer overflow")
				return
			}
		}
	}()

	snapCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	lastID, bids, asks, err := h.fetch(snapCtx, ls.symbol, defaultDepthSnapshotLim)
	cancel()
	if err != nil {
		return err
	}
	ls.book.LoadSnapshot(lastID, bids, asks)

	// Drain what arrived while the snapshot was in flight, then follow the live stream.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErr:
			return err
		case d := <-buf:
			if err := ls.book.ApplyDiff(d); err != nil {
				return err
			}
		default:
			goto live
		}
	}
live:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErr:
			return err
		case d := <-buf:
			if err := ls.book.ApplyDiff(d); err != nil {
				return err
			}
		}
	}
}

type depthStreamMsg struct {
	Event     string     `json:"e"`
	EventTime int64      `json:"E"`
	Symbol    string     `json:"s"`
	FirstID   int64      `json:"U"`
	FinalID   int64      `json:"u"`
	Bids      [][]string `json:"b"`
	Asks      [][]string `json:"a"`
}

func parseDepthDiff(raw []byte) (domain.DepthDiff, error) {
	var msg depthStreamMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return domain.DepthDiff{}, err
	}
	if msg.Event != "" && msg.Event != "depthUpdate" {
		return domain.DepthDiff{}, fmt.Errorf("unexpected depth event %q", msg.Event)
	}
	d := domain.DepthDiff{
		Symbol:  msg.Symbol,
		FirstID: msg.FirstID,
		FinalID: msg.FinalID,
		Bids:    parseDepthSide(msg.Bids),
		Asks:    parseDepthSide(msg.Asks),
	}
	if msg.EventTime > 0 {
		d.EventAt = time.UnixMilli(msg.EventTime).UTC()
	}
	return d, nil
}

func parseDepthSide(rows [][]string) []domain.DepthLevel {
	out := make([]domain.DepthLevel, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		lv, ok := domain.ParsePriceQty(row[0], row[1])
		if !ok {
			// qty 0 is a delete — ParsePriceQty rejects qty<=0
			if p := strings.TrimSpace(row[0]); p != "" {
				out = append(out, domain.DepthLevel{Price: p, Quantity: 0})
			}
			continue
		}
		out = append(out, domain.DepthLevel{Price: strings.TrimSpace(row[0]), Quantity: lv.Quantity})
	}
	return out
}
