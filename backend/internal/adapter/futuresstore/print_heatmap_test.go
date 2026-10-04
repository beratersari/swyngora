package futuresstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAggregateLiquidationPrints_MatchesGoIndex(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	origin := time.Date(2026, 8, 30, 4, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	until := origin.Add(2 * time.Hour)
	at := origin.Add(20 * time.Minute)
	const pMin, pMax = 100.0, 148.0
	nBin := domain.HuntHeatmapPriceBins
	px := 147.6
	ctx := context.Background()
	insert := func(e domain.LiquidationEvent) {
		t.Helper()
		ok, err := s.InsertLiquidation(ctx, e)
		if err != nil || !ok {
			t.Fatalf("insert %+v %v", e, err)
		}
	}
	insert(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: "sell",
		Price: px, Quantity: 1, Notional: 1000, Time: at,
	})
	insert(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: px, Quantity: 2, Notional: 250, Time: at,
	})
	insert(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: px, Quantity: 3, Notional: 999, Time: origin.Add(-time.Millisecond),
	})
	insert(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
		Price: px, Quantity: 1, Notional: 40, Time: until,
	})
	insert(domain.LiquidationEvent{
		Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
		Price: 10, Quantity: 1, Notional: 70, Time: at.Add(step),
	})
	insert(domain.LiquidationEvent{
		Exchange: domain.ExchangeOKX, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 200, Quantity: 1, Notional: 80, Time: at.Add(2 * step),
	})

	q := domain.MeasuredPrintQuery{
		Symbol: "btc-usd", Origin: origin, Until: until, Step: step, Columns: 8,
		PriceMin: pMin, PriceMax: pMax, PriceBins: nBin,
		Exchanges: []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit, domain.ExchangeOKX},
	}
	got, err := s.AggregateLiquidationPrints(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	wantT := domain.MeasuredTimeIndex(at, origin, step)
	wantP := domain.MeasuredPriceIndex(px, pMin, pMax, nBin)
	var longN float64
	var longCount int
	var sawLow, sawHigh bool
	for _, b := range got.Bins {
		switch {
		case b.Exchange == domain.ExchangeBinance && b.Side == domain.LiquidationSideLong:
			if b.TimeIndex != wantT || b.PriceIndex != wantP {
				t.Fatalf("bin %+v want t %d p %d", b, wantT, wantP)
			}
			longN += b.Notional
			longCount += b.Count
		case b.Exchange == domain.ExchangeBybit:
			if b.PriceIndex != nBin-1 || b.Notional != 70 {
				t.Fatalf("low price %+v", b)
			}
			sawLow = true
		case b.Exchange == domain.ExchangeOKX:
			if b.PriceIndex != 0 || b.Notional != 80 {
				t.Fatalf("high price %+v", b)
			}
			sawHigh = true
		default:
			t.Fatalf("unexpected %+v", b)
		}
	}
	if longN != 1250 || longCount != 2 || !sawLow || !sawHigh {
		t.Fatalf("long %v count %d low %v high %v bins %+v", longN, longCount, sawLow, sawHigh, got.Bins)
	}
	if !got.Latest[domain.ExchangeBinance].Equal(at) {
		t.Fatalf("latest %v", got.Latest)
	}
	if _, err := s.AggregateLiquidationPrints(ctx, domain.MeasuredPrintQuery{}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err %v", err)
	}
	lo, hi, err := s.LiquidationPrintPriceSpan(ctx, "BTCUSDT", origin, until, []domain.Exchange{domain.ExchangeBinance})
	if err != nil || lo != px || hi != px {
		t.Fatalf("span %v %v %v", lo, hi, err)
	}
}

func TestAggregateLiquidationPrints_OpenTailStaysOnLastColumn(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	origin := time.Date(2026, 8, 30, 4, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	const columns = 8
	closed := origin.Add(time.Duration(columns) * step)
	tail := closed.Add(time.Minute)
	until := closed.Add(5 * time.Minute)
	const pMin, pMax = 100.0, 148.0
	ctx := context.Background()
	if _, err := s.InsertLiquidation(ctx, domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 120, Quantity: 1, Notional: 80, Time: tail,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertLiquidation(ctx, domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 120, Quantity: 1, Notional: 9, Time: until.Add(step),
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.AggregateLiquidationPrints(ctx, domain.MeasuredPrintQuery{
		Symbol: "BTCUSDT", Origin: origin, Until: until, Step: step, Columns: columns,
		PriceMin: pMin, PriceMax: pMax, PriceBins: domain.HuntHeatmapPriceBins,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Bins) != 1 || got.Bins[0].TimeIndex != columns-1 || got.Bins[0].Notional != 80 {
		t.Fatalf("%+v", got.Bins)
	}
}
