package portfolio

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSpreadOrder_SlicesRepickTheCheapestBook(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT":  "100",
		"coinbase|BTC-USD": "100",
		"bybit|BTCUSDT":    "100",
	}}
	books := map[string]*domain.RawOrderBook{
		"binance":  {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 99.7, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 99, Quantity: 5}}},
		"bybit":    {Asks: []domain.PriceLevel{{Price: 101, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 98, Quantity: 5}}},
	}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: books})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "spread", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}

	thin := map[string]*domain.RawOrderBook{
		"binance":  {Asks: []domain.PriceLevel{{Price: 100, Quantity: 0.2}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 0.05}}},
		"bybit":    {Asks: []domain.PriceLevel{{Price: 101, Quantity: 0.05}}},
	}
	svc.SetBookSource(booksByVenue{by: thin})
	whole, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "spread", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil || !whole.BookExhausted || whole.FillsNow {
		t.Fatalf("full size should not fill now: %+v err %v", whole, err)
	}
	preview, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "spread", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1, Spread: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Spread || preview.FillsNow || preview.SliceCount != 10 || math.Abs(preview.SliceQuantity-0.1) > 1e-12 {
		t.Fatalf("preview %+v", preview)
	}
	if preview.BookExhausted || len(preview.RouteLegs) != 1 || preview.RouteLegs[0].Exchange != domain.ExchangeBinance {
		t.Fatalf("first slice %+v exhausted %v", preview.RouteLegs, preview.BookExhausted)
	}
	if preview.ReservedCash <= preview.CashDebit || preview.InsufficientCash {
		t.Fatalf("lock %v debit %v", preview.ReservedCash, preview.CashDebit)
	}
	payload := OrderPreviewPayload(preview)
	if payload["spread"] != true || payload["sliceCount"] != 10 {
		t.Fatalf("payload %+v", payload)
	}
	view, err := svc.View(ctx, "spread")
	if err != nil || math.Abs(view.CashBalance-100000) > 1e-9 {
		t.Fatalf("preview wrote cash %v", view.CashBalance)
	}

	svc.SetBookSource(booksByVenue{by: books})
	order, view, err := svc.PlaceSpreadOrder(ctx, OrderInput{
		ClientID: "spread", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		IdempotencyKey: "spread-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.SpreadStatusOpen || order.FilledQuantity != 0 || order.SlicesDone != 0 {
		t.Fatalf("placed %+v", order)
	}
	if math.Abs(view.CashBalance-100000) > 1e-6 || view.AvailableCash+1e-6 >= 100000 {
		t.Fatalf("cash %v available %v", view.CashBalance, view.AvailableCash)
	}
	again, _, err := svc.PlaceSpreadOrder(ctx, OrderInput{
		ClientID: "spread", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		IdempotencyKey: "spread-1",
	})
	if err != nil || again.ID != order.ID {
		t.Fatalf("replay %+v err %v", again, err)
	}

	n, err := svc.ProcessDueSpreadOrders(ctx, time.Now().UTC())
	if err != nil || n != 1 {
		t.Fatalf("first slice n=%d err=%v", n, err)
	}
	n, err = svc.ProcessDueSpreadOrders(ctx, time.Now().UTC())
	if err != nil || n != 0 {
		t.Fatalf("second pass should wait n=%d err=%v", n, err)
	}
	view, err = svc.View(ctx, "spread")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].Exchange != domain.ExchangeBinance || math.Abs(view.Positions[0].Quantity-0.1) > 1e-9 {
		t.Fatalf("first fill %+v", view.Positions)
	}
	open, err := svc.ListSpreadOrders(ctx, "spread", "open", 10, 0)
	if err != nil || len(open) != 1 || open[0].SlicesDone != 1 {
		t.Fatalf("open %+v err %v", open, err)
	}

	books["coinbase"] = &domain.RawOrderBook{Asks: []domain.PriceLevel{{Price: 99, Quantity: 5}}}
	svc.SetBookSource(booksByVenue{by: books})
	n, err = svc.ProcessDueSpreadOrders(ctx, open[0].NextSliceAt)
	if err != nil || n != 1 {
		t.Fatalf("second slice n=%d err=%v", n, err)
	}
	view, err = svc.View(ctx, "spread")
	if err != nil {
		t.Fatal(err)
	}
	var coinbase float64
	for _, pos := range view.Positions {
		if pos.Exchange == domain.ExchangeCoinbase {
			coinbase = pos.Quantity
		}
	}
	if math.Abs(coinbase-0.1) > 1e-9 {
		t.Fatalf("second slice did not move to coinbase: %+v", view.Positions)
	}

	canceled, err := svc.CancelSpreadOrder(ctx, "spread", order.ID)
	if err != nil || canceled.Status != domain.SpreadStatusCanceled || canceled.CancelReason != domain.CancelReasonUser {
		t.Fatalf("cancel %+v err %v", canceled, err)
	}
	view, err = svc.View(ctx, "spread")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.AvailableCash-view.CashBalance) > 1e-6 {
		t.Fatalf("cancel left cash locked: cash %v available %v", view.CashBalance, view.AvailableCash)
	}
	if len(view.Positions) != 2 {
		t.Fatalf("fills should stay %+v", view.Positions)
	}
}

