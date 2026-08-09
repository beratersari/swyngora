package realtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// PriceFetcher loads 24h tickers for subscribed symbols.
type PriceFetcher interface {
	GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error)
}

// PortfolioAccess checks share/owner permission and loads a snapshot.
type PortfolioAccess interface {
	CanViewPortfolio(ctx context.Context, actorClientID, portfolioID string) error
	RealtimeSnapshot(ctx context.Context, actorClientID, portfolioID string) (*domain.PortfolioView, []domain.PendingOrder, error)
}

// Outbound is a JSON envelope written to one WebSocket.
type Outbound map[string]any

// Session is one authenticated socket.
type Session struct {
	ID       string
	ClientID string
	// Out carries Outbound maps or domain.PortfolioChange (mapped to DTO in transport).
	Out chan any

	mu          sync.Mutex
	prices      map[string]domain.SymbolRef // key -> ref
	portfolioID string
	closed      bool
}

// Hub tracks subscriptions and fans out price + portfolio events.
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	// price key -> session ids
	priceSubs map[string]map[string]struct{}
	// book id -> session ids
	bookSubs map[string]map[string]struct{}

	market   PriceFetcher
	access   PortfolioAccess
	interval time.Duration
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Options configure the hub.
type Options struct {
	Market   PriceFetcher
	Access   PortfolioAccess
	Interval time.Duration
}

// NewHub constructs an empty hub.
func NewHub(opts Options) *Hub {
	iv := opts.Interval
	if iv <= 0 {
		iv = 5 * time.Second
	}
	return &Hub{
		sessions:  make(map[string]*Session),
		priceSubs: make(map[string]map[string]struct{}),
		bookSubs:  make(map[string]map[string]struct{}),
		market:    opts.Market,
		access:    opts.Access,
		interval:  iv,
		now:       time.Now,
		sleep: func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		},
	}
}

// Register adds a session. Caller owns reading/writing the socket.
func (h *Hub) Register(id, clientID string) *Session {
	s := &Session{
		ID:       id,
		ClientID: clientID,
		Out:      make(chan any, 64),
		prices:   make(map[string]domain.SymbolRef),
	}
	h.mu.Lock()
	h.sessions[id] = s
	h.mu.Unlock()
	s.send(Outbound{
		"type":     domain.RealtimeTypeHello,
		"protocol": domain.RealtimeProtocolVersion,
		"clientId": clientID,
		"path":     domain.RealtimeWSPath,
	})
	return s
}

// Unregister drops the session and all of its subscriptions.
func (h *Hub) Unregister(s *Session) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	keys := make([]string, 0, len(s.prices))
	for k := range s.prices {
		keys = append(keys, k)
	}
	book := s.portfolioID
	s.prices = map[string]domain.SymbolRef{}
	s.portfolioID = ""
	s.mu.Unlock()

	h.mu.Lock()
	delete(h.sessions, s.ID)
	for _, k := range keys {
		if m := h.priceSubs[k]; m != nil {
			delete(m, s.ID)
			if len(m) == 0 {
				delete(h.priceSubs, k)
			}
		}
	}
	if book != "" {
		if m := h.bookSubs[book]; m != nil {
			delete(m, s.ID)
			if len(m) == 0 {
				delete(h.bookSubs, book)
			}
		}
	}
	h.mu.Unlock()
	close(s.Out)
}

// SubscribePrices adds symbols (union). Immediately snapshots each ticker.
func (h *Hub) SubscribePrices(ctx context.Context, s *Session, refs []domain.SymbolRef) error {
	if s == nil {
		return fmt.Errorf("%w: session required", domain.ErrInvalidArgument)
	}
	if len(refs) == 0 {
		return fmt.Errorf("%w: symbols required", domain.ErrInvalidArgument)
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("%w: session closed", domain.ErrInvalidArgument)
	}
	for _, ref := range refs {
		if len(s.prices) >= domain.MaxRealtimePriceSymbols {
			s.mu.Unlock()
			return fmt.Errorf("%w: at most %d price symbols per connection", domain.ErrInvalidArgument, domain.MaxRealtimePriceSymbols)
		}
		key := domain.SymbolKey(ref.Exchange, ref.Symbol)
		s.prices[key] = ref
	}
	cur := make([]domain.SymbolRef, 0, len(s.prices))
	for _, r := range s.prices {
		cur = append(cur, r)
	}
	s.mu.Unlock()

	h.mu.Lock()
	for _, ref := range refs {
		key := domain.SymbolKey(ref.Exchange, ref.Symbol)
		m := h.priceSubs[key]
		if m == nil {
			m = make(map[string]struct{})
			h.priceSubs[key] = m
		}
		m[s.ID] = struct{}{}
	}
	h.mu.Unlock()

	ackSyms := make([]map[string]string, 0, len(cur))
	for _, r := range cur {
		ackSyms = append(ackSyms, map[string]string{"exchange": string(r.Exchange), "symbol": r.Symbol})
	}
	s.send(Outbound{"type": domain.RealtimeTypeAck, "op": domain.RealtimeOpSubscribePrices, "symbols": ackSyms})
	h.pushPriceSnapshots(ctx, s, refs)
	return nil
}

