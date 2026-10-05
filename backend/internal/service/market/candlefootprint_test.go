package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type footprintRows struct {
	rows   []domain.FlowAtPricesCell
	rested *bool
}

func (footprintRows) AddFlowCells(context.Context, []domain.FlowAtPricesCell) error { return nil }
func (f footprintRows) ListFlowCells(_ context.Context, ex domain.Exchange, _ string, _, _ time.Time) ([]domain.FlowAtPricesCell, error) {
	out := make([]domain.FlowAtPricesCell, 0, len(f.rows))
	for _, row := range f.rows {
		if row.Exchange == "" || row.Exchange == ex {
			out = append(out, row)
		}
	}
	return out, nil
}
func (footprintRows) FlowCursor(context.Context, domain.Exchange, string) (time.Time, error) {
	return time.Time{}, nil
}
func (footprintRows) SetFlowCursor(context.Context, domain.Exchange, string, time.Time) error {
	return nil
}
func (footprintRows) PurgeFlowCells(context.Context, time.Time) (int, error) { return 0, nil }

type footprintTape struct {
	prints  []domain.TakerPrint
	covered bool
	rested  *bool
}

func (footprintTape) GetRecentPrints(context.Context, string) ([]domain.TakerPrint, error) {
	return nil, nil
}
func (t footprintTape) ListPrints(context.Context, string, time.Time, time.Time) ([]domain.TakerPrint, error) {
	if t.rested != nil {
		*t.rested = true
	}
	return []domain.TakerPrint{{Price: 1, Notional: 1, Time: time.Now().UTC()}}, nil
}
func (t footprintTape) MemoryPrints(string, time.Time, time.Time) ([]domain.TakerPrint, bool) {
	return t.prints, t.covered
}

func TestGetCandleFootprint_UsesStoredMinutesWithoutREST(t *testing.T) {
	open := time.Now().UTC().Truncate(time.Hour)
	rested := false
	store := footprintRows{rows: []domain.FlowAtPricesCell{{
		Exchange: domain.ExchangeBinance,
		Symbol:   "BTCUSDT",
		Minute:   open.Add(5 * time.Minute),
		Price:    100,
		Buy:      8,
		Sell:     2,
		BuyN:     2,
		SellN:    1,
	}}, rested: &rested}
	svc := New(&fakeMarket{}, &fakeSupply{}).
		WithRecentPrints(map[domain.Exchange]domain.RecentPrintPort{
			domain.ExchangeBinance: footprintTape{rested: &rested},
		}).
		WithFlowAtPricesHistory(store)
	start := open
	end := open.Add(time.Hour)
	got, err := svc.GetCandleFootprint(context.Background(), "binance", "BTCUSDT", "1h", &start, &end)
	if err != nil {
		t.Fatal(err)
	}
	if rested {
		t.Fatal("candle footprint walked REST")
	}
	if !got.Definite || len(got.Venues) != 1 || len(got.Venues[0].Candles) != 1 {
		t.Fatalf("%+v", got)
	}
	candle := got.Venues[0].Candles[0]
	if candle.Buy != 8 || candle.Sell != 2 || candle.Rows[0].Imbalance != domain.TakerSideBuy {
		t.Fatalf("%+v", candle)
	}
}

func TestGetCandleFootprint_OpenMinuteReplacesStoredSlice(t *testing.T) {
	now := time.Now().UTC()
	open := now.Truncate(time.Minute)
	hour := now.Truncate(time.Hour)
	rested := false
	store := footprintRows{rows: []domain.FlowAtPricesCell{{
		Exchange: domain.ExchangeBinance,
		Symbol:   "BTCUSDT",
		Minute:   open,
		Price:    100,
		Buy:      50,
		Sell:     50,
	}}}
	tape := footprintTape{
		covered: true,
		rested:  &rested,
		prints: []domain.TakerPrint{{
			Side: domain.TakerSideBuy, Price: 100, Notional: 6, Time: now.Add(-time.Second),
		}},
	}
	svc := New(&fakeMarket{}, &fakeSupply{}).
		WithRecentPrints(map[domain.Exchange]domain.RecentPrintPort{
			domain.ExchangeBinance: tape,
		}).
		WithFlowAtPricesHistory(store)
	start := hour
	end := now
	got, err := svc.GetCandleFootprint(context.Background(), "binance", "BTCUSDT", "1h", &start, &end)
	if err != nil {
		t.Fatal(err)
	}
	if rested {
		t.Fatal("candle footprint walked REST")
	}
	if len(got.Venues) != 1 || len(got.Venues[0].Candles) != 1 || got.Venues[0].Candles[0].Buy != 6 {
		t.Fatalf("%+v", got.Venues)
	}
}

func TestGetCandleFootprint_CombinedStaysEmptyWhenAVenueIsMissing(t *testing.T) {
	open := time.Now().UTC().Truncate(time.Hour)
	store := footprintRows{rows: []domain.FlowAtPricesCell{{
		Exchange: domain.ExchangeBinance,
		Symbol:   "BTCUSDT",
		Minute:   open.Add(time.Minute),
		Price:    100,
		Buy:      3,
		Sell:     1,
	}}}
	ports := map[domain.Exchange]domain.RecentPrintPort{
		domain.ExchangeBinance: footprintTape{},
		domain.ExchangeBybit:   footprintTape{},
		domain.ExchangeOKX:     footprintTape{},
	}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithRecentPrints(ports).WithFlowAtPricesHistory(store)
	start := open
	end := open.Add(time.Hour)
	got, err := svc.GetCandleFootprint(context.Background(), "all", "BTCUSDT", "1h", &start, &end)
	if err != nil {
		t.Fatal(err)
	}
	if got.Definite || got.Combined != nil {
		t.Fatalf("combined should stay empty: %+v", got)
	}
	if len(got.Missing) != 2 || len(got.Venues) != 1 || got.Venues[0].Exchange != domain.ExchangeBinance {
		t.Fatalf("%+v missing %v", got.Venues, got.Missing)
	}
}
