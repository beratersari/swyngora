package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type ladderBookHist struct {
	snaps []domain.BookHistorySnapshot
}

func (s ladderBookHist) SnapshotAt(_ context.Context, _, _ string, at time.Time) (*domain.BookHistorySnapshot, error) {
	if len(s.snaps) == 0 {
		return nil, domain.ErrNotFound
	}
	best := s.snaps[0]
	for _, row := range s.snaps {
		if !row.SampledAt.After(at) && row.SampledAt.After(best.SampledAt) {
			best = row
		}
	}
	return &best, nil
}
func (s ladderBookHist) List(_ context.Context, _ domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return s.snaps, nil
}
func (s ladderBookHist) ListLadders(_ context.Context, _ domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return s.snaps, nil
}
func (s ladderBookHist) Compare(_ context.Context, _, _ string, from, to time.Time) (*domain.BookHistoryDiff, error) {
	d := domain.CompareBookHistory(s.snaps[0], s.snaps[len(s.snaps)-1])
	return &d, nil
}
func (ladderBookHist) Note(string, string) {}

func TestGetBookLiquidity_BidPullVsTrade(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	from := now.Add(-5 * time.Minute)
	snaps := []domain.BookHistorySnapshot{
		{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", SampledAt: from, GroupSize: 10,
			Bids: []domain.BookHistoryLevel{{Price: 65500, Notional: 8_000_000}},
			Asks: []domain.BookHistoryLevel{{Price: 65520, Notional: 4_000_000}},
		},
		{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", SampledAt: now, GroupSize: 10,
			Bids: []domain.BookHistoryLevel{{Price: 65500, Notional: 2_000_000}},
			Asks: []domain.BookHistoryLevel{{Price: 65520, Notional: 1_000_000}},
		},
	}
	prints := []domain.TakerPrint{
		{Side: domain.TakerSideSell, Price: 65500, Notional: 3_000_000, Time: from.Add(2 * time.Minute)},
		{Side: domain.TakerSideBuy, Price: 65520, Notional: 3_000_000, Time: from.Add(3 * time.Minute)},
	}
	fm := &fakeMarket{prints: map[string][]domain.TakerPrint{"BTCUSDT": prints}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{}).WithBookHistory(ladderBookHist{snaps: snaps})
	got, err := svc.GetBookLiquidity(context.Background(), "binance", "BTCUSDT", "5m", &from, &now, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	v := got.Venues[0]
	if v.Bids.Pulled != 3_000_000 || v.Bids.Traded != 3_000_000 {
		t.Fatalf("bids %+v", v.Bids)
	}
	if v.Asks.Pulled != 0 || v.Asks.Traded != 3_000_000 {
		t.Fatalf("asks %+v", v.Asks)
	}
}

func TestGetBookLiquidity_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetBookLiquidity(context.Background(), "binance", "", "5m", nil, nil, 0, 0); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetBookLiquidity_BadWindow(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetBookLiquidity(context.Background(), "binance", "BTCUSDT", "24h", nil, nil, 0, 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
