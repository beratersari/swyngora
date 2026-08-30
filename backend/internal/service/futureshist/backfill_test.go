package futureshist

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type histStub struct {
	calls []domain.LiquidationHistoryQuery
	res   domain.LiquidationHistoryResult
	err   error
}

func (s *histStub) ListLiquidationHistory(_ context.Context, q domain.LiquidationHistoryQuery) (domain.LiquidationHistoryResult, error) {
	s.calls = append(s.calls, q)
	return s.res, s.err
}

func TestBackfiller_FillDoesNotCrossVenue(t *testing.T) {
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	book := domain.NewLiquidationBook()
	book.UseClock(func() time.Time { return cur })

	book.SetLive(domain.ExchangeBybit, true)
	book.MarkWatch(domain.ExchangeBybit, "BTCUSDT")
	cur = now.Add(time.Minute)
	book.SetLive(domain.ExchangeBybit, false)
	cur = now.Add(16 * time.Minute)
	book.SetLive(domain.ExchangeBybit, true)

	mid := now.Add(8 * time.Minute)
	stub := &histStub{res: domain.LiquidationHistoryResult{
		Events: []domain.LiquidationEvent{
			{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Price: 1, Quantity: 1, Notional: 10, Time: mid},
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Price: 1, Quantity: 1, Notional: 99, Time: mid},
		},
		CoveredFrom: now.Add(time.Minute),
		CoveredTo:   now.Add(16 * time.Minute),
	}}
	st := &memStore{}
	bf := &Backfiller{
		Book:    book,
		Hist:    &Service{Store: st},
		Sources: map[domain.Exchange]domain.LiquidationHistoryPort{domain.ExchangeBybit: stub},
		Seeds:   []string{"BTCUSDT"},
	}
	bf.fillVenue(context.Background(), domain.ExchangeBybit, stub)
	if len(stub.calls) != 1 {
		t.Fatalf("calls %d", len(stub.calls))
	}
	if stub.calls[0].Exchange != domain.ExchangeBybit {
		t.Fatalf("query %+v", stub.calls[0])
	}
	feed := book.Feed("bybit")
	if len(feed.Venues) != 1 || len(feed.Venues[0].Gaps) != 0 || feed.Venues[0].MissingSeconds != 0 {
		t.Fatalf("gap should be gone %+v", feed)
	}
	if windowCount(book.Snapshot("bybit", "BTCUSDT"), domain.LiquidationWindow1h) != 1 {
		t.Fatal("bybit print missing")
	}
	if windowCount(book.Snapshot("binance", "BTCUSDT"), domain.LiquidationWindow1h) != 0 {
		t.Fatal("binance must stay empty")
	}
	st.mu.Lock()
	n := len(st.liq)
	st.mu.Unlock()
	if n != 1 {
		t.Fatalf("persisted %d", n)
	}
}

func TestBackfiller_PartialKeepsMissingSeconds(t *testing.T) {
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	book := domain.NewLiquidationBook()
	book.UseClock(func() time.Time { return cur })
	book.SetLive(domain.ExchangeBinance, true)
	cur = now.Add(time.Minute)
	book.SetLive(domain.ExchangeBinance, false)
	cur = now.Add(16 * time.Minute)
	book.SetLive(domain.ExchangeBinance, true)

	stub := &histStub{res: domain.LiquidationHistoryResult{
		Events:      nil,
		CoveredFrom: now.Add(11 * time.Minute),
		CoveredTo:   now.Add(16 * time.Minute),
	}}
	bf := &Backfiller{
		Book:    book,
		Sources: map[domain.Exchange]domain.LiquidationHistoryPort{domain.ExchangeBinance: stub},
	}
	bf.fillVenue(context.Background(), domain.ExchangeBinance, stub)
	feed := book.Feed("binance")
	if len(feed.Venues[0].Gaps) != 1 {
		t.Fatalf("expected leftover gap %+v", feed.Venues[0].Gaps)
	}
	if feed.Venues[0].MissingSeconds < 9*60 || feed.Venues[0].MissingSeconds > 11*60 {
		t.Fatalf("missing %d", feed.Venues[0].MissingSeconds)
	}
}

func TestBackfiller_UnavailableLeavesGap(t *testing.T) {
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	book := domain.NewLiquidationBook()
	book.UseClock(func() time.Time { return cur })
	book.SetLive(domain.ExchangeBybit, true)
	cur = now.Add(time.Minute)
	book.SetLive(domain.ExchangeBybit, false)
	cur = now.Add(16 * time.Minute)
	book.SetLive(domain.ExchangeBybit, true)

	stub := &histStub{} // no covered range
	bf := &Backfiller{Book: book, Sources: map[domain.Exchange]domain.LiquidationHistoryPort{domain.ExchangeBybit: stub}}
	bf.fillVenue(context.Background(), domain.ExchangeBybit, stub)
	feed := book.Feed("bybit")
	if len(feed.Venues[0].Gaps) != 1 || feed.Venues[0].MissingSeconds < 14*60 {
		t.Fatalf("gap must remain %+v", feed.Venues[0])
	}
}

func windowCount(snap *domain.LiquidationSnapshot, id string) int {
	if snap == nil {
		return 0
	}
	for _, w := range snap.Windows {
		if w.Window == id {
			return w.Count
		}
	}
	return 0
}
