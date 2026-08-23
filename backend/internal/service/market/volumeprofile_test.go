package market

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func profileCandles(now time.Time) []domain.Candle {
	// Cluster at 65200, thin volume at 64000 and 68000.
	rows := []struct {
		low, high, close, quote, buy string
	}{
		{"64000", "64100", "64050", "10000", "4000"},
		{"65150", "65250", "65200", "80000", "55000"},
		{"65180", "65240", "65210", "70000", "40000"},
		{"67900", "68000", "67950", "12000", "3000"},
	}
	out := make([]domain.Candle, len(rows))
	for i, r := range rows {
		out[i] = domain.Candle{
			OpenTime: now.Add(-time.Duration(len(rows)-i) * time.Hour),
			Low:      r.low, High: r.high, Close: r.close,
			QuoteVolume: r.quote, TakerBuyQuote: r.buy,
		}
	}
	return out
}

func TestGetVolumeProfile_PerVenueAndCombined(t *testing.T) {
	now := time.Now().UTC()
	candles := profileCandles(now)
	bn := &fakeMarket{
		candles: candles,
		ticker:  &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"},
	}
	by := &fakeMarket{
		candles: []domain.Candle{
			{OpenTime: now.Add(-2 * time.Hour), Low: "65150", High: "65250", Close: "65200", QuoteVolume: "40000"},
			{OpenTime: now.Add(-time.Hour), Low: "67900", High: "68000", Close: "67950", QuoteVolume: "5000"},
		},
		ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: bn,
		domain.ExchangeBybit:   by,
	}, &fakeSupply{})
	got, err := svc.GetVolumeProfile(context.Background(), "all", "BTCUSDT", "4h", nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || got.Exchange != "all" || got.Window != "4h" {
		t.Fatalf("meta %+v", got)
	}
	if len(got.Venues) != 2 || got.Combined == nil {
		t.Fatalf("venues %+v combined=%+v", got.Venues, got.Combined)
	}
	if got.Combined.POC.Price < 65100 || got.Combined.POC.Price > 65300 {
		t.Fatalf("combined poc %+v", got.Combined.POC)
	}
	if got.Combined.ValueArea.Low == 0 || got.Combined.ValueArea.High == 0 {
		t.Fatalf("va %+v", got.Combined.ValueArea)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
	var binance *domain.VolumeProfileVenue
	for i := range got.Venues {
		if got.Venues[i].Exchange == domain.ExchangeBinance {
			binance = &got.Venues[i]
		}
	}
	if binance == nil || !binance.BuySellKnown || binance.POC.BuyVolume <= 0 {
		t.Fatalf("binance sides %+v", binance)
	}
	if bn.lastQ.Interval != domain.Interval1m || bn.lastQ.StartTime.IsZero() {
		t.Fatalf("expected ranged 1m fetch %+v", bn.lastQ)
	}
}

func TestGetVolumeProfile_SingleVenueAndTick(t *testing.T) {
	now := time.Now().UTC()
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{
			candles: profileCandles(now),
			ticker:  &domain.Ticker24h{Symbol: "ETHUSDT", LastPrice: "65200"},
		},
		domain.ExchangeBybit: &fakeMarket{err: errors.New("down")},
	}, &fakeSupply{})
	got, err := svc.GetVolumeProfile(context.Background(), "binance", "ethusdt", "1h", nil, nil, 50)
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != "binance" || got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Venues[0].TickSize != 50 {
		t.Fatalf("tick %v", got.Venues[0].TickSize)
	}
}

func TestGetVolumeProfile_CustomRange(t *testing.T) {
	start := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 19, 14, 0, 0, 0, time.UTC)
	m := &fakeMarket{
		candles: []domain.Candle{{
			OpenTime: start.Add(time.Hour), Low: "100", High: "101", Close: "100.5",
			QuoteVolume: "20", TakerBuyQuote: "12",
		}},
		ticker: &domain.Ticker24h{LastPrice: "100.5"},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m, domain.ExchangeBybit: m,
	}, &fakeSupply{})
	got, err := svc.GetVolumeProfile(context.Background(), "binance", "BTCUSDT", "", &start, &end, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Window != "custom" || !got.From.Equal(start) || !got.To.Equal(end) {
		t.Fatalf("range %+v", got)
	}
	if !m.lastQ.StartTime.Equal(start) || !m.lastQ.EndTime.Equal(end) {
		t.Fatalf("query %+v", m.lastQ)
	}
}

func TestGetVolumeProfile_BadInput(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetVolumeProfile(context.Background(), "all", "  ", "24h", nil, nil, 0); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("symbol %v", err)
	}
	if _, err := svc.GetVolumeProfile(context.Background(), "coinbase", "BTCUSDT", "24h", nil, nil, 0); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("exchange %v", err)
	}
	if _, err := svc.GetVolumeProfile(context.Background(), "all", "BTCUSDT", "2h", nil, nil, 0); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("window %v", err)
	}
	if _, err := svc.GetVolumeProfile(context.Background(), "all", "BTCUSDT", "24h", nil, nil, -1); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("tick %v", err)
	}
}

func TestFilterCandlesRange(t *testing.T) {
	from := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)
	in := []domain.Candle{
		{OpenTime: from.Add(-time.Hour), QuoteVolume: "1"},
		{OpenTime: from.Add(30 * time.Minute), QuoteVolume: "2"},
		{OpenTime: to, QuoteVolume: "3"},
	}
	got := filterCandlesRange(in, from, to, domain.Interval1m)
	if len(got) != 1 || got[0].QuoteVolume != "2" {
		t.Fatalf("%+v", got)
	}
}

func TestGetVolumeProfile_PagesStop(t *testing.T) {
	// Ensure we do not loop forever when the adapter keeps returning the same page.
	start := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	m := &pagingMarket{batch: []domain.Candle{{
		OpenTime: start, Low: "10", High: "11", Close: "10.5", QuoteVolume: "5",
	}}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m,
	}, &fakeSupply{})
	got, err := svc.GetVolumeProfile(context.Background(), "binance", "BTCUSDT", "", &start, &end, 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.calls > domain.MaxVolumeProfilePages {
		t.Fatalf("calls %d", m.calls)
	}
	if got.Venues[0].BarCount != 1 {
		t.Fatalf("bars %+v", got.Venues[0])
	}
}

type pagingMarket struct {
	fakeMarket
	batch []domain.Candle
	calls int
}

func (p *pagingMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	p.calls++
	p.lastQ = q
	return p.batch, nil
}

func TestGetVolumeProfile_UsesLastPrice(t *testing.T) {
	now := time.Now().UTC()
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{
			candles: profileCandles(now),
			ticker:  &domain.Ticker24h{LastPrice: "69000"},
		},
	}, &fakeSupply{})
	got, err := svc.GetVolumeProfile(context.Background(), "binance", "BTCUSDT", "4h", nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].LastPrice != 69000 {
		t.Fatalf("last %v", got.Venues[0].LastPrice)
	}
	if got.Venues[0].LastVsArea != domain.VolumeProfileVsAbove {
		t.Fatalf("vs %s", got.Venues[0].LastVsArea)
	}
	_ = strconv.Itoa(got.Venues[0].BarCount)
}
