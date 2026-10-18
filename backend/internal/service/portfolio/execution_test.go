package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestCompareExecutions_PricesThreePathsAndWritesNothing(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT":  "100",
		"coinbase|BTC-USD": "100",
		"bybit|BTCUSDT":    "102",
	}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 1}, {Price: 110, Quantity: 5}},
			Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
		"bybit": {Asks: []domain.PriceLevel{{Price: 102, Quantity: 5}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 5}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 5}}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "exec-cmp", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.CompareExecutions(ctx, ExecutionCompareInput{
		ClientID: "exec-cmp", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
		SpreadMinutes: 10, SliceCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != 3 || got.Rows[0].Status != domain.ExecutionFilled || got.Rows[2].Status != domain.ExecutionFilled {
		t.Fatalf("%+v", got.Rows)
	}
	if got.Rows[2].SavedVsMarket == nil || *got.Rows[2].SavedVsMarket <= 0 {
		t.Fatalf("spread saved %+v", got.Rows[2].SavedVsMarket)
	}
	payload := ExecutionComparePayload(got)
	rows, _ := payload["rows"].([]any)
	if len(rows) != 3 || payload["touchPrice"] != 100.0 {
		t.Fatalf("payload %+v", payload)
	}
	view, err := svc.View(ctx, "exec-cmp")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-10000) > 1e-9 || len(view.Positions) != 0 {
		t.Fatalf("wrote the book %+v", view)
	}
}

func TestCompareExecutions_RejectsAnEquityVenue(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"nasdaq|AAPL": "100"}}).WithPaperCosts(domain.TradingCostFor)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "exec-eq", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.CompareExecutions(ctx, ExecutionCompareInput{
		ClientID: "exec-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "buy", Quantity: 1,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err %v", err)
	}
}

func TestCompareExecutions_KeepsTheOtherPricesWhenSlicesAreTooClose(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100", "coinbase|BTC-USD": "100", "bybit|BTCUSDT": "100",
	}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance":  {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
		"bybit":    {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "exec-gap", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.CompareExecutions(ctx, ExecutionCompareInput{
		ClientID: "exec-gap", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		SpreadMinutes: 1, SliceCount: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows[0].Status != domain.ExecutionFilled || got.Rows[1].Status != domain.ExecutionFilled {
		t.Fatalf("rows %+v", got.Rows)
	}
	if got.Rows[2].Status != domain.ExecutionSchedule || got.Rows[2].Schedule != domain.ExecutionScheduleInterval {
		t.Fatalf("spread %+v", got.Rows[2])
	}
}
