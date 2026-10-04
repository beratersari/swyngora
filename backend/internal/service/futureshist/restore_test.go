package futureshist

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type restoreStore struct {
	rows  map[string][]domain.TakerBucket
	calls int
}

func (s *restoreStore) UpsertTakerBuckets(context.Context, []domain.TakerBucket) (int, error) {
	return 0, nil
}

func (s *restoreStore) ListTakerBuckets(_ context.Context, exchange, symbol string, _, _ time.Time) ([]domain.TakerBucket, error) {
	s.calls++
	return s.rows[exchange+"|"+symbol], nil
}

func (s *restoreStore) PurgeTakerBuckets(context.Context, time.Time) (int, error) {
	return 0, nil
}

func TestRestoreTakerMemory_LoadsLiquidNamesOnce(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := &restoreStore{rows: map[string][]domain.TakerBucket{
		"binance|BTCUSDT": {{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			Start: now.Add(-time.Hour), BuyNotional: 10, SellNotional: 4,
		}},
		"bybit|ETHUSDT": {{
			Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT",
			Start: now.Add(-2 * time.Hour), BuyNotional: 3,
		}},
	}}
	binance := domain.NewTakerBook()
	bybit := domain.NewTakerBook()
	books := map[domain.Exchange]*domain.TakerBook{
		domain.ExchangeBinance: binance,
		domain.ExchangeBybit:   bybit,
	}
	symbols := append([]string{}, domain.DefaultFuturesHistorySymbols...)
	symbols = append(symbols, "ETHUSDT")
	n := RestoreTakerMemory(context.Background(), store, books, symbols, now)
	if n != 2 {
		t.Fatalf("bars %d", n)
	}
	if got := binance.Buckets(domain.ExchangeBinance, "BTCUSDT"); len(got) != 1 || got[0].BuyNotional != 10 {
		t.Fatalf("btc %+v", got)
	}
	if got := bybit.Buckets(domain.ExchangeBybit, "ETHUSDT"); len(got) != 1 || got[0].BuyNotional != 3 {
		t.Fatalf("eth %+v", got)
	}
	if !bybit.Restored(domain.ExchangeBybit, "SOLUSDT") {
		t.Fatal("empty pair should be marked so startup does not ask again")
	}
	calls := store.calls
	if again := RestoreTakerMemory(context.Background(), store, books, symbols, now); again != 0 || store.calls != calls {
		t.Fatalf("second pass bars %d calls %d -> %d", again, calls, store.calls)
	}
}
