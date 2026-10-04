package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type printAggFut struct {
	huntHeatFut
	at time.Time
}

func (p printAggFut) AggregateLiquidationPrints(_ context.Context, q domain.MeasuredPrintQuery) (domain.MeasuredPrintScan, error) {
	ev := domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: q.Symbol, Side: domain.LiquidationSideLong,
		Price: 100000, Notional: 2500, Time: p.at,
	}
	return domain.BinMeasuredPrintEvents(q, []domain.LiquidationEvent{ev}), nil
}

func (p printAggFut) LiquidationPrintPriceSpan(context.Context, string, time.Time, time.Time, []domain.Exchange) (float64, float64, error) {
	return 0, 0, nil
}

func TestGetLiquidationPrintHeatmap_BookPrintAndCoverage(t *testing.T) {
	now := time.Now().UTC()
	var candles []domain.Candle
	for i := 0; i < 60; i++ {
		ts := now.Add(-time.Duration(60-i) * 30 * time.Minute)
		candles = append(candles, domain.Candle{CloseTime: ts, Close: "100000", High: "100200", Low: "99800"})
	}
	storedAt := now.Add(-12 * time.Hour)
	freshAt := now.Add(-30 * time.Minute)
	book := domain.NewLiquidationBook()
	book.RestoreTracking(domain.ExchangeBinance, "", now.Add(-48*time.Hour), 48*time.Hour)
	book.SetLive(domain.ExchangeBinance, true)
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100000, Quantity: 1, Notional: 2500, Time: storedAt,
	})
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
		Price: 100000, Quantity: 1, Notional: 100, Time: freshAt,
	})
	fm := &fakeMarket{candles: candles, ticker: &domain.Ticker24h{LastPrice: "100000"}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
		domain.ExchangeBybit:   fm,
	}, &fakeSupply{}).WithFuturesCandles(map[domain.Exchange]domain.FuturesCandlePort{
		domain.ExchangeOKX: huntCandles{bars: candles},
	}).WithLiquidations(book, nil)
	svc.SetFuturesHistory(printAggFut{at: storedAt})

	got, err := svc.GetLiquidationPrintHeatmap(context.Background(), "all", "BTCUSDT", "24h")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.HuntHeatmapKindPrints || len(got.LeverageMix) != 0 || len(got.Review.Binance.Horizons) != 0 {
		t.Fatalf("kind %s mix %d horizons %d", got.Kind, len(got.LeverageMix), len(got.Review.Binance.Horizons))
	}
	if got.Binance.ColumnsWithOI != len(got.Times) || len(got.Times) == 0 {
		t.Fatalf("coverage cols %d times %d", got.Binance.ColumnsWithOI, len(got.Times))
	}
	sum := 0.0
	quiet := false
	hotCombined := false
	for i := range got.Times {
		col := 0.0
		for j := range got.Binance.Totals[i] {
			col += got.Binance.Totals[i][j]
		}
		sum += col
		if got.Binance.HasData[i] && col == 0 {
			quiet = true
		}
		if col > 0 && got.Combined.HasData[i] {
			hotCombined = true
		}
	}
	if sum != 2600 || !quiet || hotCombined || got.Binance.MaxIntensity != 2500 {
		t.Fatalf("sum %v quiet %v combinedHot %v max %v", sum, quiet, hotCombined, got.Binance.MaxIntensity)
	}
	missing := map[string]bool{}
	for _, name := range got.MissingVenues {
		missing[name] = true
	}
	if missing["binance"] || !missing["bybit"] || !missing["okx"] {
		t.Fatalf("missing %+v", got.MissingVenues)
	}
	if got.Combined.ColumnsWithOI != 0 {
		t.Fatalf("combined cols %d", got.Combined.ColumnsWithOI)
	}

	one, err := svc.GetLiquidationPrintHeatmap(context.Background(), "binance", "BTCUSDT", "24h")
	if err != nil || one.Binance.MaxIntensity != 2500 || one.Bybit.ColumnsWithOI != 0 {
		t.Fatalf("%v %+v", err, one)
	}
}

func TestGetLiquidationPrintHeatmap_BadRange(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetLiquidationPrintHeatmap(context.Background(), "all", "BTCUSDT", "9h")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetLiquidationPrintHeatmap_EmptyWithoutPrices(t *testing.T) {
	fm := &fakeMarket{tickerErr: domain.ErrNotFound}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
		domain.ExchangeBybit:   fm,
	}, &fakeSupply{})
	got, err := svc.GetLiquidationPrintHeatmap(context.Background(), "all", "BTCUSDT", "24h")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.HuntHeatmapKindPrints || got.PriceMax != 0 || len(got.Binance.Totals) == 0 {
		t.Fatalf("kind %s max %v cols %d", got.Kind, got.PriceMax, len(got.Binance.Totals))
	}
}
