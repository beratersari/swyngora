package portfolio

import (
	"context"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSituationRecord_CountsClosedSlicesAndSkipsABlankBuy(t *testing.T) {
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
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-rec", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "120"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "90"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	svc.situations = nil
	px.prices["binance|BTCUSDT"] = "90"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.SituationRecord(ctx, "sit-rec")
	if err != nil {
		t.Fatal(err)
	}
	if got.Counted != 2 || got.Skipped != 1 || len(got.Rows) != 2 {
		t.Fatalf("record counted=%d skipped=%d rows=%+v", got.Counted, got.Skipped, got.Rows)
	}
	for _, row := range got.Rows {
		if row.Count != 2 || row.Profitable != 1 || math.Abs(row.RealizedPnL-10) > 1e-9 {
			t.Fatalf("row %+v", row)
		}
	}
	if got.Rows[0].Clause != "priceCvd" || got.Rows[1].Clause != "funding" {
		t.Fatalf("order %+v", got.Rows)
	}
	for _, row := range got.Rows {
		if len(row.Closes) != 2 {
			t.Fatalf("closes %+v", row.Closes)
		}
		if row.Closes[0].Symbol != "BTCUSDT" || row.Closes[0].Exchange != domain.ExchangeBinance || row.Closes[0].ClosedAt.IsZero() {
			t.Fatalf("newest %+v", row.Closes[0])
		}
		if row.Closes[1].Symbol != "BTCUSDT" || !row.Closes[0].ClosedAt.After(row.Closes[1].ClosedAt) {
			t.Fatalf("order %+v", row.Closes)
		}
		if math.Abs(row.Closes[0].RealizedPnL-(-10)) > 1e-9 || math.Abs(row.Closes[1].RealizedPnL-20) > 1e-9 {
			t.Fatalf("pnl %+v", row.Closes)
		}
	}
	if !got.Rows[0].Closes[0].ClosedAt.Equal(got.Rows[1].Closes[0].ClosedAt) || got.Rows[0].Closes[1].RealizedPnL != got.Rows[1].Closes[1].RealizedPnL {
		t.Fatalf("same sells %+v %+v", got.Rows[0].Closes, got.Rows[1].Closes)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-rec-eq", StartingBalance: 10000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["nasdaq|AAPL"] = "130"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	eq, err := svc.SituationRecord(ctx, "sit-rec-eq")
	if err != nil {
		t.Fatal(err)
	}
	if eq.Counted != 0 || eq.Skipped != 0 || len(eq.Rows) != 0 {
		t.Fatalf("equity %+v", eq)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-rec-open", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	svc.SetSituationSource(src)
	px.prices["binance|BTCUSDT"] = "100"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-rec-open", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	open, err := svc.SituationRecord(ctx, "sit-rec-open")
	if err != nil {
		t.Fatal(err)
	}
	if open.Counted != 0 || open.Skipped != 0 || len(open.Rows) != 0 {
		t.Fatalf("open %+v", open)
	}
}
