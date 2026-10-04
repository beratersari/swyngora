package futureshist

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/futuresstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAggregateLiquidationPrints_DelegatesToSQLite(t *testing.T) {
	st, err := futuresstore.Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	origin := time.Date(2026, 8, 30, 4, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	at := origin.Add(20 * time.Minute)
	_, err = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
		Price: 147.6, Quantity: 1, Notional: 800, Time: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: st}
	q := domain.MeasuredPrintQuery{
		Symbol: "BTCUSDT", Origin: origin, Until: origin.Add(time.Hour), Step: step, Columns: 4,
		PriceMin: 100, PriceMax: 148, PriceBins: domain.HuntHeatmapPriceBins,
		Exchanges: []domain.Exchange{domain.ExchangeBinance},
	}
	got, err := svc.AggregateLiquidationPrints(context.Background(), q)
	if err != nil || len(got.Bins) != 1 || got.Bins[0].Notional != 800 || got.Bins[0].Side != domain.LiquidationSideShort {
		t.Fatalf("%+v %v", got, err)
	}
	if got.Bins[0].TimeIndex != domain.MeasuredTimeIndex(at, origin, step) {
		t.Fatalf("tbin %d", got.Bins[0].TimeIndex)
	}
}

func TestAggregateLiquidationPrints_FallsBackToList(t *testing.T) {
	origin := time.Date(2026, 8, 30, 4, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	st := &memStore{}
	_, _ = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 120, Quantity: 1, Notional: 50, Time: origin.Add(-time.Millisecond),
	})
	_, _ = st.InsertLiquidation(context.Background(), domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 120, Quantity: 1, Notional: 75, Time: origin.Add(step),
	})
	svc := &Service{Store: st}
	q := domain.MeasuredPrintQuery{
		Symbol: "BTCUSDT", Origin: origin, Until: origin.Add(time.Hour), Step: step, Columns: 4,
		PriceMin: 100, PriceMax: 148, PriceBins: domain.HuntHeatmapPriceBins,
		Exchanges: []domain.Exchange{domain.ExchangeBinance},
	}
	got, err := svc.AggregateLiquidationPrints(context.Background(), q)
	if err != nil || len(got.Bins) != 1 || got.Bins[0].Notional != 75 {
		t.Fatalf("%+v %v", got, err)
	}
	lo, hi, err := svc.LiquidationPrintPriceSpan(context.Background(), "BTCUSDT", origin, origin.Add(time.Hour), q.Exchanges)
	if err != nil || lo != 0 || hi != 0 {
		t.Fatalf("span %v %v %v", lo, hi, err)
	}
}
