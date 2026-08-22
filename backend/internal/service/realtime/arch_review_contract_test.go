package realtime

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestArchReview_WSPriceOmitsHalted(t *testing.T) {
	m := &fakeTicker{ticks: map[string]*domain.Ticker24h{
		"binance:BTCUSDT": {
			Symbol: "BTCUSDT", LastPrice: "42.5", Halted: true,
			PriceChangePercent: "-9.1",
		},
	}}
	h := NewHub(Options{Market: m, Interval: time.Hour})
	s := h.Register("halt-1", "client-a")
	_ = drain(t, s.Out, 1)

	ref, err := domain.NormalizeSymbolRef("binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.SubscribePrices(context.Background(), s, []domain.SymbolRef{ref}); err != nil {
		t.Fatal(err)
	}
	got := drain(t, s.Out, 2)
	price, _ := got[1].(Outbound)
	if price["type"] != domain.RealtimeTypePrice || price["lastPrice"] != "42.5" {
		t.Fatalf("price=%v", got[1])
	}
	if _, ok := price["halted"]; ok {
		return
	}
	t.Errorf("CONFIRMED: WS price snapshot lastPrice=%v omits halted (REST ticker.Halted=true keys=%v)", price["lastPrice"], keysOf(price))
}

func keysOf(m Outbound) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
