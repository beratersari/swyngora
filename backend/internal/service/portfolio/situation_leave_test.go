package portfolio

import (
	"context"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSituationLeave_SplitsAChangedKindAndKeepsABlankSellOut(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"nasdaq|AAPL":     "100",
	}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		Funding:  &domain.SituationFunding{Payer: "long"},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-leave", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "120"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown},
		Funding:  &domain.SituationFunding{Payer: "long"},
	}
	px.prices["binance|BTCUSDT"] = "90"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}

	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
	}
	px.prices["binance|BTCUSDT"] = "100"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	svc.situations = nil
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}

	px.prices["binance|BTCUSDT"] = "100"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.SituationLeave(ctx, "sit-leave")
	if err != nil {
		t.Fatal(err)
	}
	if got.Counted != 2 || got.Skipped != 1 || got.Unreadable != 1 {
		t.Fatalf("counted=%d skipped=%d unreadable=%d rows=%+v", got.Counted, got.Skipped, got.Unreadable, got.Rows)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("rows %+v", got.Rows)
	}
	price := got.Rows[0]
	if price.Clause != "priceCvd" || price.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("price %+v", price)
	}
	if price.Still.Count != 1 || price.Still.Profitable != 1 || math.Abs(price.Still.RealizedPnL-20) > 1e-9 {
		t.Fatalf("still %+v", price.Still)
	}
	if price.Left.Count != 1 || price.Left.Profitable != 0 || math.Abs(price.Left.RealizedPnL-(-10)) > 1e-9 {
		t.Fatalf("left %+v", price.Left)
	}
	if price.Still.Closes[0].Symbol != "BTCUSDT" || price.Left.Closes[0].Symbol != "BTCUSDT" {
		t.Fatalf("closes %+v %+v", price.Still.Closes, price.Left.Closes)
	}
	if !price.Left.Closes[0].ClosedAt.After(price.Still.Closes[0].ClosedAt) {
		t.Fatalf("times still=%s left=%s", price.Still.Closes[0].ClosedAt, price.Left.Closes[0].ClosedAt)
	}
	funding := got.Rows[1]
	if funding.Clause != "funding" || funding.Still.Count != 2 || funding.Left.Count != 0 || math.Abs(funding.Still.RealizedPnL-10) > 1e-9 {
		t.Fatalf("funding %+v", funding)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-leave-eq", StartingBalance: 10000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["nasdaq|AAPL"] = "130"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	eq, err := svc.SituationLeave(ctx, "sit-leave-eq")
	if err != nil {
		t.Fatal(err)
	}
	if eq.Counted != 0 || eq.Skipped != 0 || eq.Unreadable != 0 || len(eq.Rows) != 0 {
		t.Fatalf("equity %+v", eq)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-leave-open", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	svc.SetSituationSource(src)
	px.prices["binance|BTCUSDT"] = "100"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-leave-open", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	open, err := svc.SituationLeave(ctx, "sit-leave-open")
	if err != nil {
		t.Fatal(err)
	}
	if open.Counted != 0 || open.Skipped != 0 || open.Unreadable != 0 || len(open.Rows) != 0 {
		t.Fatalf("open %+v", open)
	}
}
