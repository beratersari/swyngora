package portfolio

import (
	"context"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeSituation struct {
	got *domain.CoinSituation
	n   int
}

func (f *fakeSituation) GetCoinSituation(context.Context, string) (*domain.CoinSituation, error) {
	f.n++
	return f.got, nil
}

func TestPlaceOrder_StoresCoinSituationOnTheFill(t *testing.T) {
	svc := newSvc(t, nil)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:       "BTCUSDT",
		PriceCVD:     &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		OpenInterest: &domain.SituationOpenInterest{Flow: domain.SituationOIOpened, Window: "1h"},
		Liquidation:  &domain.SituationLiquidation{Side: domain.SituationLiqLong, DistancePct: 1.2},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-mkt", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-mkt", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListTrades(ctx, "sit-mkt", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation == nil || list[0].Situation.PriceCVD == nil || list[0].Situation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("stored %+v", list)
	}
	if list[0].Situation.Liquidation == nil || list[0].Situation.Liquidation.Side != domain.SituationLiqLong {
		t.Fatalf("liq %+v", list[0].Situation)
	}
}

func TestPlacePendingOrder_KeepsTheReadFromPlacement(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:      "BTCUSDT",
		PriceCVD:    &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
		Liquidation: &domain.SituationLiquidation{Side: domain.SituationLiqNone},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-lim", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	ord, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "sit-lim", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if src.n != 1 {
		t.Fatalf("captures %d", src.n)
	}
	reloaded, err := svc.store.GetPendingOrder(ctx, "sit-lim", ord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Situation == nil || reloaded.Situation.PriceCVD == nil || reloaded.Situation.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("pending %+v", reloaded.Situation)
	}
	src.got = &domain.CoinSituation{Symbol: "BTCUSDT", PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown}}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *reloaded, 90, 0); err != nil || !ok {
		t.Fatalf("fill ok=%v err=%v", ok, err)
	}
	if src.n != 1 {
		t.Fatalf("fill must keep the placement read, captures %d", src.n)
	}
	list, _, err := svc.ListTrades(ctx, "sit-lim", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation == nil || list[0].Situation.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("fill kept %+v", list)
	}
}

func TestPlaceOrder_EquitySkipsTheFuturesRead(t *testing.T) {
	px := &fakePx{prices: map[string]string{"nasdaq|AAPL": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{Symbol: "AAPL"}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-eq", StartingBalance: 10000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-eq", Exchange: "nasdaq", Symbol: "AAPL", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if src.n != 0 {
		t.Fatalf("equity called the futures read %d times", src.n)
	}
}
