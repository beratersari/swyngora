package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetLiquidationOpenLevels_UsesHeatmapHistory(t *testing.T) {
	now := time.Now().UTC()
	var candles []domain.Candle
	var oi []domain.FuturesSnapshot
	for i := 0; i < 50; i++ {
		ts := now.Add(-time.Duration(50-i) * 30 * time.Minute)
		candles = append(candles, domain.Candle{CloseTime: ts, Close: "100000", High: "100000", Low: "100000"})
		oi = append(oi, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance,
			SampledAt: ts, Value: 5_000_000,
		})
	}
	fm := &fakeMarket{candles: candles, ticker: &domain.Ticker24h{LastPrice: "100000"}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
		domain.ExchangeBybit:   fm,
	}, &fakeSupply{}).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeBinance: oiStub{},
		domain.ExchangeBybit:   oiStub{},
		domain.ExchangeOKX:     oiStub{},
	}).WithFuturesCandles(map[domain.Exchange]domain.FuturesCandlePort{
		domain.ExchangeOKX: huntCandles{bars: candles},
	})
	svc.SetFuturesHistory(huntHeatFut{oi: oi})
	got, err := svc.GetLiquidationOpenLevels(context.Background(), "all", "BTCUSDT", "24h", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || got.Range != "24h" || got.Note == "" {
		t.Fatalf("header %+v", got)
	}
	if len(got.LeverageMix) != len(domain.DefaultHuntLeverageMix) {
		t.Fatalf("mix %d", len(got.LeverageMix))
	}
	if !got.Binance.Definite || got.Binance.OpenLong <= 0 || len(got.Binance.Segments) == 0 {
		t.Fatalf("binance %+v", got.Binance)
	}
	if !got.OKX.Definite || got.OKX.AccountBlend != "even" {
		t.Fatalf("okx blend %s definite %v", got.OKX.AccountBlend, got.OKX.Definite)
	}
	if !got.Combined.Definite || len(got.MissingVenues) != 0 {
		t.Fatalf("combined definite=%v missing=%v", got.Combined.Definite, got.MissingVenues)
	}
	if len(got.Binance.Anchors) != 2 {
		t.Fatalf("anchors %+v", got.Binance.Anchors)
	}
	for _, anchor := range got.Binance.Anchors {
		if anchor.Anchor == domain.OpenLevelAnchor4h && !anchor.Complete {
			t.Fatalf("4h anchor %+v", anchor)
		}
	}
}

func TestGetLiquidationOpenLevels_BadRange(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{})
	_, err := svc.GetLiquidationOpenLevels(context.Background(), "all", "BTCUSDT", "9h", "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetLiquidationOpenLevels_BadAnchor(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{})
	_, err := svc.GetLiquidationOpenLevels(context.Background(), "all", "BTCUSDT", "24h", "yesterday")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
