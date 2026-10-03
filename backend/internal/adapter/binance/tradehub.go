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

// TradeHub listens to Binance USD-M <symbol>@aggTrade and records measured prints.
type TradeHub struct {
	wsBase string
	dial   wsDialer
	book   *domain.TakerBook
	tape   *domain.PrintTape
	pull   func(ctx context.Context, symbol string, pages int) error
	now    func() time.Time

	mu     sync.Mutex
	want   map[string]struct{}
	closed bool
	stop   context.CancelFunc
	subCh  chan string
	subID  int
}

// TradeHubOptions configures the USD-M aggregate-trade stream.
type TradeHubOptions struct {
	WSURL string
	Dial  wsDialer
	Book  *domain.TakerBook
	Tape  *domain.PrintTape
	// Pull pages aggTrades forward when the websocket is quiet.
	Pull func(ctx context.Context, symbol string, pages int) error
}

// NewTradeHub constructs a Binance USD-M aggTrade listener.
func NewTradeHub(opts TradeHubOptions) *TradeHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = defaultFuturesWSURL
	}
	ws = strings.TrimSuffix(ws, "/ws")
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
		wsBase: ws, dial: dial, book: book, tape: opts.Tape, pull: opts.Pull, now: time.Now,
		want: map[string]struct{}{
			"BTCUSDT": {}, "ETHUSDT": {}, "SOLUSDT": {}, "XRPUSDT": {}, "BNBUSDT": {}, "DOGEUSDT": {},
		},
		subCh: make(chan string, 256),
	}
}

// Watch adds a USD-M symbol. Already-subscribed names are ignored.
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
	go h.poll(ctx)
}

func (h *TradeHub) poll(ctx context.Context) {
	if h == nil || h.pull == nil {
		return
	}
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	alt := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			_ = h.pull(ctx, "BTCUSDT", 3)
			_ = h.pull(ctx, "ETHUSDT", 1)
			syms := h.wanted()
			if len(syms) == 0 {
				continue
			}
			alt++
			sym := syms[alt%len(syms)]
			if sym == "BTCUSDT" || sym == "ETHUSDT" {
				continue
			}
			_ = h.pull(ctx, sym, 1)
		}
	}
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
	conn, _, err := h.dial(ctx, h.wsBase+"/ws", nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := h.subscribe(conn, h.wanted()); err != nil {
		return err
	}
	auxDone := make(chan struct{})
	go func() {
		resub := time.NewTicker(20 * time.Second)
		defer resub.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-auxDone:
				return
			case <-resub.C:
				_ = h.subscribe(conn, h.wanted())
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
		_ = conn.SetReadDeadline(time.Now().Add(liqReadIdle))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if p, ok := parseAggTrade(msg); ok {
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

func (h *TradeHub) subscribe(conn *websocket.Conn, symbols []string) error {
	if len(symbols) == 0 || conn == nil {
		return nil
	}
	const batch = 50
	for i := 0; i < len(symbols); i += batch {
		j := i + batch
		if j > len(symbols) {
			j = len(symbols)
		}
		params := make([]string, 0, j-i)
		for _, s := range symbols[i:j] {
			params = append(params, strings.ToLower(s)+"@aggTrade")
		}
		h.mu.Lock()
		h.subID++
		id := h.subID
		h.mu.Unlock()
		payload, _ := json.Marshal(map[string]any{
			"method": "SUBSCRIBE",
			"params": params,
			"id":     id,
		})
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return err
		}
	}
	return nil
}

// parseAggTrade reads one USD-M aggTrade. Buyer-is-maker means the taker sold.
func parseAggTrade(msg []byte) (domain.TakerPrint, bool) {
	body := msg
	var wrap struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(msg, &wrap) == nil && len(wrap.Data) > 0 {
		body = wrap.Data
	}
	var row struct {
		Event  string `json:"e"`
		Symbol string `json:"s"`
		Price  string `json:"p"`
		Qty    string `json:"q"`
		Time   int64  `json:"T"`
		Maker  bool   `json:"m"`
	}
	if json.Unmarshal(body, &row) != nil || (row.Event != "" && row.Event != "aggTrade") {
		return domain.TakerPrint{}, false
	}
	if row.Symbol == "" || row.Time <= 0 {
		return domain.TakerPrint{}, false
	}
	px, err1 := strconv.ParseFloat(row.Price, 64)
	qty, err2 := strconv.ParseFloat(row.Qty, 64)
	if err1 != nil || err2 != nil || px <= 0 || qty <= 0 {
		return domain.TakerPrint{}, false
	}
	side := domain.TakerSideBuy
	if row.Maker {
		side = domain.TakerSideSell
	}
	return domain.TakerPrint{
		Exchange: domain.ExchangeBinance,
		Symbol:   domain.NormalizeLiquidationSymbol(row.Symbol),
		Side:     side,
		Price:    px,
		Quantity: qty,
		Notional: px * qty,
		Time:     time.UnixMilli(row.Time).UTC(),
	}, true
}
