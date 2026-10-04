package futureshist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// RestoreTakerMemory loads the last TakerMemoryRetain of stored minutes into
// each live book. A pair is asked once, including when SQLite has no rows.
// Returns how many minute bars were copied.
func RestoreTakerMemory(ctx context.Context, store domain.TakerBucketStore, books map[domain.Exchange]*domain.TakerBook, symbols []string, now time.Time) int {
	if store == nil || now.IsZero() || len(books) == 0 {
		return 0
	}
	symbols = domain.MergeLiquidSymbols(nil, symbols, domain.LiquidRecorderLimit)
	if len(symbols) == 0 {
		return 0
	}
	now = now.UTC()
	from := now.Add(-domain.TakerMemoryRetain)
	n := 0
	for ex, book := range books {
		if book == nil {
			continue
		}
		for _, sym := range symbols {
			if book.Restored(ex, sym) {
				continue
			}
			recs, err := store.ListTakerBuckets(ctx, string(ex), sym, from, now)
			if err != nil {
				continue
			}
			book.MarkRestored(ex, sym)
			if len(recs) == 0 {
				continue
			}
			book.LoadBuckets(recs)
			n += len(recs)
		}
	}
	return n
}
