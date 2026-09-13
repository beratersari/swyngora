package okx

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

const (
	defaultPublicWSURL = "wss://ws.okx.com:8443/ws/v5/public"
	liqReadIdle        = 90 * time.Second
)

// LiquidationSink receives parsed SWAP liquidation events.
type LiquidationSink interface {
	Record(domain.LiquidationEvent)
	SetLive(domain.Exchange, bool)
	NoteSeen(domain.Exchange)
}

type wsDialer func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error)

// LiquidationHub listens to OKX public liquidation-orders for all SWAP.
type LiquidationHub struct {
	wsBase string
	dial   wsDialer
	sink   LiquidationSink
	client *Client
	now    func() time.Time

	mu     sync.Mutex
	closed bool
	stop   context.CancelFunc
}

// LiquidationOptions configures the SWAP liquidation stream.
type LiquidationOptions struct {
	WSURL  string
	Dial   wsDialer
	Sink   LiquidationSink
	Client *Client
}

// NewLiquidationHub constructs an OKX SWAP liquidation listener.
func NewLiquidationHub(opts LiquidationOptions) *LiquidationHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = defaultPublicWSURL
	}
	dial := opts.Dial
	if dial == nil {
		dial = func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error) {
			d := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
			return d.DialContext(ctx, urlStr, header)
		}
	}
	return &LiquidationHub{
		wsBase: ws, dial: dial, sink: opts.Sink, client: opts.Client, now: time.Now,
	}
}

// Start reconnects until ctx is cancelled.
func (h *LiquidationHub) Start(ctx context.Context) {
	if h == nil || h.sink == nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	h.mu.Lock()
	h.stop = cancel
	h.mu.Unlock()
	backoff := time.Second
	for runCtx.Err() == nil {
		err := h.listen(runCtx)
		h.sink.SetLive(domain.ExchangeOKX, false)
		if runCtx.Err() != nil {
			return
		}
		if err != nil {
			select {
			case <-runCtx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

// Close stops the stream.
func (h *LiquidationHub) Close() {
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
}

func (h *LiquidationHub) listen(ctx context.Context) error {
	conn, _, err := h.dial(ctx, h.wsBase, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	sub, err := json.Marshal(map[string]any{
		"op": "subscribe",
		"args": []map[string]string{{
			"channel":  "liquidation-orders",
			"instType": "SWAP",
		}},
	})
	if err != nil {
		return err
	}
	if err := conn.WriteMessage(websocket.TextMessage, sub); err != nil {
		return err
	}
	h.sink.SetLive(domain.ExchangeOKX, true)

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
				_ = conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			}
		}
	}()
	defer close(auxDone)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(h.now().Add(liqReadIdle))
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		h.sink.NoteSeen(domain.ExchangeOKX)
		if string(payload) == "pong" || string(payload) == "ping" {
			if string(payload) == "ping" {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
			}
			continue
		}
		for _, ev := range ParseLiquidationWS(payload, h.ctVal) {
			h.sink.Record(ev)
		}
	}
}

func (h *LiquidationHub) ctVal(instID string) float64 {
	if h.client == nil {
		return 1
	}
	v := h.client.contractValue(context.Background(), instID)
	if v <= 0 {
		return 1
	}
	return v
}

// ParseLiquidationWS turns one public liquidation-orders push into events.
func ParseLiquidationWS(payload []byte, ctVal func(instID string) float64) []domain.LiquidationEvent {
	var msg struct {
		Arg  json.RawMessage `json:"arg"`
		Data []okxLiqFamily  `json:"data"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil || len(msg.Data) == 0 {
		return nil
	}
	if ctVal == nil {
		ctVal = func(string) float64 { return 1 }
	}
	var out []domain.LiquidationEvent
	for _, fam := range msg.Data {
		sym := SymbolFromInstID(fam.InstID)
		if sym == "" {
			continue
		}
		mult := ctVal(fam.InstID)
		for _, d := range fam.Details {
			ev, ok := detailToEvent(sym, d, mult)
			if ok {
				out = append(out, ev)
			}
		}
	}
	return out
}
