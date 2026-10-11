package portfolio

import (
	"context"
	"testing"
	"time"

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

func TestPlaceMarginOrder_StoresEntrySituation(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m-sit", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, ord, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m-sit", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil || ord != nil || pos == nil || pos.EntrySituation == nil || pos.EntrySituation.PriceCVD == nil || pos.EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("pos=%+v ord=%v err=%v", pos, ord, err)
	}
	if _, _, err := svc.CloseMarginPosition(ctx, MarginCloseInput{
		ClientID: "m-sit", PositionID: pos.ID, Quantity: 0.4,
	}); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListMarginPositions(ctx, "m-sit")
	if err != nil || len(list) != 1 || list[0].EntrySituation == nil || list[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("partial close dropped the read: %+v %v", list, err)
	}
}

func TestPlaceMarginLimit_KeepsPlacementSituation(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m-lim", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, ord, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m-lim", Symbol: "BTCUSDT", Side: "long", Type: "limit",
		Quantity: 1, Leverage: 5, LimitPrice: 90,
	})
	if err != nil || pos != nil || ord == nil || ord.Situation == nil || ord.Situation.PriceCVD == nil || ord.Situation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("ord=%+v err=%v", ord, err)
	}
	src.got = &domain.CoinSituation{Symbol: "BTCUSDT", PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown}}
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "90"}}
	filled, _, _, err := svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil || filled != 1 {
		t.Fatalf("filled=%d err=%v", filled, err)
	}
	if src.n != 1 {
		t.Fatalf("fill read the tape again: %d", src.n)
	}
	list, err := svc.ListMarginPositions(ctx, "m-lim")
	if err != nil || len(list) != 1 || list[0].EntrySituation == nil || list[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("fill replaced the placement read: %+v %v", list, err)
	}
}

func TestEntrySituation_UsesTheNewestOpenBuyAcrossBooks(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-entry", Name: "one", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-entry", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown},
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-entry", Name: "two", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	books, err := svc.List(ctx, "sit-entry")
	if err != nil || len(books) != 2 {
		t.Fatalf("books %+v %v", books, err)
	}
	second := books[0].ID
	if books[0].Name == "one" {
		second = books[1].ID
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-entry", PortfolioID: second, Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.EntrySituation(ctx, "sit-entry", "binance", "BTCUSDT")
	if err != nil || got == nil || got.PriceCVD == nil || got.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("newest book %+v %v", got, err)
	}
	if eq, err := svc.EntrySituation(ctx, "sit-entry", "nasdaq", "AAPL"); err != nil || eq != nil {
		t.Fatalf("equity %+v %v", eq, err)
	}
}

func TestView_EntrySituationFollowsTheNewestOpenBuy(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "nasdaq|AAPL": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-open", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-open", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown},
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-open", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	view, err := svc.View(ctx, "sit-open")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation == nil || view.Positions[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("newest buy %+v", view.Positions[0].EntrySituation)
	}
	if view.Positions[0].EntryAt.IsZero() {
		t.Fatal("entry time")
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-open", Symbol: "BTCUSDT", Side: "sell", Quantity: 1, LotMethod: "lifo",
	}); err != nil {
		t.Fatal(err)
	}
	view, err = svc.View(ctx, "sit-open")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation == nil || view.Positions[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("older lot remains %+v", view.Positions[0].EntrySituation)
	}
	svc.situations = nil
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-open", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	view, err = svc.View(ctx, "sit-open")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation != nil || !view.Positions[0].EntryAt.IsZero() {
		t.Fatalf("blank newest buy %+v at %v", view.Positions[0].EntrySituation, view.Positions[0].EntryAt)
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
	view, err := svc.View(ctx, "sit-eq")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation != nil || !view.Positions[0].EntryAt.IsZero() {
		t.Fatalf("equity position %+v", view.Positions)
	}
}