// UnsubscribePrices removes symbols. Empty list = unsubscribe all prices.
func (h *Hub) UnsubscribePrices(s *Session, refs []domain.SymbolRef) {
	if s == nil {
		return
	}
	s.mu.Lock()
	var keys []string
	if len(refs) == 0 {
		for k := range s.prices {
			keys = append(keys, k)
		}
		s.prices = make(map[string]domain.SymbolRef)
	} else {
		for _, ref := range refs {
			k := domain.SymbolKey(ref.Exchange, ref.Symbol)
			delete(s.prices, k)
			keys = append(keys, k)
		}
	}
	s.mu.Unlock()

	h.mu.Lock()
	for _, k := range keys {
		if m := h.priceSubs[k]; m != nil {
			delete(m, s.ID)
			if len(m) == 0 {
				delete(h.priceSubs, k)
			}
		}
	}
	h.mu.Unlock()
	s.send(Outbound{"type": domain.RealtimeTypeAck, "op": domain.RealtimeOpUnsubscribePrices})
}

// SubscribePortfolio binds this connection to one book the actor can view.
func (h *Hub) SubscribePortfolio(ctx context.Context, s *Session, portfolioID string) error {
	if s == nil {
		return fmt.Errorf("%w: session required", domain.ErrInvalidArgument)
	}
	portfolioID = trimID(portfolioID)
	if portfolioID == "" {
		return fmt.Errorf("%w: portfolioId is required", domain.ErrInvalidArgument)
	}
	if h.access == nil {
		return fmt.Errorf("%w: portfolio realtime not configured", domain.ErrUpstream)
	}
	if err := h.access.CanViewPortfolio(ctx, s.ClientID, portfolioID); err != nil {
		return err
	}
	view, orders, err := h.access.RealtimeSnapshot(ctx, s.ClientID, portfolioID)
	if err != nil {
		return err
	}
	if view != nil && view.ID != "" {
		portfolioID = view.ID
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("%w: session closed", domain.ErrInvalidArgument)
	}
	prev := s.portfolioID
	s.portfolioID = portfolioID
	s.mu.Unlock()

	h.mu.Lock()
	if prev != "" && prev != portfolioID {
		if m := h.bookSubs[prev]; m != nil {
			delete(m, s.ID)
			if len(m) == 0 {
				delete(h.bookSubs, prev)
			}
		}
	}
	m := h.bookSubs[portfolioID]
	if m == nil {
		m = make(map[string]struct{})
		h.bookSubs[portfolioID] = m
	}
	m[s.ID] = struct{}{}
	h.mu.Unlock()

	s.send(Outbound{
		"type":        domain.RealtimeTypeAck,
		"op":          domain.RealtimeOpSubscribePortfolio,
		"portfolioId": portfolioID,
	})
	s.send(domain.PortfolioChange{
		PortfolioID: portfolioID,
		Reason:      domain.PortfolioChangeSnapshot,
		View:        view,
		Orders:      orders,
	})
	return nil
}

// UnsubscribePortfolio drops the book subscription.
func (h *Hub) UnsubscribePortfolio(s *Session) {
	if s == nil {
		return
	}
	s.mu.Lock()
	book := s.portfolioID
	s.portfolioID = ""
	s.mu.Unlock()
	if book == "" {
		return
	}
	h.mu.Lock()
	if m := h.bookSubs[book]; m != nil {
		delete(m, s.ID)
		if len(m) == 0 {
			delete(h.bookSubs, book)
		}
	}
	h.mu.Unlock()
	s.send(Outbound{"type": domain.RealtimeTypeAck, "op": domain.RealtimeOpUnsubscribePortfolio})
}

// OnPortfolioChange implements domain.PortfolioChangeSink.
func (h *Hub) OnPortfolioChange(ev domain.PortfolioChange) {
	if h == nil || ev.PortfolioID == "" {
		return
	}
	h.mu.RLock()
	ids := make([]string, 0, len(h.bookSubs[ev.PortfolioID]))
	for id := range h.bookSubs[ev.PortfolioID] {
		ids = append(ids, id)
	}
	sessions := make([]*Session, 0, len(ids))
	for _, id := range ids {
		if s := h.sessions[id]; s != nil {
			sessions = append(sessions, s)
		}
	}
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, s := range sessions {
		if h.access != nil {
			if err := h.access.CanViewPortfolio(ctx, s.ClientID, ev.PortfolioID); err != nil {
				h.dropBook(s, ev.PortfolioID, err)
				continue
			}
		}
		s.send(ev)
	}
}

