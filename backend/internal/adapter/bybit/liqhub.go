package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultLinearWSURL = "wss://stream.bybit.com/v5/public/linear"
	liqReadIdle        = 90 * time.Second
	liqSubscribeBatch  = 10
	liqTopSymbols      = 40
	liqRefreshEvery    = 5 * time.Minute
)

// LiquidationSink receives parsed linear liquidation events.
type LiquidationSink interface {
	Record(domain.LiquidationEvent)
	SetLive(domain.Exchange, bool)
	NoteSeen(domain.Exchange)
	MarkWatch(domain.Exchange, string)
}

// LiquidationHub listens to Bybit linear allLiquidation.{symbol}.
type LiquidationHub struct {
	wsBase   string
	restBase string
	http     *http.Client
	dial     wsDialer
	sink     LiquidationSink
	now      func() time.Time

	mu     sync.Mutex
	want   map[string]struct{}
	closed bool
	stop   context.CancelFunc
	subCh  chan string
}

// LiquidationOptions configures the linear liquidation stream.
type LiquidationOptions struct {
	WSURL   string
	BaseURL string
	HTTP    *http.Client
	Dial    wsDialer
	Sink    LiquidationSink
}

// NewLiquidationHub constructs a Bybit linear liquidation listener.
func NewLiquidationHub(opts LiquidationOptions) *LiquidationHub {
	ws := strings.TrimRight(strings.TrimSpace(opts.WSURL), "/")
	if ws == "" {
		ws = defaultLinearWSURL
	}
	rest := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if rest == "" {
		rest = defaultBaseURL
	}
	hc := opts.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	dial := opts.Dial
	if dial == nil {
		dial = func(ctx context.Context, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error) {
			d := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
			return d.DialContext(ctx, urlStr, header)
		}
	}
	h := &LiquidationHub{
		wsBase: ws, restBase: rest, http: hc, dial: dial, sink: opts.Sink, now: time.Now,
		want: map[string]struct{}{
			"BTCUSDT": {}, "ETHUSDT": {}, "SOLUSDT": {}, "XRPUSDT": {}, "BNBUSDT": {}, "DOGEUSDT": {},
		},
		subCh: make(chan string, 64),
	}
	return h
}

// Watch adds a linear symbol to the subscribe set.
func (h *LiquidationHub) Watch(symbol string) {
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
	if h.sink != nil {
		h.sink.MarkWatch(domain.ExchangeBybit, sym)
	}
	select {
	case h.subCh <- sym:
	default:
	}
}

// Start reconnects and keeps top linear symbols subscribed.
func (h *LiquidationHub) Start(ctx context.Context) {
	if h == nil || h.sink == nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	h.mu.Lock()
	h.stop = cancel
	h.mu.Unlock()
	h.markWanted()
	h.refreshTop(runCtx)
	h.markWanted()
	backoff := time.Second
	for runCtx.Err() == nil {
		err := h.listen(runCtx)
		h.sink.SetLive(domain.ExchangeBybit, false)
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
	h.sink.SetLive(domain.ExchangeBybit, true)
	if err := h.subscribeAll(conn); err != nil {
		return err
	}

	auxDone := make(chan struct{})
	go func() {
		ping := time.NewTicker(20 * time.Second)
		refresh := time.NewTicker(liqRefreshEvery)
		defer ping.Stop()
		defer refresh.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-auxDone:
				return
			case <-ping.C:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":"ping"}`))
			case sym := <-h.subCh:
				_ = writeBybitSubscribe(conn, []string{sym})
			case <-refresh.C:
				h.refreshTop(ctx)
				_ = h.subscribeAll(conn)
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
		h.sink.NoteSeen(domain.ExchangeBybit)
		for _, ev := range ParseAllLiquidation(payload) {
			h.sink.Record(ev)
		}
	}
}

func (h *LiquidationHub) subscribeAll(conn *websocket.Conn) error {
	h.mu.Lock()
	syms := make([]string, 0, len(h.want))
	for s := range h.want {
		syms = append(syms, s)
	}
	h.mu.Unlock()
	sort.Strings(syms)
	for i := 0; i < len(syms); i += liqSubscribeBatch {
		j := i + liqSubscribeBatch
		if j > len(syms) {
			j = len(syms)
		}
		if err := writeBybitSubscribe(conn, syms[i:j]); err != nil {
			return err
		}
	}
	return nil
}

func writeBybitSubscribe(conn *websocket.Conn, symbols []string) error {
	args := make([]string, 0, len(symbols))
	for _, s := range symbols {
		args = append(args, "allLiquidation."+s)
	}
	body, err := json.Marshal(map[string]any{"op": "subscribe", "args": args})
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, body)
}

