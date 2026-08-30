package futureshist

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/futuresstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestRestoreBook_Last24hSeparateVenues(t *testing.T) {
	now := time.Now().UTC()
	st := &memStore{}
	hist := &Service{Store: st}
	_, _ = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 64000, Quantity: 1, Notional: 1000, Time: now.Add(-30 * time.Minute),
	})
	_, _ = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT", Side: domain.LiquidationSideShort,
		Price: 3000, Quantity: 2, Notional: 6000, Time: now.Add(-2 * time.Hour),
	})
	_, _ = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
		Price: 1, Quantity: 1, Notional: 1, Time: now.Add(-30 * time.Hour),
	})

	book := domain.NewLiquidationBook()
	n := RestoreBook(context.Background(), book, hist, now)
	if n != 2 {
		t.Fatalf("restore count %d (must drop older than 24h)", n)
	}
	bn := book.Snapshot("binance", "BTCUSDT")
	var w1h domain.LiquidationWindowTotals
	for _, w := range bn.Windows {
		if w.Window == domain.LiquidationWindow1h {
			w1h = w
		}
	}
	if w1h.Count != 1 || w1h.LongNotional != "1000" {
		t.Fatalf("binance 1h %+v", w1h)
	}
	bb := book.Snapshot("bybit", "ETHUSDT")
	var w4 domain.LiquidationWindowTotals
	for _, w := range bb.Windows {
		if w.Window == domain.LiquidationWindow4h {
			w4 = w
		}
	}
	if w4.Count != 1 || w4.ShortNotional != "6000" {
		t.Fatalf("bybit %+v", w4)
	}
	var bybitBTC int
	for _, w := range book.Snapshot("bybit", "BTCUSDT").Windows {
		bybitBTC += w.Count
	}
	if bybitBTC != 0 {
		t.Fatal("bybit BTC must stay empty")
	}
}

func TestRestoreBook_PersistedCoverageSurvivesReopen(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "futures.db")
	st, err := futuresstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 64000, Quantity: 1, Notional: 1000, Time: now.Add(-30 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = st.UpsertLiquidationCoverage(context.Background(), []domain.LiquidationCoverage{
		{Exchange: domain.ExchangeBinance, FirstWatch: now.Add(-24 * time.Hour), Live: 24 * time.Hour},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st2, err := futuresstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	book := domain.NewLiquidationBook()
	n := RestoreBook(context.Background(), book, &Service{Store: st2}, now)
	if n != 1 {
		t.Fatalf("restore %d", n)
	}
	snap := book.Snapshot("binance", "BTCUSDT")
	var w24 domain.LiquidationWindowTotals
	for _, w := range snap.Windows {
		if w.Window == domain.LiquidationWindow24h {
			w24 = w
		}
	}
	if !w24.Complete || w24.Count != 1 {
		t.Fatalf("24h after reopen %+v", w24)
	}
}

func TestPersistSink_FlushOnStop(t *testing.T) {
	st := &memStore{}
	hist := &Service{Store: st}
	book := domain.NewLiquidationBook()
	sink := NewPersistSink(book, hist)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sink.Start(ctx)
		close(done)
	}()
	sink.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 10, Quantity: 1, Notional: 10, Time: time.Now().UTC(),
	})
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("start did not return")
	}
	st.mu.Lock()
	n := len(st.liq)
	st.mu.Unlock()
	if n != 1 {
		t.Fatalf("queued print lost on stop, have %d", n)
	}
}