func (h *Hub) dropBook(s *Session, bookID string, accessErr error) {
	s.mu.Lock()
	if s.portfolioID == bookID {
		s.portfolioID = ""
	}
	s.mu.Unlock()
	h.mu.Lock()
	if m := h.bookSubs[bookID]; m != nil {
		delete(m, s.ID)
		if len(m) == 0 {
			delete(h.bookSubs, bookID)
		}
	}
	h.mu.Unlock()
	s.send(Outbound{
		"type":        domain.RealtimeTypeError,
		"code":        "forbidden",
		"message":     "no longer have access to this portfolio",
		"portfolioId": bookID,
		"detail":      accessErr.Error(),
	})
}

// Start runs the price pump until ctx is cancelled.
func (h *Hub) Start(ctx context.Context) {
	_ = h.PumpOnce(ctx)
	for {
		if err := h.sleep(ctx, h.interval); err != nil {
			return
		}
		_ = h.PumpOnce(ctx)
	}
}

// PumpOnce fetches every currently subscribed symbol and broadcasts ticks.
func (h *Hub) PumpOnce(ctx context.Context) error {
	if h.market == nil {
		return nil
	}
	refs := h.subscribedSymbols()
	if len(refs) == 0 {
		return nil
	}
	now := h.now().UTC()
	for _, ref := range refs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		tkr, err := h.market.GetTicker24h(ctx, string(ref.Exchange), ref.Symbol)
		if err != nil || tkr == nil {
			continue
		}
		tick := domain.PriceTickFromTicker(ref.Exchange, tkr, now)
		if tick.Symbol == "" {
			tick.Symbol = ref.Symbol
		}
		h.broadcastPrice(tick)
	}
	return nil
}

func (h *Hub) subscribedSymbols() []domain.SymbolRef {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]domain.SymbolRef, 0, len(h.priceSubs))
	seen := map[string]struct{}{}
	for _, s := range h.sessions {
		s.mu.Lock()
		for k, ref := range s.prices {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, ref)
		}
		s.mu.Unlock()
	}
	return out
}

func (h *Hub) pushPriceSnapshots(ctx context.Context, s *Session, refs []domain.SymbolRef) {
	if h.market == nil || s == nil {
		return
	}
	now := h.now().UTC()
	for _, ref := range refs {
		tkr, err := h.market.GetTicker24h(ctx, string(ref.Exchange), ref.Symbol)
		if err != nil || tkr == nil {
			continue
		}
		tick := domain.PriceTickFromTicker(ref.Exchange, tkr, now)
		if tick.Symbol == "" {
			tick.Symbol = ref.Symbol
		}
		s.send(priceOutbound(tick))
	}
}

func (h *Hub) broadcastPrice(tick domain.PriceTick) {
	key := domain.SymbolKey(tick.Exchange, tick.Symbol)
	msg := priceOutbound(tick)
	h.mu.RLock()
	ids := h.priceSubs[key]
	sessions := make([]*Session, 0, len(ids))
	for id := range ids {
		if s := h.sessions[id]; s != nil {
			sessions = append(sessions, s)
		}
	}
	h.mu.RUnlock()
	for _, s := range sessions {
		s.send(msg)
	}
}

func (s *Session) send(msg any) {
	if s == nil || msg == nil {
		return
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return
	}
	select {
	case s.Out <- msg:
	default:
		// Slow consumer: drop this event rather than blocking the hub.
	}
}

func priceOutbound(tick domain.PriceTick) Outbound {
	return Outbound{
		"type":               domain.RealtimeTypePrice,
		"exchange":           string(tick.Exchange),
		"symbol":             tick.Symbol,
		"lastPrice":          tick.LastPrice,
		"priceChange":        tick.PriceChange,
		"priceChangePercent": tick.PriceChangePercent,
		"openPrice":          tick.OpenPrice,
		"highPrice":          tick.HighPrice,
		"lowPrice":           tick.LowPrice,
		"volume":             tick.Volume,
		"quoteVolume":        tick.QuoteVolume,
		"tradeCount":         tick.TradeCount,
		"openTime":           formatTime(tick.OpenTime),
		"closeTime":          formatTime(tick.CloseTime),
		"ts":                 formatTime(tick.UpdatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func trimID(s string) string {
	return strings.TrimSpace(s)
}
