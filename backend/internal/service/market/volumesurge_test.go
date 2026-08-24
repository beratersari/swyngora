package market

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func surgeCandles(n int, lastVol float64) []domain.Candle {
	t0 := time.Now().UTC().Add(-time.Duration(n) * 5 * time.Minute).Truncate(5 * time.Minute)
	out := make([]domain.Candle, n)
	for i := 0; i < n; i++ {
		vol := 2_000_000.0
		buy := 1_000_000.0
		if i == n-1 {
			vol = lastVol
			buy = lastVol * 0.8
		}
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
		out[i] = domain.Candle{
			OpenTime: at, Open: "1", High: "1", Low: "1", Close: "1",
			QuoteVolume: f(vol), TakerBuyQuote: f(buy),
		}
	}
	return out
}

func TestGetVolumeSurge_FiveXBuySpike(t *testing.T) {
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{},
		by: map[string]map[domain.CandleInterval][]domain.Candle{
			"BTCUSDT": {domain.Interval5m: surgeCandles(24, 10_000_000)},
		},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m, domain.ExchangeBybit: m,
	}, &fakeSupply{})
	got, err := svc.GetVolumeSurge(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	v := got.Venues[0]
	if v.MaxRatio < 4.9 || v.Hottest != domain.VolumeSurgeWindow5m {
		t.Fatalf("venue %+v", v)
	}
	var w5 domain.VolumeSurgeWindow
	for _, w := range v.Windows {
		if w.Window == domain.VolumeSurgeWindow5m {
			w5 = w
		}
	}
	if w5.Dominant != domain.TakerSideBuy {
		t.Fatalf("dominant %+v", w5)
	}
	if m.lastQ.Interval != domain.Interval5m || m.lastQ.Limit != domain.VolumeSurgeLookbackBars {
		t.Fatalf("query %+v", m.lastQ)
	}
}

func TestScanVolumeSurges_RanksHotCoins(t *testing.T) {
	spot := []domain.SpotMarket{
		{Symbol: "HOTUSDT", BaseAsset: "HOT", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "9000"},
		{Symbol: "QUIETUSDT", BaseAsset: "QUIET", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "8000"},
	}
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{spot: spot},
		by: map[string]map[domain.CandleInterval][]domain.Candle{
			"HOTUSDT":   {domain.Interval5m: surgeCandles(20, 10_000_000)},
			"QUIETUSDT": {domain.Interval5m: surgeCandles(20, 2_000_000)},
		},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.ScanVolumeSurges(context.Background(), "binance", "USDT", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) != 1 || got.Hits[0].Symbol != "HOTUSDT" {
		t.Fatalf("hits %+v", got.Hits)
	}
	if got.Hits[0].MaxRatio < 4.9 {
		t.Fatalf("ratio %+v", got.Hits[0])
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestGetVolumeSurge_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetVolumeSurge(context.Background(), "all", "  "); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
