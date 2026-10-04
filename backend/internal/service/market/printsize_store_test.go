package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type emptyTape struct {
	rested bool
}

func (emptyTape) GetRecentPrints(context.Context, string) ([]domain.TakerPrint, error) {
	return nil, nil
}

func (e *emptyTape) ListPrints(context.Context, string, time.Time, time.Time) ([]domain.TakerPrint, error) {
	e.rested = true
	return []domain.TakerPrint{{
		Side: domain.TakerSideBuy, Notional: 1, Time: time.Now().UTC(),
	}}, nil
}

func (emptyTape) MemoryPrints(string, time.Time, time.Time) ([]domain.TakerPrint, bool) {
	return nil, false
}

type storedFlow struct {
	cells []domain.FlowAtPricesCell
}

func (storedFlow) AddFlowCells(context.Context, []domain.FlowAtPricesCell) error { return nil }
func (s storedFlow) ListFlowCells(context.Context, domain.Exchange, string, time.Time, time.Time) ([]domain.FlowAtPricesCell, error) {
	return s.cells, nil
}
func (storedFlow) FlowCursor(context.Context, domain.Exchange, string) (time.Time, error) {
	return time.Time{}, nil
}
func (storedFlow) SetFlowCursor(context.Context, domain.Exchange, string, time.Time) error {
	return nil
}
func (storedFlow) PurgeFlowCells(context.Context, time.Time) (int, error) { return 0, nil }

func TestGetPrintSize_UsesStoredFlowWhenTapeIsEmpty(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	tape := &emptyTape{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{}).
		WithRecentPrints(map[domain.Exchange]domain.RecentPrintPort{
			domain.ExchangeBinance: tape,
		}).
		WithFlowAtPricesHistory(storedFlow{cells: []domain.FlowAtPricesCell{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Minute: now.Add(-20 * time.Minute), Buy: 40_000, BuyN: 2},
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Minute: now.Add(-2 * time.Minute), Buy: 360_000, BuyN: 2},
		}})
	got, err := svc.GetPrintSize(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if tape.rested {
		t.Fatal("public trades were read while stored flow covered the window")
	}
	var w domain.PrintSizeWindow
	for _, x := range got.Venues[0].Windows {
		if x.Window == domain.PrintSizeWindow15m {
			w = x
		}
	}
	if w.Buy.Count != 2 || w.Buy.PriorCount != 2 || w.Buy.Kind != domain.PrintSizeKindJumped {
		t.Fatalf("%+v", w.Buy)
	}
}

func TestGetOrderCount_UsesStoredFlowWhenTapeIsEmpty(t *testing.T) {
	now := time.Now().UTC()
	tape := &emptyTape{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{}).
		WithRecentPrints(map[domain.Exchange]domain.RecentPrintPort{
			domain.ExchangeBinance: tape,
		}).
		WithFlowAtPricesHistory(storedFlow{cells: []domain.FlowAtPricesCell{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Minute: now.Add(-15 * time.Second), Buy: 10, BuyN: 8, Sell: 2, SellN: 2},
		}})
	got, err := svc.GetOrderCount(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if tape.rested {
		t.Fatal("public trades were read while stored flow covered the window")
	}
	var w domain.OrderCountWindow
	for _, x := range got.Venues[0].Windows {
		if x.Window == domain.OrderCount1m {
			w = x
		}
	}
	if w.BuyCount != 8 || w.SellCount != 2 || !w.Complete {
		t.Fatalf("%+v", w)
	}
}
