package portfolio

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func flatBooks(price, qty float64) map[string]*domain.RawOrderBook {
	level := []domain.PriceLevel{{Price: price, Quantity: qty}}
	bid := []domain.PriceLevel{{Price: price, Quantity: qty}}
	return map[string]*domain.RawOrderBook{
		"binance":  {Asks: level, Bids: bid},
		"bybit":    {Asks: level, Bids: bid},
		"coinbase": {Asks: level, Bids: bid},
	}
}

func TestFillCost_MarketPaysMoreThanTheQuote(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100", "coinbase|BTC-USD": "100", "bybit|BTCUSDT": "100",
	}}
	books := flatBooks(100, 5)
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: books})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cost", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	quote, err := svc.CompareExecutions(ctx, ExecutionCompareInput{
		ClientID: "cost", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil || quote.ID == "" {
		t.Fatalf("quote %+v err %v", quote, err)
	}
	if payload := ExecutionComparePayload(quote); payload["id"] != quote.ID {
		t.Fatalf("payload id %+v", payload["id"])
	}
	miss, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "cost", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
		ExecutionQuoteID: quote.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if miss.CostCheck != nil {
		t.Fatalf("a different size must not keep the quote: %+v", miss.CostCheck)
	}

	books["binance"] = &domain.RawOrderBook{
		Asks: []domain.PriceLevel{{Price: 110, Quantity: 5}},
		Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}},
	}
	got, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "cost", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		ExecutionQuoteID: quote.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	check := got.CostCheck
	if check == nil {
		t.Fatal("missing cost check")
	}
	if math.Abs(check.ExpectedFee-quote.Rows[0].Fee) > 1e-9 {
		t.Fatalf("expected fee %v quote %v", check.ExpectedFee, quote.Rows[0].Fee)
	}
	if math.Abs(check.ExpectedSlippage) > 1e-9 {
		t.Fatalf("quote slippage %v", check.ExpectedSlippage)
	}
	if math.Abs(check.ActualSlippage-10) > 1e-6 || math.Abs(check.SlippageDelta()-10) > 1e-6 {
		t.Fatalf("slippage %+v", check)
	}
	if check.FeeDelta() <= 0 {
		t.Fatalf("fee should be higher than the quote: %+v", check)
	}
	again, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "cost", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		ExecutionQuoteID: quote.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.CostCheck != nil {
		t.Fatal("one quote belongs to one order")
	}
	list, _, err := svc.ListTrades(ctx, "cost", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	var shown int
	for _, tr := range list {
		if tr.ID == got.ID {
			if tr.CostCheck == nil || math.Abs(tr.CostCheck.SlippageDelta()-10) > 1e-6 {
				t.Fatalf("history %+v", tr.CostCheck)
			}
		}
		if tr.CostCheck != nil {
			shown++
		}
	}
	if shown != 1 {
		t.Fatalf("shown %d", shown)
	}
}

func TestFillCost_SpreadShowsTheDifferenceWhenItFinishes(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100", "coinbase|BTC-USD": "100", "bybit|BTCUSDT": "100",
	}}
	books := flatBooks(100, 5)
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: books})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cost-spread", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	quote, err := svc.CompareExecutions(ctx, ExecutionCompareInput{
		ClientID: "cost-spread", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		SpreadMinutes: 10, SliceCount: 2,
	})
	if err != nil || quote.ID == "" || quote.Rows[2].Status != domain.ExecutionFilled {
		t.Fatalf("quote %+v err %v", quote, err)
	}
	order, _, err := svc.PlaceSpreadOrder(ctx, OrderInput{
		ClientID: "cost-spread", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		SpreadMinutes: 10, SliceCount: 2, ExecutionQuoteID: quote.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := svc.ProcessDueSpreadOrders(ctx, time.Now().UTC()); err != nil || n != 1 {
		t.Fatalf("first n=%d err=%v", n, err)
	}
	list, _, err := svc.ListTrades(ctx, "cost-spread", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].CostCheck != nil {
		t.Fatalf("open spread should not publish the difference yet: %+v", list)
	}
	// The buy lock sits 1% above the quote. 101 still fits. 110 would shrink the last slice.
	books["binance"] = &domain.RawOrderBook{
		Asks: []domain.PriceLevel{{Price: 101, Quantity: 5}},
		Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}},
	}
	books["bybit"] = &domain.RawOrderBook{
		Asks: []domain.PriceLevel{{Price: 102, Quantity: 5}},
		Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}},
	}
	books["coinbase"] = &domain.RawOrderBook{
		Asks: []domain.PriceLevel{{Price: 102, Quantity: 5}},
		Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}},
	}
	open, err := svc.ListSpreadOrders(ctx, "cost-spread", "open", 10, 0)
	if err != nil || len(open) != 1 {
		t.Fatalf("open %+v err %v", open, err)
	}
	if n, err := svc.ProcessDueSpreadOrders(ctx, open[0].NextSliceAt); err != nil || n != 1 {
		t.Fatalf("second n=%d err=%v", n, err)
	}
	done, err := svc.ListSpreadOrders(ctx, "cost-spread", "filled", 10, 0)
	if err != nil || len(done) != 1 || done[0].ID != order.ID {
		t.Fatalf("done %+v err %v", done, err)
	}
	list, _, err = svc.ListTrades(ctx, "cost-spread", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	var check *domain.FillCostCheck
	for i := range list {
		if list[i].CostCheck != nil {
			if check != nil {
				t.Fatal("the difference is shown once")
			}
			check = list[i].CostCheck
		}
	}
	if check == nil {
		t.Fatal("finished spread has no cost check")
	}
	if math.Abs(check.QuotedQuantity-1) > 1e-9 || math.Abs(check.FilledQuantity-1) > 1e-6 {
		t.Fatalf("size %+v", check)
	}
	if math.Abs(check.ExpectedSlippage) > 1e-6 || math.Abs(check.ActualSlippage-0.5) > 1e-4 {
		t.Fatalf("slippage %+v", check)
	}
	if check.FeeDelta() <= 0 {
		t.Fatalf("fee %+v", check)
	}
	if FillCostPayload(check)["slippageDelta"] == nil {
		t.Fatal("payload")
	}
}
