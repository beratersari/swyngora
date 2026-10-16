package portfolio

import (
	"context"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func reasonOnlyPrice() domain.SituationReason {
	return domain.SituationReason{Set: true, Clauses: []string{domain.SituationClausePriceCVD}}
}

func fullSituation() *domain.CoinSituation {
	return &domain.CoinSituation{
		Symbol:       "BTCUSDT",
		PriceCVD:     &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		OpenInterest: &domain.SituationOpenInterest{Flow: domain.SituationOIOpened, Window: "1h"},
	}
}

func TestPlaceOrder_StoresOnlyTheNamedClauses(t *testing.T) {
	svc := newSvc(t, nil)
	src := &fakeSituation{got: fullSituation()}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-buy", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-buy", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		Reason: reasonOnlyPrice(),
	}); err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListTrades(ctx, "reason-buy", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation == nil || list[0].Situation.PriceCVD == nil || list[0].Situation.OpenInterest != nil {
		t.Fatalf("buy kept %+v", list[0].Situation)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-sell", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-sell", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-sell", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
		Reason: reasonOnlyPrice(),
	}); err != nil {
		t.Fatal(err)
	}
	sells, _, err := svc.ListTrades(ctx, "reason-sell", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sell *domain.Trade
	for i := range sells {
		if sells[i].Side == domain.TradeSideSell {
			sell = &sells[i]
		}
	}
	if sell == nil || sell.Situation == nil || sell.Situation.OpenInterest == nil {
		t.Fatalf("sell keeps the whole read %+v", sell)
	}
}

func TestPlaceOrder_EmptyReasonStoresNoRead(t *testing.T) {
	svc := newSvc(t, nil)
	svc.SetSituationSource(&fakeSituation{got: fullSituation()})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-none", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "reason-none", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		Reason: domain.SituationReason{Set: true},
	}); err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListTrades(ctx, "reason-none", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation != nil {
		t.Fatalf("empty reason %+v", list[0].Situation)
	}
}

func TestPlacePendingOrder_FillKeepsTheNamedClauses(t *testing.T) {
	svc := newSvc(t, nil)
	src := &fakeSituation{got: fullSituation()}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-lim", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	ord, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "reason-lim", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 90, Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ord.ReasonSet || len(ord.Reason) != 1 || ord.Situation == nil || ord.Situation.OpenInterest != nil {
		t.Fatalf("order %+v reason %v", ord.Situation, ord.Reason)
	}
	reloaded, err := svc.store.GetPendingOrder(ctx, "reason-lim", ord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.ReasonSet || len(reloaded.Reason) != 1 || reloaded.Reason[0] != domain.SituationClausePriceCVD {
		t.Fatalf("reloaded reason %+v", reloaded.Reason)
	}
	src.got = &domain.CoinSituation{
		Symbol:       "BTCUSDT",
		PriceCVD:     &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown},
		OpenInterest: &domain.SituationOpenInterest{Flow: domain.SituationOIClosed},
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *reloaded, 90, 0); err != nil || !ok {
		t.Fatalf("fill ok=%v err=%v", ok, err)
	}
	list, _, err := svc.ListTrades(ctx, "reason-lim", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation == nil || list[0].Situation.PriceCVD == nil ||
		list[0].Situation.PriceCVD.Kind != domain.SituationCVDPriceDown || list[0].Situation.OpenInterest != nil {
		t.Fatalf("fill %+v", list[0].Situation)
	}
}

func TestPreviewOrder_ReportsTheLiveRead(t *testing.T) {
	svc := newSvc(t, nil)
	svc.SetSituationSource(&fakeSituation{got: fullSituation()})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "reason-prev", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-prev", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
		Reason: reasonOnlyPrice(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.SituationStatus != domain.SituationPreviewReady || got.Situation == nil || got.Situation.OpenInterest == nil {
		t.Fatalf("preview read %+v %s", got.Situation, got.SituationStatus)
	}
	if !got.ReasonSet || len(got.Reason) != 1 || got.Reason[0] != domain.SituationClausePriceCVD {
		t.Fatalf("preview reason %+v", got.Reason)
	}
	svc.situations = &fakeSituation{err: domain.ErrUpstream}
	failed, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "reason-prev", Symbol: "BTCUSDT", Side: "buy", Type: "market", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed.SituationStatus != domain.SituationPreviewFailed || failed.FillPrice <= 0 {
		t.Fatalf("failed preview still prices: %+v", failed.SituationStatus)
	}
}
