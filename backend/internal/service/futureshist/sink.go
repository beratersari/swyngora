package futureshist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const coverageSaveEvery = 30 * time.Second

// PersistSink writes liquidation events to the book and to SQLite without
// blocking the websocket path. Duplicate events are ignored by the store.
type PersistSink struct {
	Book     *domain.LiquidationBook
	Hist     *Service
	Backfill *Backfiller
	ch       chan domain.LiquidationEvent
	now      func() time.Time
}

// NewPersistSink starts a background writer. Close by canceling the context
// passed to Start — queued prints and coverage clocks are flushed first.
func NewPersistSink(book *domain.LiquidationBook, hist *Service) *PersistSink {
	return &PersistSink{
		Book: book,
		Hist: hist,
		ch:   make(chan domain.LiquidationEvent, 16384),
		now:  time.Now,
	}
}

// Start drains persisted events until ctx is done, then flushes the queue.
func (s *PersistSink) Start(ctx context.Context) {
	if s == nil {
		return
	}
	tick := time.NewTicker(coverageSaveEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			s.drain()
			s.saveCoverage()
			return
		case e := <-s.ch:
			s.saveEvent(e)
		case <-tick.C:
			s.saveCoverage()
		}
	}
}

func (s *PersistSink) saveEvent(e domain.LiquidationEvent) {
	if s != nil && s.Hist != nil {
		s.Hist.SaveLiquidation(context.Background(), e)
	}
}

func (s *PersistSink) drain() {
	if s == nil || s.ch == nil {
		return
	}
	for {
		select {
		case e := <-s.ch:
			s.saveEvent(e)
		default:
			return
		}
	}
}

func (s *PersistSink) saveCoverage() {
	if s == nil || s.Book == nil || s.Hist == nil {
		return
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	s.Hist.SaveCoverage(context.Background(), s.Book.CoverageSnapshot(now))
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
		s.saveEvent(e)
		return
	}
	select {
	case s.ch <- e:
	default:
		// Writer is behind — do not drop the 24h history.
		s.saveEvent(e)
	}
}

// SetLive forwards venue liveness to the book and persists coverage.
// A reconnect schedules a same-venue history fill for closed gaps.
func (s *PersistSink) SetLive(ex domain.Exchange, live bool) {
	if s != nil && s.Book != nil {
		s.Book.SetLive(ex, live)
	}
	if live && s != nil && s.Backfill != nil {
		s.Backfill.Schedule(ex)
	}
	if !live {
		s.saveCoverage()
	}
}

// NoteSeen records that the venue socket delivered a payload.
func (s *PersistSink) NoteSeen(ex domain.Exchange) {
	if s != nil && s.Book != nil {
		s.Book.NoteSeen(ex)
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

// RestoreBook loads the last 24h of persisted liquidations and coverage
// clocks into memory. Binance and Bybit rows stay on their own venue.
func RestoreBook(ctx context.Context, book *domain.LiquidationBook, hist *Service, now time.Time) int {
	if book == nil || hist == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	ev := hist.LoadRecentLiquidations(ctx, now.Add(-24*time.Hour))
	for _, e := range ev {
		book.Record(e)
	}
	cov := hist.LoadCoverage(ctx)
	if len(cov) == 0 {
		cov = domain.CoverageFromEvents(ev, now)
	}
	for _, c := range cov {
		book.RestoreTracking(c.Exchange, c.Symbol, c.FirstWatch, c.Live)
		book.RestoreFeed(c, now)
	}
	if len(cov) == 0 {
		// Upgrade path: last print per venue from restored events.
		for _, e := range ev {
			book.NoteSeen(e.Exchange)
		}
	}
	return len(ev)
}