func TestSpreadSell_RepicksTheBetterBid(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT":  "100",
		"coinbase|BTC-USD": "100",
		"bybit|BTCUSDT":    "100",
	}}
	books := map[string]*domain.RawOrderBook{
		"binance":  {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 102, Quantity: 5}}},
		"bybit":    {Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 90, Quantity: 5}}},
	}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: books})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "spread-sell", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceRoutedOrder(ctx, OrderInput{ClientID: "spread-sell", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	books["binance"] = &domain.RawOrderBook{Asks: []domain.PriceLevel{{Price: 1000, Quantity: 5}}, Bids: books["binance"].Bids}
	books["bybit"] = &domain.RawOrderBook{Asks: []domain.PriceLevel{{Price: 1000, Quantity: 5}}, Bids: books["bybit"].Bids}
	svc.SetBookSource(booksByVenue{by: books})
	if trades, _, err := svc.PlaceRoutedOrder(ctx, OrderInput{ClientID: "spread-sell", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil || len(trades) != 1 || trades[0].Exchange != domain.ExchangeCoinbase {
		t.Fatalf("coinbase inventory %+v err %v", trades, err)
	}
	books["binance"] = &domain.RawOrderBook{Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}}
	books["coinbase"] = &domain.RawOrderBook{Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}}, Bids: []domain.PriceLevel{{Price: 102, Quantity: 5}}}
	svc.SetBookSource(booksByVenue{by: books})
	preview, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "spread-sell", Exchange: "all", Symbol: "BTCUSDT", Side: "sell", Quantity: 1, Spread: true,
	})
	if err != nil || !preview.Spread || preview.ReservedQuantity != 1 || len(preview.RouteLegs) != 1 || preview.RouteLegs[0].Exchange != domain.ExchangeCoinbase {
		t.Fatalf("preview %+v err %v", preview, err)
	}
	order, _, err := svc.PlaceSpreadOrder(ctx, OrderInput{
		ClientID: "spread-sell", Exchange: "all", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(order.Holds) != 2 {
		t.Fatalf("holds %+v", order.Holds)
	}
	if _, err := svc.ProcessDueSpreadOrders(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	view, err := svc.View(ctx, "spread-sell")
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range view.Positions {
		if pos.Exchange == domain.ExchangeCoinbase && math.Abs(pos.Quantity-0.9) > 1e-9 {
			t.Fatalf("coinbase %v", pos.Quantity)
		}
		if pos.Exchange == domain.ExchangeBinance && math.Abs(pos.Quantity-1) > 1e-9 {
			t.Fatalf("binance should still be full %+v", pos)
		}
	}
	books["binance"] = &domain.RawOrderBook{Bids: []domain.PriceLevel{{Price: 110, Quantity: 5}}}
	books["coinbase"] = &domain.RawOrderBook{Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}}
	svc.SetBookSource(booksByVenue{by: books})
	open, err := svc.ListSpreadOrders(ctx, "spread-sell", "", 10, 0)
	if err != nil || len(open) != 1 {
		t.Fatalf("open %+v %v", open, err)
	}
	if _, err := svc.ProcessDueSpreadOrders(ctx, open[0].NextSliceAt); err != nil {
		t.Fatal(err)
	}
	view, err = svc.View(ctx, "spread-sell")
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range view.Positions {
		if pos.Exchange == domain.ExchangeBinance && math.Abs(pos.Quantity-0.9) > 1e-9 {
			t.Fatalf("second slice should sell binance %+v", view.Positions)
		}
	}
}
