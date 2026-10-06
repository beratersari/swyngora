package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type tapeTakerStore struct {
	rows []domain.TakerBucket
}

func (t tapeTakerStore) UpsertTakerBuckets(context.Context, []domain.TakerBucket) (int, error) {
	return 0, nil
}
func (t tapeTakerStore) ListTakerBuckets(context.Context, string, string, time.Time, time.Time) ([]domain.TakerBucket, error) {
	return t.rows, nil
}
func (t tapeTakerStore) PurgeTakerBuckets(context.Context, time.Time) (int, error) {
	return 0, nil
}

type tapeFutHist struct {
	events []domain.LiquidationEvent
}

func (t tapeFutHist) History(context.Context, domain.FuturesHistoryQuery) (any, error) {
	return t.events, nil
}

type tapeBookHist struct {
	rows []domain.BookHistorySnapshot
}

func (t tapeBookHist) SnapshotAt(context.Context, string, string, time.Time) (*domain.BookHistorySnapshot, error) {
	return nil, nil
}
func (t tapeBookHist) List(context.Context, domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return t.rows, nil
}
func (t tapeBookHist) ListLadders(context.Context, domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return t.rows, nil
}
func (t tapeBookHist) Compare(context.Context, string, string, time.Time, time.Time) (*domain.BookHistoryDiff, error) {
	return nil, nil
}
func (t tapeBookHist) Note(string, string) {}

func TestGetTapeBacktest_CVDFromStoredMinutes(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	hour1 := now.Add(-72 * time.Hour)
	hour2 := hour1.Add(time.Hour)
	signalAt := hour2.Add(time.Hour)
	rows := append(fillServiceHour(domain.ExchangeBinance, hour1, 40, 60), fillServiceHour(domain.ExchangeBinance, hour2, 70, 30)...)
	step := 5 * time.Minute
	mkt := &fakeMarket{candles: []domain.Candle{
		{OpenTime: signalAt.Add(-step), Close: "100"},
		{OpenTime: signalAt.Add(time.Hour).Add(-step), Close: "110"},
	}}
	svc := New(mkt, &fakeSupply{}).WithTakerBucketStore(tapeTakerStore{rows: rows})
	got, err := svc.GetTapeBacktest(context.Background(), "binance", "BTCUSDT", "cvd", "buy", 0.65, hour1, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.PriceExchange != "binance" || len(got.Venues) != 1 || got.Venues[0].SignalCount != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Venues[0].Signals[0].Return1h == nil || *got.Venues[0].Signals[0].Return1h != 10 {
		t.Fatalf("%+v", got.Venues[0].Signals)
	}
}

func TestGetTapeBacktest_BadCondition(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetTapeBacktest(context.Background(), "binance", "BTCUSDT", "heatmap", "", 0, time.Time{}, time.Time{}); err == nil {
		t.Fatal("expected invalid condition")
	}
}

func fillServiceHour(ex domain.Exchange, start time.Time, buy, sell float64) []domain.TakerBucket {
	const minutes = 50
	out := make([]domain.TakerBucket, 0, minutes)
	for i := 0; i < minutes; i++ {
		out = append(out, domain.TakerBucket{
			Exchange: ex, Symbol: "BTCUSDT", Start: start.Add(time.Duration(i) * time.Minute),
			BuyNotional: buy / minutes, SellNotional: sell / minutes,
		})
	}
	return out
}
