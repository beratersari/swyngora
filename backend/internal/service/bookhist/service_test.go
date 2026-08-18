package bookhist

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type memStore struct {
	rows []domain.BookHistorySnapshot
}

func (m *memStore) InsertSnapshot(_ context.Context, rec domain.BookHistorySnapshot) (bool, error) {
	for _, r := range m.rows {
		if r.Exchange == rec.Exchange && r.Symbol == rec.Symbol && r.SampledAt.Equal(rec.SampledAt) {
			return false, nil
		}
	}
	m.rows = append(m.rows, rec)
	return true, nil
}

func (m *memStore) NearestAt(_ context.Context, exchange, symbol string, at time.Time) (*domain.BookHistorySnapshot, error) {
	var best *domain.BookHistorySnapshot
	for i := range m.rows {
		r := &m.rows[i]
		if string(r.Exchange) != exchange || r.Symbol != symbol {
			continue
		}
		if r.SampledAt.After(at) {
			continue
		}
		if best == nil || r.SampledAt.After(best.SampledAt) {
			best = r
		}
	}
	return best, nil
}

func (m *memStore) ListSnapshots(_ context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	var out []domain.BookHistorySnapshot
	for _, r := range m.rows {
		if q.Exchange != "" && q.Exchange != "all" && string(r.Exchange) != q.Exchange {
			continue
		}
		if r.Symbol != q.Symbol {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *memStore) PurgeOlderThan(_ context.Context, cutoff time.Time) (int, error) {
	kept := m.rows[:0]
	n := 0
	for _, r := range m.rows {
		if r.SampledAt.Before(cutoff) {
			n++
			continue
		}
		kept = append(kept, r)
	}
	m.rows = kept
	return n, nil
}

type memBooks struct {
	book *domain.RawOrderBook
	err  error
}

func (m memBooks) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	return nil, nil
}
func (m memBooks) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) { return nil, nil }
func (m memBooks) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return m.book, m.err
}
func (m memBooks) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) { return nil, nil }
func (m memBooks) ListProductTags(context.Context) ([]string, error)            { return nil, nil }
func (m memBooks) TagsByBase(context.Context) (map[string][]string, error)      { return nil, nil }

func TestSaveAndCompare(t *testing.T) {
	st := &memStore{}
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	svc := &Service{
		Store: st,
		Books: map[domain.Exchange]domain.MarketDataPort{
			domain.ExchangeBinance: memBooks{book: &domain.RawOrderBook{
				Symbol: "BTCUSDT",
				Bids:   []domain.PriceLevel{{Price: 99, Quantity: 5}, {Price: 98, Quantity: 4}},
				Asks:   []domain.PriceLevel{{Price: 101, Quantity: 2}, {Price: 102, Quantity: 3}},
				Live:   true,
			}},
		},
		Seeds: []string{"BTCUSDT"},
	}
	ok, err := svc.SaveSymbol(context.Background(), domain.ExchangeBinance, "BTCUSDT", t0)
	if err != nil || !ok {
		t.Fatalf("save %v %v", ok, err)
	}
	// Second sample: more ask size, less bid.
	svc.Books[domain.ExchangeBinance] = memBooks{book: &domain.RawOrderBook{
		Symbol: "BTCUSDT",
		Bids:   []domain.PriceLevel{{Price: 100, Quantity: 1}, {Price: 99, Quantity: 1}},
		Asks:   []domain.PriceLevel{{Price: 102, Quantity: 8}, {Price: 103, Quantity: 6}},
		Live:   true,
	}}
	ok, err = svc.SaveSymbol(context.Background(), domain.ExchangeBinance, "BTCUSDT", t0.Add(5*time.Minute))
	if err != nil || !ok {
		t.Fatalf("save2 %v %v", ok, err)
	}
	diff, err := svc.Compare(context.Background(), "binance", "BTCUSDT", t0, t0.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if diff.Summary == "" || len(diff.Gained)+len(diff.Lost) == 0 {
		t.Fatalf("%+v", diff)
	}
	at, err := svc.SnapshotAt(context.Background(), "binance", "BTCUSDT", t0.Add(30*time.Second))
	if err != nil || at.Mid <= 0 {
		t.Fatalf("%+v %v", at, err)
	}
}

func TestSnapshotAt_Missing(t *testing.T) {
	svc := &Service{Store: &memStore{}}
	_, err := svc.SnapshotAt(context.Background(), "binance", "BTCUSDT", time.Now().UTC())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestNotConfigured(t *testing.T) {
	svc := &Service{}
	_, err := svc.List(context.Background(), domain.BookHistoryQuery{Symbol: "BTCUSDT"})
	if !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("%v", err)
	}
}
