package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type closeFailStore struct {
	domain.PortfolioPort
}

func (closeFailStore) ListSituationCloses(context.Context, string) ([]domain.SituationClose, error) {
	return nil, errors.New("closes down")
}

func TestPreviewOrder_ReasonHistoryUsesTheSelectedLiveKind(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		Funding:  &domain.SituationFunding{Payer: "long"},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-hist", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-hist", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "120"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-hist", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
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
		ClientID: "reason-hist", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}

	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		Funding:  &domain.SituationFunding{Payer: "long"},
	}
	px.prices["binance|BTCUSDT"] = "100"
	priceOnly, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-hist", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if priceOnly.SituationStatus != domain.SituationPreviewReady || len(priceOnly.ReasonHistory) != 1 {
		t.Fatalf("status %s rows %+v", priceOnly.SituationStatus, priceOnly.ReasonHistory)
	}
	row := priceOnly.ReasonHistory[0]
	if row.Clause != domain.SituationClausePriceCVD || row.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("price row %+v", row)
	}
	if row.Still.Count != 1 || row.Still.Profitable != 1 || math.Abs(row.Still.RealizedPnL-20) > 1e-9 {
		t.Fatalf("still %+v", row.Still)
	}
	if row.Left.Count != 1 || row.Left.Profitable != 0 || math.Abs(row.Left.RealizedPnL-(-10)) > 1e-9 {
		t.Fatalf("left %+v", row.Left)
	}
	if row.Still.Closes[0].Symbol != "BTCUSDT" || row.Left.Closes[0].Symbol != "BTCUSDT" {
		t.Fatalf("closes %+v %+v", row.Still.Closes, row.Left.Closes)
	}

	both, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-hist", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: domain.SituationReason{Set: true, Clauses: []string{
			domain.SituationClauseFunding, domain.SituationClausePriceCVD,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(both.ReasonHistory) != 2 {
		t.Fatalf("two rows %+v", both.ReasonHistory)
	}
	if both.ReasonHistory[0].Clause != domain.SituationClausePriceCVD || both.ReasonHistory[1].Clause != domain.SituationClauseFunding {
		t.Fatalf("order %+v", both.ReasonHistory)
	}
	if both.ReasonHistory[1].Still.Count != 2 || both.ReasonHistory[1].Left.Count != 0 || math.Abs(both.ReasonHistory[1].Still.RealizedPnL-10) > 1e-9 {
		t.Fatalf("funding %+v", both.ReasonHistory[1])
	}

	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
	}
	none, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-hist", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(none.ReasonHistory) != 1 || none.ReasonHistory[0].Kind != domain.SituationCVDAgrees ||
		none.ReasonHistory[0].Still.Count != 0 || none.ReasonHistory[0].Left.Count != 0 {
		t.Fatalf("no past close %+v", none.ReasonHistory)
	}
}

func TestPreviewOrder_ReasonHistoryStaysOffThePrice(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"nasdaq|AAPL":     "100",
	}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: fullSituation()}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-quiet", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "110"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "100"

	unset, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unset.SituationStatus != domain.SituationPreviewReady || unset.ReasonSet || len(unset.ReasonHistory) != 0 {
		t.Fatalf("unset %+v %v", unset.SituationStatus, unset.ReasonHistory)
	}

	empty, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: domain.SituationReason{Set: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !empty.ReasonSet || len(empty.Reason) != 0 || len(empty.ReasonHistory) != 0 || empty.FillPrice <= 0 {
		t.Fatalf("empty reason %+v price %v", empty.Reason, empty.FillPrice)
	}

	sell, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "sell", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if sell.SituationStatus != domain.SituationPreviewSkipped || len(sell.ReasonHistory) != 0 {
		t.Fatalf("sell %+v", sell.SituationStatus)
	}

	src.err = domain.ErrUpstream
	failed, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed.SituationStatus != domain.SituationPreviewFailed || len(failed.ReasonHistory) != 0 || failed.FillPrice <= 0 {
		t.Fatalf("failed %+v price %v", failed.SituationStatus, failed.FillPrice)
	}

	src.err = nil
	svc.store = closeFailStore{svc.store}
	priced, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-quiet", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if priced.SituationStatus != domain.SituationPreviewReady || !priced.ReasonSet || len(priced.ReasonHistory) != 0 || priced.FillPrice <= 0 {
		t.Fatalf("history miss still prices %+v %v", priced.SituationStatus, priced.FillPrice)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-eq", StartingBalance: 10000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	equity, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "buy", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if equity.SituationStatus != domain.SituationPreviewSkipped || len(equity.ReasonHistory) != 0 || equity.FillPrice <= 0 {
		t.Fatalf("equity %+v", equity.SituationStatus)
	}
}
