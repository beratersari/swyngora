package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// TradeHub listens to Bybit linear publicTrade.{symbol} for taker flow.
type TradeHub struct {
	wsBase string
	dial   wsDialer
	book   *domain.TakerBook
	now    func() time.Time

	mu     sync.Mutex
	want   map[string]struct{}
	closed bool
	stop   context.CancelFunc
	subCh  chan string
}

// TradeHubOptions configures the public-trade stream.
type TradeHubOptions struct {
	WSURL string
	Dial  wsDialer
	Book  *domain.TakerBook
}

// NewTradeHub constructs a Bybit linear public-trade listener.
func NewTradeHub(opts TradeHubOptions) *TradeHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = defaultLinearWSURL
	}
	dial := opts.Dial
	if dial == nil {
		dial = func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error) {
			d := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
			return d.DialContext(ctx, urlStr, header)
		}
	}
	book := opts.Book
	if book == nil {
		book = domain.NewTakerBook()
	}
	return &TradeHub{
		wsBase: ws, dial: dial, book: book, now: time.Now,
		want: map[string]struct{}{
			"BTCUSDT": {}, "ETHUSDT": {}, "SOLUSDT": {}, "XRPUSDT": {}, "BNBUSDT": {}, "DOGEUSDT": {},
		},
		subCh: make(chan string, 64),
	}
}

// Watch adds a linear symbol.
func (h *TradeHub) Watch(symbol string) {
	if h == nil {
		return
	}
	sym := domain.NormalizeLiquidationSymbol(symbol)
	if sym == "" {
		return
	}
	h.mu.Lock()
	if _, ok := h.want[sym]; ok {
		h.mu.Unlock()
		return
	}
	h.want[sym] = struct{}{}
	h.mu.Unlock()
	select {
	case h.subCh <- sym:
	default:
	}
}

// Start runs until ctx is canceled.
func (h *TradeHub) Start(ctx context.Context) {
	if h == nil {
		return
	}
	ctx, h.stop = context.WithCancel(ctx)
	go h.loop(ctx)
}

// Close stops the hub.
func (h *TradeHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.closed = true
	stop := h.stop
	h.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (h *TradeHub) loop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := h.session(ctx); err != nil && ctx.Err() == nil {
			time.Sleep(2 * time.Second)
		}
	}
}

func (h *TradeHub) session(ctx context.Context) error {
	conn, _, err := h.dial(ctx, h.wsBase, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := h.subscribe(conn, h.wanted()); err != nil {
		return err
	}

	auxDone := make(chan struct{})
	go func() {
		ping := time.NewTicker(20 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-auxDone:
				return
			case <-ping.C:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":"ping"}`))
			case sym := <-h.subCh:
				_ = h.subscribe(conn, []string{sym})
			}
		}
	}()
	defer close(auxDone)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Any read error (including deadline) poisons the gorilla conn —
		// reconnect instead of spinning on the same socket.
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		h.handle(msg)
	}
}

func (h *TradeHub) wanted() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.want))
	for s := range h.want {
		out = append(out, s)
	}
	return out
}

func (h *TradeHub) subscribe(conn *websocket.Conn, symbols []string) error {
	if len(symbols) == 0 {
		return nil
	}
	args := make([]string, 0, len(symbols))
	for _, s := range symbols {
		args = append(args, "publicTrade."+s)
	}
	payload, _ := json.Marshal(map[string]any{"op": "subscribe", "args": args})
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func (h *TradeHub) handle(msg []byte) {
	var raw struct {
		Topic string `json:"topic"`
		Data  []struct {
			Side  string `json:"S"`
			Price string `json:"p"`
			Size  string `json:"v"`
			Time  int64  `json:"T"`
			Sym   string `json:"s"`
		} `json:"data"`
	}
	if json.Unmarshal(msg, &raw) != nil || len(raw.Data) == 0 {
		return
	}
	for _, row := range raw.Data {
		ts := strconv.FormatInt(row.Time, 10)
		p := parseBybitTaker(domain.ExchangeBybit, row.Sym, row.Side, row.Price, row.Size, ts)
		if p.Notional > 0 && h.book != nil {
			h.book.Record(p)
		}
	}
}
