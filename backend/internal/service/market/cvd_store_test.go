package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type bucketRows struct {
	rows []domain.TakerBucket
}

func (bucketRows) UpsertTakerBuckets(context.Context, []domain.TakerBucket) (int, error) {
	return 0, nil
}

func (b bucketRows) ListTakerBuckets(context.Context, string, string, time.Time, time.Time) ([]domain.TakerBucket, error) {
	return b.rows, nil
}

func (bucketRows) PurgeTakerBuckets(context.Context, time.Time) (int, error) { return 0, nil }

func TestGetCVD_UsesStoredMinutesWhenLiveBookIsDown(t *testing.T) {
	at := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Minute)
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{candles: []domain.Candle{{
			OpenTime: at, Close: "100", QuoteVolume: "1", TakerBuyQuote: "1",
		}}},
	}, &fakeSupply{}).
		WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
			domain.ExchangeBinance: &fakeTaker{err: errors.New("tape empty")},
		}).
		WithTakerBucketStore(bucketRows{rows: []domain.TakerBucket{{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: at,
			BuyNotional: 80, SellNotional: 20,
		}}})
	got, err := svc.GetCVD(context.Background(), "binance", "BTCUSDT", "24h")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 || got.Venues[0].Error != "" || len(got.Venues[0].Points) == 0 {
		t.Fatalf("%+v", got.Venues)
	}
}

func TestGetAbsorption_UsesStoredMinutesWhenLiveBookIsDown(t *testing.T) {
	at := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Minute)
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{candles: []domain.Candle{{
			OpenTime: at, Close: "100", High: "101", Low: "99", QuoteVolume: "1", TakerBuyQuote: "1",
		}}},
	}, &fakeSupply{}).
		WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
			domain.ExchangeBinance: &fakeTaker{err: errors.New("tape empty")},
		}).
		WithTakerBucketStore(bucketRows{rows: []domain.TakerBucket{{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: at,
			BuyNotional: 80, SellNotional: 20,
		}}})
	got, err := svc.GetAbsorption(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 || got.Venues[0].Error != "" {
		t.Fatalf("%+v", got.Venues)
	}
}
