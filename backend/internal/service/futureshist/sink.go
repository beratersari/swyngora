package futureshist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// PersistSink writes liquidation events to the book and to SQLite without
// blocking the websocket path. Duplicate events are ignored by the store.
type PersistSink struct {
	Book *domain.LiquidationBook
	Hist *Service
	ch   chan domain.LiquidationEvent
}

// NewPersistSink starts a background writer. Close by canceling the context
// passed to Start.
func NewPersistSink(book *domain.LiquidationBook, hist *Service) *PersistSink {
	return &PersistSink{
		Book: book,
		Hist: hist,
		ch:   make(chan domain.LiquidationEvent, 1024),
	}
}

// Start drains persisted events until ctx is done.
func (s *PersistSink) Start(ctx context.Context) {
	if s == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-s.ch:
			if s.Hist != nil {
				s.Hist.SaveLiquidation(context.Background(), e)
			}
		}
	}
}

// Record keeps the in-memory book up to date and queues a durable write.
func (s *PersistSink) Record(e domain.LiquidationEvent) {
	if s == nil {
		return
	}
	if s.Book != nil {
		s.Book.Record(e)
	}
	if s.Hist != nil {
		s.Hist.NoteSymbol(e.Symbol)
	}
	if s.ch == nil {
		return
	}
	select {
	case s.ch <- e:
	default:
		// Drop persist if the writer is behind; memory book still has the event.
	}
}

// SetLive forwards venue liveness to the book.
func (s *PersistSink) SetLive(ex domain.Exchange, live bool) {
	if s != nil && s.Book != nil {
		s.Book.SetLive(ex, live)
	}
}

// MarkWatch forwards Bybit per-symbol coverage to the book.
func (s *PersistSink) MarkWatch(ex domain.Exchange, symbol string) {
	if s != nil && s.Book != nil {
		s.Book.MarkWatch(ex, symbol)
	}
	if s != nil && s.Hist != nil {
		s.Hist.NoteSymbol(symbol)
	}
}

// RestoreBook loads the last 24h of persisted liquidations into memory.
func RestoreBook(ctx context.Context, book *domain.LiquidationBook, hist *Service, now time.Time) int {
	if book == nil || hist == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ev := hist.LoadRecentLiquidations(ctx, now.Add(-24*time.Hour))
	for _, e := range ev {
		book.Record(e)
	}
	return len(ev)
}
