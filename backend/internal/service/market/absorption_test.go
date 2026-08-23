package market

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetAbsorption_PerVenueAndCombined(t *testing.T) {
	t0 := time.Now().UTC().Add(-55 * time.Minute).Truncate(5 * time.Minute)
	var bn, by []domain.TakerBucket
	var candles []domain.Candle
	px := 70000.0
	for i := 0; i < 12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		bn = append(bn, domain.TakerBucket{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: at, BuyNotional: 20, SellNotional: 180})
		by = append(by, domain.TakerBucket{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Start: at, BuyNotional: 10, SellNotional: 90})
		candles = append(candles, domain.Candle{OpenTime: at, Close: strconv.FormatFloat(px, 'f', 2, 64), QuoteVolume: "200", TakerBuyQuote: "30"})
	}
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{},
		by:         map[string]map[domain.CandleInterval][]domain.Candle{"BTCUSDT": {"5m": candles}},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m, domain.ExchangeBybit: m,
	}, &fakeSupply{}).WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: &fakeTaker{buckets: bn, flow: &domain.TakerVenueFlow{}},
		domain.ExchangeBybit:   &fakeTaker{buckets: by, flow: &domain.TakerVenueFlow{}},
	})
	got, err := svc.GetAbsorption(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 || got.Combined == nil || len(got.Combined.Points) == 0 {
		t.Fatalf("%+v", got)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
	var binance *domain.AbsorptionVenue
	for i := range got.Venues {
		if got.Venues[i].Exchange == domain.ExchangeBinance {
			binance = &got.Venues[i]
		}
	}
	if binance == nil || len(binance.Windows) != 4 {
		t.Fatalf("binance %+v", binance)
	}
	var w1 domain.AbsorptionWindowStat
	for _, w := range binance.Windows {
		if w.Window == domain.CVDWindow1h {
			w1 = w
		}
	}
	if w1.Kind != domain.AbsorptionKindBid || w1.SellNotional < w1.BuyNotional {
		t.Fatalf("1h %+v", w1)
	}
	if len(got.SpotVenues) == 0 {
		t.Fatal("spot venues")
	}
}

func TestGetAbsorption_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetAbsorption(context.Background(), "all", "  "); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
