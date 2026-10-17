package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type booksByVenue struct {
	by  map[string]*domain.RawOrderBook
	err map[string]error
}

func (b booksByVenue) GetRawOrderBook(_ context.Context, exchange, _ string) (*domain.RawOrderBook, error) {
	if err := b.err[exchange]; err != nil {
		return nil, err
	}
	book := b.by[exchange]
	if book == nil {
		return nil, domain.ErrNotFound
	}
	cp := *book
	return &cp, nil
}

func TestPreviewAndPlaceRoute_FeePicksBinanceOverCheaperCoinbase(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT":  "100",
		"coinbase|BTC-USD": "99.60",
		"bybit|BTCUSDT":    "100",
	}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance":  {Asks: []domain.PriceLevel{{Price: 100, Quantity: 2}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 99.60, Quantity: 2}}},
		"bybit":    {Asks: []domain.PriceLevel{{Price: 101, Quantity: 2}}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "route-fee", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "route-fee", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		CompareExchange: "coinbase",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Routed || !got.FillsNow || len(got.RouteLegs) != 1 || got.RouteLegs[0].Exchange != domain.ExchangeBinance {
		t.Fatalf("preview %+v legs %+v", got, got.RouteLegs)
	}
	if math.Abs(got.CashDebit-100.1) > 1e-6 {
		t.Fatalf("debit %v", got.CashDebit)
	}
	if got.RouteCompare == nil || !got.RouteCompare.Known || got.RouteCompare.SavedCash <= 0 {
		t.Fatalf("compare %+v", got.RouteCompare)
	}
	payload := OrderPreviewPayload(got)
	legs, _ := payload["routeLegs"].([]any)
	if payload["routed"] != true || len(legs) != 1 {
		t.Fatalf("payload routed=%v legs=%d", payload["routed"], len(legs))
	}
	view, err := svc.View(ctx, "route-fee")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-10000) > 1e-9 {
		t.Fatalf("preview wrote cash %v", view.CashBalance)
	}
	trades, view, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-fee", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		IdempotencyKey: "route-fee-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 || trades[0].Exchange != domain.ExchangeBinance || trades[0].Symbol != "BTCUSDT" {
		t.Fatalf("trades %+v", trades)
	}
	if math.Abs(view.CashBalance-(10000-100.1)) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("cash %v positions %+v", view.CashBalance, view.Positions)
	}
	again, _, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-fee", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		IdempotencyKey: "route-fee-1",
	})
	if err != nil || len(again) != 1 || again[0].ID != trades[0].ID {
		t.Fatalf("replay %+v err %v", again, err)
	}
	view, err = svc.View(ctx, "route-fee")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-1) > 1e-9 {
		t.Fatalf("replay doubled the fill: %+v", view.Positions)
	}
}

func TestPlaceRoute_SplitsAThinLevelAcrossTwoVenues(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "bybit|BTCUSDT": "100"}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 0.4}, {Price: 101, Quantity: 2}}},
		"bybit":   {Asks: []domain.PriceLevel{{Price: 100.05, Quantity: 2}}},
	}, err: map[string]error{"coinbase": errors.New("timeout")}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "route-split", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "route-split", Exchange: "best", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		CompareExchange: "binance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.RouteLegs) != 2 || got.RouteMissing == nil || got.RouteMissing[0] != "coinbase" {
		t.Fatalf("legs %+v missing %v", got.RouteLegs, got.RouteMissing)
	}
	if got.RouteCompare == nil || got.RouteCompare.SavedCash <= 0.5 {
		t.Fatalf("saved %+v", got.RouteCompare)
	}
	trades, view, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-split", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 2 || len(view.Positions) != 2 {
		t.Fatalf("trades %d positions %+v", len(trades), view.Positions)
	}
	var binanceQty, bybitQty float64
	for _, pos := range view.Positions {
		switch pos.Exchange {
		case domain.ExchangeBinance:
			binanceQty = pos.Quantity
		case domain.ExchangeBybit:
			bybitQty = pos.Quantity
		}
	}
	if math.Abs(binanceQty-0.4) > 1e-9 || math.Abs(bybitQty-0.6) > 1e-9 {
		t.Fatalf("qty binance %v bybit %v", binanceQty, bybitQty)
	}
}

func TestPlaceRoute_CoinbaseFillIsStoredOnTheUSDPair(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "coinbase|BTC-USD": "90"}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance":  {Asks: []domain.PriceLevel{{Price: 100, Quantity: 2}}, Bids: []domain.PriceLevel{{Price: 100, Quantity: 2}}},
		"coinbase": {Asks: []domain.PriceLevel{{Price: 90, Quantity: 2}}, Bids: []domain.PriceLevel{{Price: 90, Quantity: 2}}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "route-cb", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	trades, view, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-cb", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 || trades[0].Exchange != domain.ExchangeCoinbase || trades[0].Symbol != "BTC-USD" {
		t.Fatalf("buy %+v", trades)
	}
	if len(view.Positions) != 1 || view.Positions[0].Symbol != "BTC-USD" {
		t.Fatalf("position %+v", view.Positions)
	}
	sold, view, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-cb", Exchange: "all", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sold) != 1 || sold[0].Exchange != domain.ExchangeCoinbase || sold[0].Symbol != "BTC-USD" {
		t.Fatalf("sell %+v", sold)
	}
	if len(view.Positions) != 0 {
		t.Fatalf("still open %+v", view.Positions)
	}
}

func TestPlaceRoute_RejectsAShortBookAndAShortSell(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "bybit|BTCUSDT": "100"}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 0.2}}, Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}}},
		"bybit":   {Asks: []domain.PriceLevel{{Price: 100, Quantity: 0.2}}, Bids: []domain.PriceLevel{{Price: 99, Quantity: 5}}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "route-short", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-short", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); !errors.Is(err, domain.ErrBookExhausted) {
		t.Fatalf("buy %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "route-short", Symbol: "BTCUSDT", Side: "buy", Quantity: 0.2}); err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "route-short", Exchange: "all", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.InsufficientQuantity || preview.FillsNow {
		t.Fatalf("sell preview %+v", preview)
	}
	if _, _, err := svc.PlaceRoutedOrder(ctx, OrderInput{
		ClientID: "route-short", Exchange: "all", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err == nil || !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("sell %v", err)
	}
}

func TestPreviewRoute_RejectsMarginAndEquities(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	svc.SetBookSource(booksByVenue{by: map[string]*domain.RawOrderBook{
		"binance": {Asks: []domain.PriceLevel{{Price: 100, Quantity: 1}}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "route-no", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "route-no", Exchange: "all", Symbol: "BTCUSDT", Product: "margin", Side: "long", Quantity: 1, Leverage: 2,
	}); err == nil {
		t.Fatal("margin should be rejected")
	}
	if _, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "route-no", Exchange: "all", Symbol: "AAPL", Side: "buy", Quantity: 1,
	}); err == nil {
		t.Fatal("equity should be rejected")
	}
	if _, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "route-no", Exchange: "all", Symbol: "BTCUSDT", Side: "buy", Type: "limit_buy", Quantity: 1, TriggerPrice: 90,
	}); err == nil {
		t.Fatal("limit should be rejected")
	}
}
