package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type wsDialer func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error)

// TradeHub listens to Hyperliquid public trades for USDT-perp names.
type TradeHub struct {
	wsBase string
	dial   wsDialer
	book   *domain.TakerBook
	tape   *domain.PrintTape

	mu     sync.Mutex
	want   map[string]struct{}
	closed bool
	stop   context.CancelFunc
	subCh  chan string
}

// TradeHubOptions configures the public trade stream.
type TradeHubOptions struct {
	WSURL string
	Dial  wsDialer
	Book  *domain.TakerBook
	Tape  *domain.PrintTape
}

// NewTradeHub constructs a Hyperliquid trade listener.
func NewTradeHub(opts TradeHubOptions) *TradeHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = "wss://api.hyperliquid.xyz/ws"
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
		wsBase: ws, dial: dial, book: book, tape: opts.Tape,
		want: map[string]struct{}{
			"BTC": {}, "ETH": {}, "SOL": {}, "XRP": {}, "BNB": {}, "DOGE": {},
		},
		subCh: make(chan string, 256),
	}
}

// Watch adds a BTCUSDT-style symbol. A name Hyperliquid does not use is ignored.
func (h *TradeHub) Watch(symbol string) {
	if h == nil {
		return
	}
	coin, ok := Coin(symbol)
	if !ok {
		return
	}
	h.mu.Lock()
	if _, exists := h.want[coin]; exists {
		h.mu.Unlock()
		return
	}
	h.want[coin] = struct{}{}
	h.mu.Unlock()
	select {
	case h.subCh <- coin:
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
	backoff := time.Second
	for ctx.Err() == nil {
		err := h.session(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
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
		ping := time.NewTicker(15 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-auxDone:
				return
			case <-ping.C:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"method":"ping"}`))
			case coin := <-h.subCh:
				_ = h.subscribe(conn, []string{coin})
			}
		}
	}()
	defer close(auxDone)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		for _, p := range parseTradeMessage(msg) {
			domain.RecordMeasuredPrint(h.book, h.tape, p)
		}
	}
}

func (h *TradeHub) wanted() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.want))
	for coin := range h.want {
		out = append(out, coin)
	}
	return out
}

func (h *TradeHub) subscribe(conn *websocket.Conn, coins []string) error {
	if len(coins) == 0 || conn == nil {
		return nil
	}
	for _, coin := range coins {
		if !perpCoin(coin) {
			continue
		}
		payload, err := json.Marshal(map[string]any{
			"method":       "subscribe",
			"subscription": map[string]string{"type": "trades", "coin": coin},
		})
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return err
		}
	}
	return nil
}

func parseTradeMessage(msg []byte) []domain.TakerPrint {
	var env struct {
		Channel string    `json:"channel"`
		Data    []hlTrade `json:"data"`
	}
	if json.Unmarshal(msg, &env) != nil || env.Channel != "trades" || len(env.Data) == 0 {
		return nil
	}
	out := make([]domain.TakerPrint, 0, len(env.Data))
	for _, row := range env.Data {
		if p, ok := printFromTrade(row); ok {
			out = append(out, p)
		}
	}
	return out
}
