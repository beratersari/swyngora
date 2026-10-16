package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type situationPrintArchive struct {
	scan    domain.LiquidationPrintSideScan
	err     error
	history any
}

func (a situationPrintArchive) History(context.Context, domain.FuturesHistoryQuery) (any, error) {
	return a.history, nil
}

func (situationPrintArchive) SumLiquidationSymbols(context.Context, time.Time, time.Time, []domain.Exchange) (domain.LiquidationSymbolScan, error) {
	return domain.LiquidationSymbolScan{}, nil
}

func (situationPrintArchive) BucketLiquidations(context.Context, time.Time, time.Time, time.Duration, []domain.Exchange) (domain.LiquidationBucketScan, error) {
	return domain.LiquidationBucketScan{}, nil
}

func (a situationPrintArchive) SumLiquidationPrintSides(context.Context, string, time.Time, time.Time, []domain.Exchange) (domain.LiquidationPrintSideScan, error) {
	return a.scan, a.err
}

func TestGetCoinSituation_MeasuredPrintsUseStoredRowsAndNewerBook(t *testing.T) {
	now := time.Now().UTC()
	storedAt := now.Add(-time.Hour)
	fresh := now.Add(-time.Minute)
	book := domain.NewLiquidationBook()
	start := now.Add(-48 * time.Hour)
	for _, ex := range domain.FuturesVenues() {
		book.RestoreTracking(ex, "", start, 48*time.Hour)
		book.RestoreTracking(ex, "BTCUSDT", start, 48*time.Hour)
	}
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100, Quantity: 1, Notional: 100, Time: storedAt,
	})
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100, Quantity: 1, Notional: 25, Time: fresh,
	})
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	svc.SetFuturesHistory(situationPrintArchive{
		scan: domain.LiquidationPrintSideScan{
			Rows: []domain.LiquidationPrintSideRow{
				{Exchange: domain.ExchangeBinance, Long: 100},
				{Exchange: domain.ExchangeBybit, Short: 40},
				{Exchange: domain.ExchangeOKX, Long: 10},
			},
			Latest: map[domain.Exchange]time.Time{domain.ExchangeBinance: storedAt},
		},
	})
	got, err := svc.GetCoinSituation(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prints == nil || got.Prints.Side != domain.SituationPrintLong || got.Prints.Window != "24h" {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
	if got.Prints.LongNotional != 135 || got.Prints.ShortNotional != 40 {
		t.Fatalf("notionals %+v", got.Prints)
	}
	if len(got.PrintAnchors) != 2 || got.PrintAnchors[0].Anchor != domain.PrintAnchorDay || got.PrintAnchors[1].Anchor != domain.PrintAnchor4h {
		t.Fatalf("anchors %+v", got.PrintAnchors)
	}
	for _, row := range got.PrintAnchors {
		if !row.Complete || row.Side != domain.SituationPrintLong || row.LongNotional != 135 || row.ShortNotional != 40 {
			t.Fatalf("anchor %+v", row)
		}
	}
	if got.Liquidation != nil && got.Liquidation.Side == got.Prints.Side && got.Prints.LongNotional == 0 {
		t.Fatal("measured notionals must stay on the print clause")
	}
}

func TestGetCoinSituation_MeasuredPrintsStayBlankWithoutTheArchive(t *testing.T) {
	now := time.Now().UTC()
	book := domain.NewLiquidationBook()
	start := now.Add(-48 * time.Hour)
	for _, ex := range domain.FuturesVenues() {
		book.RestoreTracking(ex, "BTCUSDT", start, 48*time.Hour)
	}
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100, Quantity: 1, Notional: 500, Time: now.Add(-time.Minute),
	})
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	svc.SetFuturesHistory(situationPrintArchive{err: errors.New("db")})
	got, err := svc.GetCoinSituation(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prints != nil || len(got.PrintsMissing) != len(domain.FuturesVenues()) {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
	if len(got.PrintAnchors) != 2 {
		t.Fatalf("anchors %+v", got.PrintAnchors)
	}
	for _, row := range got.PrintAnchors {
		if row.Complete || row.Side != "" || len(row.Missing) != len(domain.FuturesVenues()) {
			t.Fatalf("blank anchor %+v", row)
		}
	}
}

func TestGetCoinSituation_PrintClockNamesWhenTheSideChanged(t *testing.T) {
	now := time.Now().UTC()
	book := domain.NewLiquidationBook()
	start := now.Add(-72 * time.Hour)
	for _, ex := range domain.FuturesVenues() {
		book.RestoreTracking(ex, "", start, 72*time.Hour)
		book.RestoreTracking(ex, "BTCUSDT", start, 72*time.Hour)
	}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	svc.SetFuturesHistory(situationPrintArchive{
		scan: domain.LiquidationPrintSideScan{
			Rows: []domain.LiquidationPrintSideRow{
				{Exchange: domain.ExchangeBinance, Long: 200},
			},
			Latest: map[domain.Exchange]time.Time{
				domain.ExchangeBinance: now.Add(-2 * time.Hour),
			},
		},
		history: []domain.LiquidationEvent{
			{
				Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
				Notional: 100, Time: now.Add(-30 * time.Hour),
			},
			{
				Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
				Notional: 200, Time: now.Add(-2 * time.Hour),
			},
		},
	})
	got, err := svc.GetCoinSituation(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prints == nil || got.Prints.Side != domain.SituationPrintLong {
		t.Fatalf("prints %+v", got.Prints)
	}
	if got.Prints.Since.IsZero() || got.Prints.Since.After(now) {
		t.Fatalf("since %s", got.Prints.Since)
	}
}