func (h *LiquidationHub) refreshTop(ctx context.Context) {
	if h.http == nil || h.restBase == "" {
		return
	}
	q := url.Values{}
	q.Set("category", "linear")
	u := h.restBase + "/v5/market/tickers?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := h.http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var out struct {
		RetCode int `json:"retCode"`
		Result  struct {
			List []struct {
				Symbol      string `json:"symbol"`
				Turnover24h string `json:"turnover24h"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.RetCode != 0 {
		return
	}
	type row struct {
		sym string
		to  float64
	}
	var rows []row
	for _, it := range out.Result.List {
		if !strings.HasSuffix(it.Symbol, "USDT") {
			continue
		}
		to, _ := strconv.ParseFloat(it.Turnover24h, 64)
		rows = append(rows, row{sym: it.Symbol, to: to})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].to > rows[j].to })
	if len(rows) > liqTopSymbols {
		rows = rows[:liqTopSymbols]
	}
	added := make([]string, 0, len(rows))
	h.mu.Lock()
	for _, r := range rows {
		if _, ok := h.want[r.sym]; !ok {
			added = append(added, r.sym)
		}
		h.want[r.sym] = struct{}{}
	}
	h.mu.Unlock()
	for _, s := range added {
		if h.sink != nil {
			h.sink.MarkWatch(domain.ExchangeBybit, s)
		}
	}
}

func (h *LiquidationHub) markWanted() {
	if h.sink == nil {
		return
	}
	h.mu.Lock()
	syms := make([]string, 0, len(h.want))
	for s := range h.want {
		syms = append(syms, s)
	}
	h.mu.Unlock()
	for _, s := range syms {
		h.sink.MarkWatch(domain.ExchangeBybit, s)
	}
}

type bybitLiqMsg struct {
	Op    string `json:"op"`
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  []struct {
		T    int64  `json:"T"`
		S    string `json:"s"`
		Side string `json:"S"`
		V    string `json:"v"`
		P    string `json:"p"`
	} `json:"data"`
}

// ParseAllLiquidation maps a Bybit allLiquidation payload.
func ParseAllLiquidation(raw []byte) []domain.LiquidationEvent {
	var msg bybitLiqMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}
	if msg.Op == "pong" || msg.Op == "ping" || msg.Op == "subscribe" {
		return nil
	}
	if !strings.HasPrefix(msg.Topic, "allLiquidation.") {
		return nil
	}
	out := make([]domain.LiquidationEvent, 0, len(msg.Data))
	for _, d := range msg.Data {
		side, err := domain.LiquidationSideFromBybit(d.Side)
		if err != nil {
			continue
		}
		px, err1 := strconv.ParseFloat(strings.TrimSpace(d.P), 64)
		qty, err2 := strconv.ParseFloat(strings.TrimSpace(d.V), 64)
		if err1 != nil || err2 != nil || px <= 0 || qty <= 0 {
			continue
		}
		at := time.Now().UTC()
		if d.T > 0 {
			at = time.UnixMilli(d.T).UTC()
		}
		out = append(out, domain.LiquidationEvent{
			Exchange: domain.ExchangeBybit,
			Symbol:   d.S,
			Side:     side,
			Price:    px,
			Quantity: qty,
			Notional: px * qty,
			Time:     at,
		})
	}
	return out
}
