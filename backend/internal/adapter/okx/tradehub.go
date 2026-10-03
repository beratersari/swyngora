package okx

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

// TradeHub listens to OKX public trades for USDT swaps.
type TradeHub struct {
	wsBase string
	dial   wsDialer
	book   *domain.TakerBook
	tape   *domain.PrintTape
	ctVal  func(inst string) float64
	now    func() time.Time

	mu     sync.Mutex
	want   map[string]struct{}
	ct     map[string]float64
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
	// ContractValue maps a swap instId to base size per contract.
	ContractValue func(inst string) float64
}

// NewTradeHub constructs an OKX swap trade listener.
func NewTradeHub(opts TradeHubOptions) *TradeHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = "wss://ws.okx.com:8443/ws/v5/public"
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
		wsBase: ws, dial: dial, book: book, tape: opts.Tape, ctVal: opts.ContractValue, now: time.Now,
		want: map[string]struct{}{
			"BTC-USDT-SWAP": {}, "ETH-USDT-SWAP": {}, "SOL-USDT-SWAP": {},
			"XRP-USDT-SWAP": {}, "BNB-USDT-SWAP": {}, "DOGE-USDT-SWAP": {},
		},
		ct:    map[string]float64{},
		subCh: make(chan string, 256),
	}
}

// Watch adds a BTCUSDT-style symbol (converted to a swap instId).
func (h *TradeHub) Watch(symbol string) {
	if h == nil {
		return
	}
	inst := SwapInstID(symbol)
	if inst == "" {
		return
	}
	h.mu.Lock()
	if _, ok := h.want[inst]; ok {
		h.mu.Unlock()
		return
	}
	h.want[inst] = struct{}{}
	h.mu.Unlock()
	select {
	case h.subCh <- inst:
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
				_ = h.subscribe(conn, h.wanted())
			case inst := <-h.subCh:
				_ = h.subscribe(conn, []string{inst})
			}
		}
	}()
	defer close(auxDone)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if string(msg) == "pong" {
			continue
		}
		for _, p := range h.parseTrades(msg) {
			domain.RecordMeasuredPrint(h.book, h.tape, p)
		}
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

func (h *TradeHub) subscribe(conn *websocket.Conn, insts []string) error {
	if len(insts) == 0 || conn == nil {
		return nil
	}
	const batch = 20
	for i := 0; i < len(insts); i += batch {
		j := i + batch
		if j > len(insts) {
			j = len(insts)
		}
		args := make([]map[string]string, 0, j-i)
		for _, inst := range insts[i:j] {
			args = append(args, map[string]string{"channel": "trades", "instId": inst})
		}
		payload, _ := json.Marshal(map[string]any{"op": "subscribe", "args": args})
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return err
		}
	}
	return nil
}

func (h *TradeHub) contractSize(inst string) float64 {
	h.mu.Lock()
	v := h.ct[inst]
	h.mu.Unlock()
	if v > 0 {
		return v
	}
	if h.ctVal == nil {
		return 0
	}
	v = h.ctVal(inst)
	if v <= 0 {
		return 0
	}
	h.mu.Lock()
	h.ct[inst] = v
	h.mu.Unlock()
	return v
}

func (h *TradeHub) parseTrades(msg []byte) []domain.TakerPrint {
	var raw struct {
		Arg struct {
			InstID string `json:"instId"`
		} `json:"arg"`
		Data []struct {
			InstID string `json:"instId"`
			Side   string `json:"side"`
			Px     string `json:"px"`
			Sz     string `json:"sz"`
			Ts     string `json:"ts"`
		} `json:"data"`
	}
	if json.Unmarshal(msg, &raw) != nil || len(raw.Data) == 0 {
		return nil
	}
	out := make([]domain.TakerPrint, 0, len(raw.Data))
	for _, row := range raw.Data {
		inst := row.InstID
		if inst == "" {
			inst = raw.Arg.InstID
		}
		ct := h.contractSize(inst)
		p, ok := tradePrint(inst, row.Side, row.Px, row.Sz, row.Ts, ct)
		if ok {
			out = append(out, p)
		}
	}
	return out
}

func tradePrint(inst, side, px, sz, ts string, ctVal float64) (domain.TakerPrint, bool) {
	if ctVal <= 0 {
		return domain.TakerPrint{}, false
	}
	symbol := SymbolFromInstID(inst)
	if symbol == "" {
		return domain.TakerPrint{}, false
	}
	price, err1 := strconv.ParseFloat(px, 64)
	contracts, err2 := strconv.ParseFloat(sz, 64)
	ms, err3 := strconv.ParseInt(ts, 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || price <= 0 || contracts <= 0 {
		return domain.TakerPrint{}, false
	}
	taker := domain.TakerSideBuy
	switch strings.ToLower(side) {
	case "buy":
		taker = domain.TakerSideBuy
	case "sell":
		taker = domain.TakerSideSell
	default:
		return domain.TakerPrint{}, false
	}
	qty := contracts * ctVal
	return domain.TakerPrint{
		Exchange: domain.ExchangeOKX,
		Symbol:   symbol,
		Side:     taker,
		Price:    price,
		Quantity: qty,
		Notional: qty * price,
		Time:     time.UnixMilli(ms).UTC(),
	}, true
}
