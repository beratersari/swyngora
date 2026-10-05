package market

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func profilePrint(ex domain.Exchange, symbol string, minute time.Time, price, buy, sell float64) domain.FlowAtPricesCell {
	return domain.FlowAtPricesCell{
		Exchange: ex, Symbol: symbol, Minute: minute, Price: price,
		Buy: buy, Sell: sell, BuyN: 1, SellN: 1,
	}
}

func profilePorts(exchanges ...domain.Exchange) map[domain.Exchange]domain.RecentPrintPort {
	out := map[domain.Exchange]domain.RecentPrintPort{}
	for _, ex := range exchanges {
		out[ex] = footprintTape{}
	}
	return out
}

func TestGetVolumeProfile_PerVenueAndCombined(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	from := now.Add(-4 * time.Hour)
	bn := &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"}}
	by := &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"}}
	store := footprintRows{rows: []domain.FlowAtPricesCell{
		profilePrint(domain.ExchangeBinance, "BTCUSDT", from, 64000, 4000, 6000),
		profilePrint(domain.ExchangeBinance, "BTCUSDT", from.Add(time.Hour), 65200, 55000, 25000),
		profilePrint(domain.ExchangeBinance, "BTCUSDT", from.Add(2*time.Hour), 65210, 40000, 30000),
		profilePrint(domain.ExchangeBybit, "BTCUSDT", from, 65200, 20000, 20000),
		profilePrint(domain.ExchangeBybit, "BTCUSDT", from.Add(time.Hour), 67950, 1000, 4000),
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: bn,
		domain.ExchangeBybit:   by,
	}, &fakeSupply{}).
		WithRecentPrints(profilePorts(domain.ExchangeBinance, domain.ExchangeBybit)).
		WithFlowAtPricesHistory(store)
	got, err := svc.GetVolumeProfile(context.Background(), "all", "BTCUSDT", "4h", nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || got.Exchange != "all" || got.Window != "4h" || got.Source != domain.VolumeProfileSourcePrints {
		t.Fatalf("meta %+v", got)
	}
	if !got.Definite || len(got.Missing) != 0 || len(got.Venues) != 2 || got.Combined == nil {
		t.Fatalf("venues %+v missing=%v combined=%+v", got.Venues, got.Missing, got.Combined)
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
	if binance == nil || !binance.BuySellKnown || !binance.Complete || binance.POC.BuyVolume <= 0 {
		t.Fatalf("binance sides %+v", binance)
	}
	if bn.lastQ.Symbol != "" {
		t.Fatalf("profile walked candles %+v", bn.lastQ)
	}
}

func TestGetVolumeProfile_OiAndLiquidationsOnSameRows(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	from := now.Add(-4 * time.Hour)
	cluster := from.Add(time.Hour)
	book := domain.NewLiquidationBook()
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 65220, Quantity: 1, Notional: 80_000, Time: cluster.Add(20 * time.Second),
	})
	store := footprintRows{rows: []domain.FlowAtPricesCell{
		profilePrint(domain.ExchangeBinance, "BTCUSDT", from, 65200, 1000, 1000),
		profilePrint(domain.ExchangeBinance, "BTCUSDT", cluster, 65200, 55000, 25000),
		profilePrint(domain.ExchangeBybit, "BTCUSDT", from, 65200, 20000, 20000),
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"}},
		domain.ExchangeBybit:   &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"}},
	}, &fakeSupply{}).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeBinance: &fakeOI{ser: &domain.OpenInterestSeries{
			Current: domain.OpenInterestPoint{Time: now, Value: 2_000_000, Contracts: 30},
			History: []domain.OpenInterestPoint{
				{Time: cluster, Value: 2_000_000, Contracts: 30},
				{Time: cluster.Add(time.Minute), Value: 2_400_000, Contracts: 36},
			},
		}},
	}).WithLiquidations(book, nil).
		WithRecentPrints(profilePorts(domain.ExchangeBinance, domain.ExchangeBybit)).
		WithFlowAtPricesHistory(store)

	got, err := svc.GetVolumeProfile(context.Background(), "all", "BTCUSDT", "4h", nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	var binance *domain.VolumeProfileVenue
	for i := range got.Venues {
		if got.Venues[i].Exchange == domain.ExchangeBinance {
			binance = &got.Venues[i]
		}
	}
	if binance == nil || !binance.OiKnown || !binance.LiqKnown {
		t.Fatalf("binance activity %+v", binance)
	}
	var row *domain.VolumeProfileBin
	for i := range binance.Bins {
		b := &binance.Bins[i]
		if b.Price <= 65220 && 65220 < b.High && b.Volume > 0 {
			row = b
		}
	}
	if row == nil || row.OiOpened < 100_000 || row.LongLiq <= 0 {
		t.Fatalf("expected OI and long liq on the 65200 row, bins=%+v", binance.Bins)
	}
	if got.Combined == nil || !got.Combined.OiKnown || got.Combined.LongLiq <= 0 {
		t.Fatalf("combined %+v", got.Combined)
	}
}

