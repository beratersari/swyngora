package market

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type situationAtArchive struct {
	oi      []domain.FuturesSnapshot
	funding []domain.FuturesSnapshot
	crowd   []domain.FuturesSnapshot
	events  []domain.LiquidationEvent
	err     error
}

func (a *situationAtArchive) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	if a.err != nil {
		return nil, a.err
	}
	switch q.Metric {
	case domain.FuturesMetricOpenInterest:
		return a.oi, nil
	case domain.FuturesMetricFunding:
		return a.funding, nil
	case domain.FuturesMetricLongShort:
		return a.crowd, nil
	case "liquidations":
		if a.events == nil {
			return []domain.LiquidationEvent{}, nil
		}
		return a.events, nil
	default:
		return nil, nil
	}
}

type situationAtBuckets struct {
	rows    []domain.TakerBucket
	upserts int
}

func (s *situationAtBuckets) ListTakerBuckets(_ context.Context, exchange, symbol string, from, to time.Time) ([]domain.TakerBucket, error) {
	var out []domain.TakerBucket
	for _, row := range s.rows {
		if string(row.Exchange) != exchange {
			continue
		}
		if symbol != "" && row.Symbol != "" && row.Symbol != symbol {
			continue
		}
		if row.Start.Before(from) || row.Start.After(to) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (s *situationAtBuckets) UpsertTakerBuckets(_ context.Context, recs []domain.TakerBucket) (int, error) {
	s.upserts += len(recs)
	return len(recs), nil
}

func (s *situationAtBuckets) PurgeTakerBuckets(context.Context, time.Time) (int, error) {
	return 0, nil
}

func TestGetCoinSituationAt_RejectsAFutureClock(t *testing.T) {
	svc := New(nil, nil)
	_, err := svc.GetCoinSituationAt(context.Background(), "BTCUSDT", time.Time{})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("zero at: %v", err)
	}
	_, err = svc.GetCoinSituationAt(context.Background(), "BTCUSDT", time.Now().UTC().Add(2*time.Minute))
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("future at: %v", err)
	}
	got, err := svc.GetCoinSituationAt(context.Background(), "BTCUSDT", time.Now().UTC().Add(30*time.Second))
	if err != nil || got == nil {
		t.Fatalf("slack at err=%v got=%v", err, got)
	}
}

func TestGetCoinSituationAt_NamesEveryClauseWhenTheArchiveIsMissing(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	got, err := New(nil, nil).GetCoinSituationAt(context.Background(), "BTCUSDT", at)
	if err != nil || got == nil {
		t.Fatalf("err=%v got=%v", err, got)
	}
	if got.PriceCVD != nil || len(got.PriceCVDMissing) != 3 {
		t.Fatalf("price %+v missing %v", got.PriceCVD, got.PriceCVDMissing)
	}
	if got.Size != nil || len(got.SizeMissing) != 3 {
		t.Fatalf("size %+v missing %v", got.Size, got.SizeMissing)
	}
	if got.OpenInterest != nil || len(got.OpenInterestMissing) != 3 {
		t.Fatalf("oi %+v missing %v", got.OpenInterest, got.OpenInterestMissing)
	}
	if got.Funding != nil || len(got.FundingMissing) != 3 {
		t.Fatalf("funding %+v missing %v", got.Funding, got.FundingMissing)
	}
	if got.Crowd != nil || len(got.CrowdMissing) != 2 {
		t.Fatalf("crowd %+v missing %v", got.Crowd, got.CrowdMissing)
	}
	for _, name := range got.CrowdMissing {
		if name == string(domain.ExchangeBybit) {
			t.Fatalf("bybit is not a crowd venue: %v", got.CrowdMissing)
		}
	}
	if got.Prints != nil || len(got.PrintsMissing) != 3 {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
	if got.Liquidation != nil || len(got.LiquidationMissing) != 0 {
		t.Fatalf("pocket %+v missing %v", got.Liquidation, got.LiquidationMissing)
	}
	if got.Note == "" || !strings.Contains(got.Note, "not part of this read") || strings.Contains(got.Note, "The model names the closer estimated pocket") {
		t.Fatalf("note %q", got.Note)
	}
}

func TestGetCoinSituationAt_HistoryErrorIsNotQuiet(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	book := domain.NewLiquidationBook()
	start := at.Add(-48 * time.Hour)
	for _, ex := range domain.FuturesVenues() {
		book.RestoreTracking(ex, "", start, 48*time.Hour)
		book.RestoreTracking(ex, "BTCUSDT", start, 48*time.Hour)
	}
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100, Quantity: 1, Notional: 500, Time: at.Add(-time.Minute),
	})
	svc := New(nil, nil).WithLiquidations(book, nil)
	svc.SetFuturesHistory(&situationAtArchive{err: errors.New("db")})
	got, err := svc.GetCoinSituationAt(context.Background(), "BTCUSDT", at)
	if err != nil || got == nil {
		t.Fatalf("err=%v got=%v", err, got)
	}
	if got.Prints != nil || len(got.PrintsMissing) != 3 {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
}

