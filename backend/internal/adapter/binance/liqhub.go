package binance

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

const (
	defaultFuturesWSURL = "wss://fstream.binance.com"
	liqReadIdle         = 90 * time.Second
)

// LiquidationSink receives parsed USD-M force-order events.
type LiquidationSink interface {
	Record(domain.LiquidationEvent)
	SetLive(domain.Exchange, bool)
	NoteSeen(domain.Exchange)
}

// LiquidationHub listens to Binance USD-M !forceOrder@arr.
type LiquidationHub struct {
	wsBase string
	dial   wsDialer
	sink   LiquidationSink
	now    func() time.Time

	mu     sync.Mutex
	closed bool
	stop   context.CancelFunc
}

// LiquidationOptions configures the USD-M liquidation stream.
type LiquidationOptions struct {
	WSURL string
	Dial  wsDialer
	Sink  LiquidationSink
}

// NewLiquidationHub constructs a Binance USD-M liquidation listener.
func NewLiquidationHub(opts LiquidationOptions) *LiquidationHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = defaultFuturesWSURL
	}
	dial := opts.Dial
	if dial == nil {
		dial = func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error) {
			d := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
			return d.DialContext(ctx, urlStr, header)
		}
	}
	return &LiquidationHub{wsBase: ws, dial: dial, sink: opts.Sink, now: time.Now}
}

// Start reconnects until ctx is cancelled. Call in a goroutine.
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
		h.sink.SetLive(domain.ExchangeBinance, false)
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
	u := h.wsBase + "/ws/!forceOrder@arr"
	conn, _, err := h.dial(ctx, u, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	h.sink.SetLive(domain.ExchangeBinance, true)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(h.now().Add(liqReadIdle))
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		h.sink.NoteSeen(domain.ExchangeBinance)
		ev, ok := ParseForceOrder(payload)
		if !ok {
			continue
		}
		h.sink.Record(ev)
	}
}

type forceOrderMsg struct {
	Event string `json:"e"`
	E     int64  `json:"E"`
	O     struct {
		Symbol   string `json:"s"`
		Side     string `json:"S"`
		Price    string `json:"p"`
		AvgPrice string `json:"ap"`
		Qty      string `json:"q"`
		Filled   string `json:"z"`
		TradeT   int64  `json:"T"`
	} `json:"o"`
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

// ParseForceOrder maps a Binance USD-M forceOrder payload to a domain event.
func ParseForceOrder(raw []byte) (domain.LiquidationEvent, bool) {
	var wrap forceOrderMsg
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return domain.LiquidationEvent{}, false
	}
	body := raw
	if len(wrap.Data) > 0 && (wrap.Event == "" || wrap.O.Symbol == "") {
		body = wrap.Data
		wrap = forceOrderMsg{}
		if err := json.Unmarshal(body, &wrap); err != nil {
			return domain.LiquidationEvent{}, false
		}
	}
	if wrap.Event != "" && wrap.Event != "forceOrder" {
		return domain.LiquidationEvent{}, false
	}
	if wrap.O.Symbol == "" || wrap.O.Side == "" {
		return domain.LiquidationEvent{}, false
	}
	side, err := domain.LiquidationSideFromBinanceOrder(wrap.O.Side)
	if err != nil {
		return domain.LiquidationEvent{}, false
	}
	px := parseLiqFloat(wrap.O.AvgPrice)
	if px <= 0 {
		px = parseLiqFloat(wrap.O.Price)
	}
	qty := parseLiqFloat(wrap.O.Filled)
	if qty <= 0 {
		qty = parseLiqFloat(wrap.O.Qty)
	}
	if px <= 0 || qty <= 0 {
		return domain.LiquidationEvent{}, false
	}
	ts := wrap.O.TradeT
	if ts == 0 {
		ts = wrap.E
	}
	at := time.Now().UTC()
	if ts > 0 {
		at = time.UnixMilli(ts).UTC()
	}
	return domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance,
		Symbol:   wrap.O.Symbol,
		Side:     side,
		Price:    px,
		Quantity: qty,
		Notional: px * qty,
		Time:     at,
	}, true
}

func parseLiqFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}