func TestGetVolumeProfile_SingleVenueAndTick(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	from := now.Add(-time.Hour)
	store := footprintRows{rows: []domain.FlowAtPricesCell{
		profilePrint(domain.ExchangeBinance, "ETHUSDT", from, 65200, 40, 10),
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{ticker: &domain.Ticker24h{Symbol: "ETHUSDT", LastPrice: "65200"}},
		domain.ExchangeBybit:   &fakeMarket{err: errors.New("down")},
	}, &fakeSupply{}).
		WithRecentPrints(profilePorts(domain.ExchangeBinance)).
		WithFlowAtPricesHistory(store)
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
	m := &fakeMarket{ticker: &domain.Ticker24h{LastPrice: "100.5"}}
	store := footprintRows{rows: []domain.FlowAtPricesCell{
		profilePrint(domain.ExchangeBinance, "BTCUSDT", start, 100, 12, 8),
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m, domain.ExchangeBybit: m,
	}, &fakeSupply{}).
		WithRecentPrints(profilePorts(domain.ExchangeBinance)).
		WithFlowAtPricesHistory(store)
	got, err := svc.GetVolumeProfile(context.Background(), "binance", "BTCUSDT", "", &start, &end, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Window != "custom" || !got.From.Equal(start) || !got.To.Equal(end) {
		t.Fatalf("range %+v", got)
	}
	if !got.Definite || got.Venues[0].POC.Price != 100 {
		t.Fatalf("%+v", got.Venues[0])
	}
	if m.lastQ.Symbol != "" {
		t.Fatalf("walked candles %+v", m.lastQ)
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

func TestGetVolumeProfile_LateVenueIsNamedAndCombinedStaysOff(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	from := now.Add(-4 * time.Hour)
	store := footprintRows{rows: []domain.FlowAtPricesCell{
		profilePrint(domain.ExchangeBinance, "BTCUSDT", from, 65200, 40, 10),
		profilePrint(domain.ExchangeBybit, "BTCUSDT", now.Add(-30*time.Minute), 65200, 5, 5),
		profilePrint(domain.ExchangeOKX, "BTCUSDT", from, 65200, 8, 2),
	}}
	svc := New(&fakeMarket{ticker: &domain.Ticker24h{LastPrice: "65500"}}, &fakeSupply{}).
		WithRecentPrints(profilePorts(domain.ExchangeBinance, domain.ExchangeBybit, domain.ExchangeOKX)).
		WithFlowAtPricesHistory(store)
	got, err := svc.GetVolumeProfile(context.Background(), "all", "BTCUSDT", "4h", nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Definite || got.Combined != nil {
		t.Fatalf("combined should stay off: %+v missing=%v", got.Combined, got.Missing)
	}
	if len(got.Missing) != 1 || got.Missing[0] != domain.ExchangeBybit {
		t.Fatalf("missing %v", got.Missing)
	}
	if got.Summary == "" || !strings.Contains(got.Summary, "bybit") {
		t.Fatalf("summary %q", got.Summary)
	}
}

func TestGetVolumeProfile_UsesLastPrice(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	store := footprintRows{rows: []domain.FlowAtPricesCell{
		profilePrint(domain.ExchangeBinance, "BTCUSDT", now.Add(-4*time.Hour), 65200, 80, 20),
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{ticker: &domain.Ticker24h{LastPrice: "69000"}},
	}, &fakeSupply{}).
		WithRecentPrints(profilePorts(domain.ExchangeBinance)).
		WithFlowAtPricesHistory(store)
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
