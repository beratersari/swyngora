package futuresstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSumAndBucketLiquidations_UsesStoredPrints(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	origin := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	inside := origin.Add(90 * time.Minute)
	outside := origin.Add(5 * time.Hour)
	for _, ev := range []domain.LiquidationEvent{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Price: 100, Quantity: 1, Notional: 80, Time: inside},
		{Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT", Side: domain.LiquidationSideShort, Price: 3, Quantity: 2, Notional: 20, Time: inside.Add(time.Minute)},
		{Exchange: domain.ExchangeOKX, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Price: 100, Quantity: 1, Notional: 999, Time: outside},
	} {
		if _, err := s.InsertLiquidation(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}
	from, to := origin, origin.Add(4*time.Hour)
	sums, err := s.SumLiquidationSymbols(context.Background(), from, to, domain.FuturesVenues())
	if err != nil {
		t.Fatal(err)
	}
	if len(sums.Rows) != 2 || sums.Biggest == nil || sums.Biggest.Notional != 80 {
		t.Fatalf("sums %+v biggest %+v", sums.Rows, sums.Biggest)
	}
	if sums.Latest[domain.ExchangeOKX].IsZero() == false {
		t.Fatal("okx print is outside the window")
	}
	if !sums.Latest[domain.ExchangeBinance].Equal(inside) {
		t.Fatalf("latest %s", sums.Latest[domain.ExchangeBinance])
	}
	step := time.Hour
	bars, err := s.BucketLiquidations(context.Background(), from, to, step, []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit})
	if err != nil {
		t.Fatal(err)
	}
	if len(bars.Buckets) != 1 || bars.Buckets[0].Long != 80 || bars.Buckets[0].Short != 20 {
		t.Fatalf("%+v", bars.Buckets)
	}
	sides, err := s.SumLiquidationPrintSides(context.Background(), "BTCUSDT", from, to, domain.FuturesVenues())
	if err != nil {
		t.Fatal(err)
	}
	if len(sides.Rows) != 1 || sides.Rows[0].Exchange != domain.ExchangeBinance || sides.Rows[0].Long != 80 || sides.Rows[0].Short != 0 {
		t.Fatalf("btc sides %+v", sides.Rows)
	}
	if !sides.Latest[domain.ExchangeBinance].Equal(inside) || !sides.Latest[domain.ExchangeOKX].IsZero() {
		t.Fatalf("latest %+v", sides.Latest)
	}
	atEnd, err := s.SumLiquidationPrintSides(context.Background(), "btcusdt", to, to.Add(time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(atEnd.Rows) != 0 {
		t.Fatalf("window end is exclusive: %+v", atEnd.Rows)
	}
	if !bars.Buckets[0].Start.Equal(origin.Add(time.Hour)) {
		t.Fatalf("bar start %s", bars.Buckets[0].Start)
	}
}
