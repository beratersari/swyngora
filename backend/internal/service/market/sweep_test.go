package market

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetLiquiditySweeps_FindsHighSweep(t *testing.T) {
	t0 := time.Now().UTC().Add(-5 * time.Hour).Truncate(15 * time.Minute)
	seq := [][5]float64{
		{96, 96.2, 95.8, 96, 10}, {96, 97, 95.9, 97, 10}, {97, 98, 96.8, 98, 10},
		{98, 99, 97.8, 99, 10}, {99, 100.5, 98.8, 99.2, 20}, {99.2, 99.3, 98.5, 98.8, 10},
		{98.8, 98.9, 97.4, 97.8, 10}, {97.8, 98.6, 97.5, 98.4, 10}, {98.4, 100.4, 98.2, 99.1, 20},
		{99.1, 99.2, 98.4, 98.7, 10}, {98.7, 98.8, 97.9, 98.2, 10}, {98.2, 99.1, 98.0, 98.9, 10},
		{98.9, 100.8, 98.7, 99.3, 50},
	}
	candles := make([]domain.Candle, len(seq))
	for i, r := range seq {
		at := t0.Add(time.Duration(i) * 15 * time.Minute)
		f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
		candles[i] = domain.Candle{
			OpenTime: at, Open: f(r[0]), High: f(r[1]), Low: f(r[2]), Close: f(r[3]),
			QuoteVolume: f(r[4]), TakerBuyQuote: f(r[4] * 0.4),
		}
	}
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{ticker: &domain.Ticker24h{LastPrice: "99.30"}},
		by:         map[string]map[domain.CandleInterval][]domain.Candle{"BTCUSDT": {domain.Interval15m: candles}},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m, domain.ExchangeBybit: m,
	}, &fakeSupply{})
	got, err := svc.GetLiquiditySweeps(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || len(got.Venues) != 2 {
		t.Fatalf("%+v", got)
	}
	var bn *domain.LiquiditySweepVenue
	for i := range got.Venues {
		if got.Venues[i].Exchange == domain.ExchangeBinance {
			bn = &got.Venues[i]
		}
	}
	if bn == nil || len(bn.Sweeps) != 1 || bn.Sweeps[0].Side != domain.LiquiditySweepSideHigh {
		t.Fatalf("binance %+v", bn)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
	if m.lastQ.Interval != domain.Interval15m || m.lastQ.Limit != domain.SweepCandleLimit {
		t.Fatalf("query %+v", m.lastQ)
	}
}

func TestGetLiquiditySweeps_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetLiquiditySweeps(context.Background(), "all", "  "); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
