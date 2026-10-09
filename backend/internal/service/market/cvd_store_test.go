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
	got, err := svc.GetCVD(context.Background(), "binance", "BTCUSDT", "24h", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 || got.Venues[0].Error != "" || len(got.Venues[0].Points) == 0 {
		t.Fatalf("%+v", got.Venues)
	}
}

func TestGetCVD_ClickAnchorSumsStoredMinutes(t *testing.T) {
	now := time.Now().UTC()
	late := now.Add(-35 * time.Minute).Truncate(5 * time.Minute)
	early := late.Add(-time.Hour)
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{candles: []domain.Candle{{
			OpenTime: late, Close: "100", QuoteVolume: "1", TakerBuyQuote: "1",
		}}},
	}, &fakeSupply{}).
		WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
			domain.ExchangeBinance: &fakeTaker{err: errors.New("tape empty")},
		}).
		WithTakerBucketStore(bucketRows{rows: []domain.TakerBucket{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: early, BuyNotional: 100, SellNotional: 0},
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: late, BuyNotional: 0, SellNotional: 40},
		}})
	got, err := svc.GetCVD(context.Background(), "binance", "BTCUSDT", "24h", late.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 || got.Venues[0].Error != "" {
		t.Fatalf("%+v", got.Venues)
	}
	var click domain.CVDAnchor
	var day bool
	for _, a := range got.Venues[0].Anchors {
		switch a.Anchor {
		case domain.CVDAnchorClick:
			click = a
		case domain.CVDAnchorDay:
			day = true
		}
	}
	if !day || !click.From.Equal(late) || click.CVD != -40 || !click.Complete {
		t.Fatalf("day=%v click=%+v", day, click)
	}
	if _, err := svc.GetCVD(context.Background(), "binance", "BTCUSDT", "24h", "yesterday"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad anchor %v", err)
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
