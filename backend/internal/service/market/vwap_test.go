package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetVWAP_WeightsVolumeAndCombines(t *testing.T) {
	start := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	bn := &fakeMarket{
		candles: []domain.Candle{
			{OpenTime: start, High: "100", Low: "100", Close: "100", QuoteVolume: "1000000"},
			{OpenTime: start.Add(time.Hour), High: "110", Low: "110", Close: "110", QuoteVolume: "9000000"},
		},
		ticker: &domain.Ticker24h{LastPrice: "112"},
	}
	by := &fakeMarket{
		candles: []domain.Candle{
			{OpenTime: start, High: "109", Low: "109", Close: "109", QuoteVolume: "5000000"},
		},
		ticker: &domain.Ticker24h{LastPrice: "112"},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: bn, domain.ExchangeBybit: by,
	}, &fakeSupply{})
	got, err := svc.GetVWAP(context.Background(), "all", "BTCUSDT", "", &start, &end)
	if err != nil {
		t.Fatal(err)
	}
	if got.Window != "custom" || !got.From.Equal(start) {
		t.Fatalf("range %+v", got)
	}
	var binance *domain.VWAPVenue
	for i := range got.Venues {
		if got.Venues[i].Exchange == domain.ExchangeBinance {
			binance = &got.Venues[i]
		}
	}
	if binance == nil || binance.VWAP < 108.9 || binance.VWAP > 109.1 {
		t.Fatalf("binance %+v", binance)
	}
	if got.Combined == nil || got.Combined.Volume != 15_000_000 {
		t.Fatalf("combined %+v", got.Combined)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestGetVWAP_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetVWAP(context.Background(), "all", "  ", "24h", nil, nil); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