func TestGetCoinSituationAt_ReadsTheStoredMinute(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	var rows []domain.TakerBucket
	for _, ex := range domain.FuturesVenues() {
		rows = append(rows, situationAtTape(at, ex)...)
	}
	store := &situationAtBuckets{rows: rows}
	mkt := &fakeMarket{candles: []domain.Candle{
		{OpenTime: at.Add(-24 * time.Hour), Close: "100"},
		{OpenTime: at.Add(time.Minute), Close: "999"},
	}}
	book := domain.NewLiquidationBook()
	start := at.Add(-48 * time.Hour)
	for _, ex := range domain.FuturesVenues() {
		book.RestoreTracking(ex, "", start, 48*time.Hour)
		book.RestoreTracking(ex, "BTCUSDT", start, 48*time.Hour)
	}
	var events []domain.LiquidationEvent
	for _, ex := range domain.FuturesVenues() {
		events = append(events, domain.LiquidationEvent{
			Exchange: ex, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
			Price: 100, Quantity: 1, Notional: 1000, Time: at.Add(-5 * time.Minute),
		})
	}
	events = append(events, domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
		Price: 100, Quantity: 1, Notional: 5_000_000, Time: at,
	})
	var oi, funding []domain.FuturesSnapshot
	for _, ex := range domain.FuturesVenues() {
		oi = append(oi,
			domain.FuturesSnapshot{Exchange: ex, SampledAt: at.Add(-time.Hour), Contracts: 100, Value: 1_000_000},
			domain.FuturesSnapshot{Exchange: ex, SampledAt: at, Contracts: 110, Value: 1_000_000},
			domain.FuturesSnapshot{Exchange: ex, SampledAt: at.Add(time.Minute), Contracts: 1, Value: 1},
		)
		funding = append(funding, domain.FuturesSnapshot{
			Exchange: ex, SampledAt: at, Predicted: true, FundingRate: 0.0001,
		})
	}
	crowd := []domain.FuturesSnapshot{
		{Exchange: domain.ExchangeBinance, SampledAt: at, LongShare: 0.40, ShortShare: 0.60, SizeLongShare: 0.60},
		{Exchange: domain.ExchangeOKX, SampledAt: at, LongShare: 0.40, ShortShare: 0.60, SizeLongShare: 0.60},
	}
	svc := New(mkt, &fakeSupply{}).WithLiquidations(book, nil).WithTakerBucketStore(store)
	svc.SetFuturesHistory(&situationAtArchive{oi: oi, funding: funding, crowd: crowd, events: events})
	svc.SetOnFuturesSymbol(func(string) { t.Fatal("past read subscribed the coin") })

	got, err := svc.GetCoinSituationAt(context.Background(), "BTCUSDT", at)
	if err != nil || got == nil {
		t.Fatalf("err=%v got=%v", err, got)
	}
	if store.upserts != 0 {
		t.Fatalf("upserts %d", store.upserts)
	}
	if !mkt.lastQ.EndTime.Equal(at) || !mkt.lastQ.StartTime.Equal(at.Add(-24*time.Hour)) || mkt.lastQ.Limit != 320 {
		t.Fatalf("candles %+v", mkt.lastQ)
	}
	if got.PriceCVD == nil || got.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("price %+v missing %v", got.PriceCVD, got.PriceCVDMissing)
	}
	if got.Size == nil || got.Size.Kind != domain.SituationSizeWhaleBuyRetailSell {
		t.Fatalf("size %+v missing %v", got.Size, got.SizeMissing)
	}
	if got.OpenInterest == nil || got.OpenInterest.Flow != domain.SituationOIOpened {
		t.Fatalf("oi %+v missing %v", got.OpenInterest, got.OpenInterestMissing)
	}
	if got.Funding == nil || got.Funding.Payer != "long" || got.Funding.RatePct != 0.01 {
		t.Fatalf("funding %+v missing %v", got.Funding, got.FundingMissing)
	}
	if got.Crowd == nil || got.Crowd.Lean != domain.LongShortGapLeanTopLong || got.Crowd.GapPct != 20 {
		t.Fatalf("crowd %+v missing %v", got.Crowd, got.CrowdMissing)
	}
	if got.Prints == nil || got.Prints.Side != domain.SituationPrintLong || got.Prints.LongNotional != 3000 || got.Prints.ShortNotional != 0 {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
	if got.Liquidation != nil || len(got.LiquidationMissing) != 0 {
		t.Fatalf("pocket %+v missing %v", got.Liquidation, got.LiquidationMissing)
	}
	if !strings.Contains(got.Note, "not part of this read") {
		t.Fatalf("note %q", got.Note)
	}
}

func situationAtTape(at time.Time, ex domain.Exchange) []domain.TakerBucket {
	var out []domain.TakerBucket
	for ts := at.Add(-24 * time.Hour); !ts.After(at.Add(-5 * time.Minute)); ts = ts.Add(5 * time.Minute) {
		row := domain.TakerBucket{
			Exchange: ex, Symbol: "BTCUSDT", Start: ts, BuyNotional: 1,
		}
		if ts.Equal(at.Add(-5 * time.Minute)) {
			row.BuyNotional = 200_000
			row.SellNotional = 50
			row.WhaleBuy = 200_000
			row.RetailSell = 50
			row.SizeKnown = true
		}
		out = append(out, row)
	}
	out = append(out, domain.TakerBucket{
		Exchange: ex, Symbol: "BTCUSDT", Start: at,
		BuyNotional: 50, SellNotional: 200_000,
		WhaleSell: 200_000, RetailBuy: 50, SizeKnown: true,
	})
	return out
}
