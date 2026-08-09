package realtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeTicker struct {
	ticks map[string]*domain.Ticker24h
	err   error
}

func (f *fakeTicker) GetTicker24h(_ context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	if f.err != nil {
		return nil, f.err
	}
	if t, ok := f.ticks[exchange+":"+symbol]; ok {
		return t, nil
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "1"}, nil
}

type fakeAccess struct {
	deny     error
	view     *domain.PortfolioView
	orders   []domain.PendingOrder
	sawActor string
	sawBook  string
}

func (f *fakeAccess) CanViewPortfolio(_ context.Context, actor, book string) error {
	f.sawActor, f.sawBook = actor, book
	return f.deny
}

func (f *fakeAccess) RealtimeSnapshot(_ context.Context, actor, book string) (*domain.PortfolioView, []domain.PendingOrder, error) {
	f.sawActor, f.sawBook = actor, book
	if f.deny != nil {
		return nil, nil, f.deny
	}
	v := f.view
	if v == nil {
		v = &domain.PortfolioView{ID: book, ClientID: "owner", Name: "Main", Role: domain.PortfolioRoleViewer}
	}
	return v, f.orders, nil
}

func drain(t *testing.T, ch <-chan any, n int) []any {
	t.Helper()
	out := make([]any, 0, n)
	deadline := time.After(time.Second)
	for len(out) < n {
		select {
		case m, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, m)
		case <-deadline:
			t.Fatalf("timeout waiting for %d messages, got %d", n, len(out))
		}
	}
	return out
}

func TestHub_SubscribePricesPushesSnapshotAndPump(t *testing.T) {
	m := &fakeTicker{ticks: map[string]*domain.Ticker24h{
		"binance:BTCUSDT": {Symbol: "BTCUSDT", LastPrice: "100", PriceChangePercent: "1.5"},
	}}
	h := NewHub(Options{Market: m, Interval: time.Hour})
	s := h.Register("s1", "client-a")
	hello := drain(t, s.Out, 1)
	env, ok := hello[0].(Outbound)
	if !ok || env["type"] != domain.RealtimeTypeHello {
		t.Fatalf("hello=%v", hello[0])
	}

	ref, err := domain.NormalizeSymbolRef("binance", "btcusdt")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.SubscribePrices(context.Background(), s, []domain.SymbolRef{ref}); err != nil {
		t.Fatal(err)
	}
	got := drain(t, s.Out, 2) // ack + price snapshot
	ack, _ := got[0].(Outbound)
	if ack["type"] != domain.RealtimeTypeAck || ack["op"] != domain.RealtimeOpSubscribePrices {
		t.Fatalf("ack=%v", got[0])
	}
	price, _ := got[1].(Outbound)
	if price["type"] != domain.RealtimeTypePrice || price["lastPrice"] != "100" || price["symbol"] != "BTCUSDT" {
		t.Fatalf("price=%v", got[1])
	}

	m.ticks["binance:BTCUSDT"] = &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "101"}
	if err := h.PumpOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	pumped := drain(t, s.Out, 1)
	p2, _ := pumped[0].(Outbound)
	if p2["lastPrice"] != "101" {
		t.Fatalf("pump=%v", pumped[0])
	}
	h.Unregister(s)
}

func TestHub_SubscribePortfolioRequiresAccess(t *testing.T) {
	acc := &fakeAccess{deny: domain.ErrForbidden}
	h := NewHub(Options{Access: acc, Interval: time.Hour})
	s := h.Register("s1", "viewer")
	_ = drain(t, s.Out, 1)
	err := h.SubscribePortfolio(context.Background(), s, "book-1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
	acc.deny = nil
	acc.view = &domain.PortfolioView{ID: "book-1", ClientID: "owner", Name: "Swing", Role: domain.PortfolioRoleTrader}
	if err := h.SubscribePortfolio(context.Background(), s, "book-1"); err != nil {
		t.Fatal(err)
	}
	msgs := drain(t, s.Out, 2)
	ack, _ := msgs[0].(Outbound)
	if ack["op"] != domain.RealtimeOpSubscribePortfolio {
		t.Fatalf("ack=%v", msgs[0])
	}
	ch, ok := msgs[1].(domain.PortfolioChange)
	if !ok || ch.Reason != domain.PortfolioChangeSnapshot || ch.View == nil || ch.View.ID != "book-1" {
		t.Fatalf("change=%#v", msgs[1])
	}
}

func TestHub_PortfolioEventOnlyToAuthorizedSubscribers(t *testing.T) {
	acc := &fakeAccess{}
	h := NewHub(Options{Access: acc, Interval: time.Hour})
	owner := h.Register("s-owner", "owner")
	other := h.Register("s-other", "stranger")
	_ = drain(t, owner.Out, 1)
	_ = drain(t, other.Out, 1)
	if err := h.SubscribePortfolio(context.Background(), owner, "book-1"); err != nil {
		t.Fatal(err)
	}
	_ = drain(t, owner.Out, 2)

	h.OnPortfolioChange(domain.PortfolioChange{
		PortfolioID: "book-1",
		Reason:      domain.PortfolioChangeOrderFilled,
		Trade:       &domain.Trade{ID: "t1", Quantity: 1},
	})
	got := drain(t, owner.Out, 1)
	ch := got[0].(domain.PortfolioChange)
	if ch.Trade == nil || ch.Trade.ID != "t1" {
		t.Fatalf("got=%#v", got[0])
	}
	select {
	case m := <-other.Out:
		t.Fatalf("stranger got %v", m)
	case <-time.After(30 * time.Millisecond):
	}

	acc.deny = domain.ErrForbidden
	h.OnPortfolioChange(domain.PortfolioChange{PortfolioID: "book-1", Reason: domain.PortfolioChangeOrderPlaced})
	denied := drain(t, owner.Out, 1)
	env, _ := denied[0].(Outbound)
	if env["type"] != domain.RealtimeTypeError {
		t.Fatalf("want error after revoke, got %#v", denied[0])
	}
}

func TestHub_UnsubscribePricesStopsPump(t *testing.T) {
	m := &fakeTicker{ticks: map[string]*domain.Ticker24h{
		"binance:ETHUSDT": {Symbol: "ETHUSDT", LastPrice: "9"},
	}}
	h := NewHub(Options{Market: m, Interval: time.Hour})
	s := h.Register("s1", "c")
	_ = drain(t, s.Out, 1)
	ref, _ := domain.NormalizeSymbolRef("binance", "ETHUSDT")
	if err := h.SubscribePrices(context.Background(), s, []domain.SymbolRef{ref}); err != nil {
		t.Fatal(err)
	}
	_ = drain(t, s.Out, 2)
	h.UnsubscribePrices(s, []domain.SymbolRef{ref})
	_ = drain(t, s.Out, 1) // ack
	if err := h.PumpOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-s.Out:
		t.Fatalf("unexpected after unsubscribe: %v", m)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestHub_MaxSymbols(t *testing.T) {
	h := NewHub(Options{Market: &fakeTicker{}, Interval: time.Hour})
	s := h.Register("s1", "c")
	_ = drain(t, s.Out, 1)
	refs := make([]domain.SymbolRef, 0, domain.MaxRealtimePriceSymbols+1)
	for i := 0; i < domain.MaxRealtimePriceSymbols+1; i++ {
		refs = append(refs, domain.SymbolRef{Exchange: domain.ExchangeBinance, Symbol: "S" + string(rune('A'+i%26)) + string(rune('0'+i/26))})
	}
	err := h.SubscribePrices(context.Background(), s, refs)
	if err == nil {
		t.Fatal("expected max symbols error")
	}
}
