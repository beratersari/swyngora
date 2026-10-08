package market

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/futuresstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/futureshist"
)

func TestLiquidationDesk_UsesStoredPrintsOnce(t *testing.T) {
	store, err := futuresstore.Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	old := now.Add(-72 * time.Hour)
	recent := now.Add(-30 * time.Minute)
	fresh := now.Add(-time.Second)
	for _, ev := range []domain.LiquidationEvent{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Price: 100, Quantity: 1, Notional: 70, Time: old},
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Price: 100, Quantity: 1, Notional: 80, Time: recent},
	} {
		if _, err := store.InsertLiquidation(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}
	book := domain.NewLiquidationBook()
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100, Quantity: 1, Notional: 80, Time: recent,
	})
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT", Side: domain.LiquidationSideShort,
		Price: 3, Quantity: 1, Notional: 15, Time: fresh,
	})
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	svc.SetFuturesHistory(&futureshist.Service{Store: store})

	levels, err := svc.GetLiquidationLevels(context.Background(), "all", "all", "7d")
	if err != nil {
		t.Fatal(err)
	}
	if levels.Kind != domain.LiquidationLevelsKindTotal || levels.From.After(old) {
		t.Fatalf("from %s kind %s", levels.From, levels.Kind)
	}
	var longN, shortN float64
	for _, bar := range levels.Bars {
		if v, e := strconv.ParseFloat(bar.LongNotional, 64); e == nil {
			longN += v
		}
		if v, e := strconv.ParseFloat(bar.ShortNotional, 64); e == nil {
			shortN += v
		}
	}
	if longN != 150 || shortN != 15 {
		t.Fatalf("long=%v short=%v", longN, shortN)
	}

	overview, err := svc.GetLiquidationOverview(context.Background(), "all", "1h", 10)
	if err != nil {
		t.Fatal(err)
	}
	var hour domain.LiquidationWindowTotals
	for _, w := range overview.Windows {
		if w.Window == "1h" {
			hour = w
		}
	}
	hourLong, _ := strconv.ParseFloat(hour.LongNotional, 64)
	hourShort, _ := strconv.ParseFloat(hour.ShortNotional, 64)
	if hourLong != 80 || hourShort != 15 || hour.Count != 2 {
		t.Fatalf("hour %+v", hour)
	}
	if len(overview.Coins) != 2 || overview.Coins[0].Symbol != "BTCUSDT" {
		t.Fatalf("coins %+v", overview.Coins)
	}
	day, err := svc.GetLiquidationOverview(context.Background(), "all", "24h", 10)
	if err != nil {
		t.Fatal(err)
	}
	var dayLong float64
	for _, w := range day.Windows {
		if w.Window == "24h" {
			dayLong, _ = strconv.ParseFloat(w.LongNotional, 64)
		}
	}
	if dayLong != 80 {
		t.Fatalf("24h long %v includes the 3-day print", dayLong)
	}
}
