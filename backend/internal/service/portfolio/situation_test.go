package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeSituation struct {
	got *domain.CoinSituation
	err error
	n   int
}

func (f *fakeSituation) GetCoinSituation(context.Context, string) (*domain.CoinSituation, error) {
	f.n++
	if f.err != nil {
		return nil, f.err
	}
	return f.got, nil
}

func TestSituationAlertWatch_NamesABlankBuyAndAFlatBook(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	flat, err := svc.SituationAlertWatch(ctx, "watch-flat", "binance", "BTCUSDT")
	if err != nil || flat.Status != domain.SituationWatchFlat {
		t.Fatalf("flat %+v %v", flat, err)
	}
	eq, err := svc.SituationAlertWatch(ctx, "watch-flat", "nasdaq", "AAPL")
	if err != nil || eq.Status != domain.SituationWatchFlat {
		t.Fatalf("equity %+v %v", eq, err)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "watch-blank", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "watch-blank", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	blank, err := svc.SituationAlertWatch(ctx, "watch-blank", "binance", "BTCUSDT")
	if err != nil || blank.Status != domain.SituationWatchBlank {
		t.Fatalf("blank %+v %v", blank, err)
	}
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc.SetSituationSource(src)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "watch-stored", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "watch-stored", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := svc.SituationAlertWatch(ctx, "watch-stored", "binance", "BTCUSDT")
	if err != nil || stored.Status != domain.SituationWatchStored {
		t.Fatalf("stored %+v %v", stored, err)
	}
	src.got = nil
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "watch-stored", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	newer, err := svc.SituationAlertWatch(ctx, "watch-stored", "binance", "BTCUSDT")
	if err != nil || newer.Status != domain.SituationWatchBlank {
		t.Fatalf("newest blank %+v %v", newer, err)
	}
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

func TestPlacePendingOrder_StoresTheReadAtTheFill(t *testing.T) {
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
		t.Fatalf("placement captures %d", src.n)
	}
	reloaded, err := svc.store.GetPendingOrder(ctx, "sit-lim", ord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Situation == nil || reloaded.Situation.PriceCVD == nil || reloaded.Situation.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("pending %+v", reloaded.Situation)
	}
	src.got = &domain.CoinSituation{Symbol: "BTCUSDT", PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown}}
	filled, ok, err := svc.TryFillPendingOrder(ctx, *reloaded, 90, 0)
	if err != nil || !ok {
		t.Fatalf("fill ok=%v err=%v", ok, err)
	}
	if src.n != 2 {
		t.Fatalf("fill captures %d", src.n)
	}
	if filled == nil || filled.Situation == nil || filled.Situation.PriceCVD == nil || filled.Situation.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("order keeps placement %+v", filled)
	}
	list, _, err := svc.ListTrades(ctx, "sit-lim", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation == nil || list[0].Situation.PriceCVD == nil || list[0].Situation.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("fill read %+v", list)
	}
	view, err := svc.View(ctx, "sit-lim")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation == nil || view.Positions[0].EntrySituation.PriceCVD == nil || view.Positions[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("entry %+v", view.Positions)
	}
}

func TestPlacePendingOrder_EachPartialFillStoresItsOwnRead(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-part", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	ord, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "sit-part", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := svc.store.GetPendingOrder(ctx, "sit-part", ord.ID)
	if err != nil {
		t.Fatal(err)
	}
	src.got = &domain.CoinSituation{Symbol: "BTCUSDT", PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp}}
	first, ok, err := svc.TryFillPendingOrder(ctx, *reloaded, 90, 0.4)
	if err != nil || !ok || first == nil || first.Status != domain.PendingStatusOpen {
		t.Fatalf("partial %+v ok=%v err=%v", first, ok, err)
	}
	src.got = &domain.CoinSituation{Symbol: "BTCUSDT", PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown}}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *first, 90, 0); err != nil || !ok {
		t.Fatalf("rest ok=%v err=%v", ok, err)
	}
	list, _, err := svc.ListTrades(ctx, "sit-part", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("trades %d", len(list))
	}
	got := map[string]bool{}
	for _, tr := range list {
		if tr.Situation == nil || tr.Situation.PriceCVD == nil {
			t.Fatalf("blank fill %+v", tr.Situation)
		}
		got[tr.Situation.PriceCVD.Kind] = true
	}
	if !got[domain.SituationCVDPriceUp] || !got[domain.SituationCVDPriceDown] {
		t.Fatalf("partial reads %+v", got)
	}
	view, err := svc.View(ctx, "sit-part")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation == nil || view.Positions[0].EntrySituation.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("newest fill %+v", view.Positions[0].EntrySituation)
	}
}

func TestPlacePendingOrder_BlankFillReadDoesNotKeepPlacement(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-miss", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	ord, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "sit-miss", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := svc.store.GetPendingOrder(ctx, "sit-miss", ord.ID)
	if err != nil {
		t.Fatal(err)
	}
	src.err = errors.New("tape down")
	filled, ok, err := svc.TryFillPendingOrder(ctx, *reloaded, 90, 0)
	if err != nil || !ok {
		t.Fatalf("fill ok=%v err=%v", ok, err)
	}
	if filled == nil || filled.Situation == nil || filled.Situation.PriceCVD == nil || filled.Situation.PriceCVD.Kind != domain.SituationCVDAgrees {
		t.Fatalf("order %+v", filled)
	}
	list, _, err := svc.ListTrades(ctx, "sit-miss", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation != nil {
		t.Fatalf("fill kept a read %+v", list)
	}
	view, err := svc.View(ctx, "sit-miss")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].EntrySituation != nil {
		t.Fatalf("entry %+v", view.Positions)
	}
}

func TestPlacePendingOrder_EquityStillSkipsTheRead(t *testing.T) {
	px := &fakePx{prices: map[string]string{"nasdaq|AAPL": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{Symbol: "AAPL"}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-eq-lim", StartingBalance: 10000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	ord, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "sit-eq-lim", Exchange: "nasdaq", Symbol: "AAPL", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := svc.store.GetPendingOrder(ctx, "sit-eq-lim", ord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := svc.TryFillPendingOrder(ctx, *reloaded, 90, 0); err != nil || !ok {
		t.Fatalf("fill ok=%v err=%v", ok, err)
	}
	if src.n != 0 {
		t.Fatalf("equity captures %d", src.n)
	}
	list, _, err := svc.ListTrades(ctx, "sit-eq-lim", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Situation != nil {
		t.Fatalf("equity fill %+v", list)
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
	reads := view.Positions[0].OpenReads
	if len(reads) != 2 || !reads[0].Newest || reads[0].Situation == nil || reads[0].Situation.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("newest lot %+v", reads)
	}
	if reads[1].Newest || reads[1].Situation == nil || reads[1].Situation.PriceCVD.Kind != domain.SituationCVDPriceUp || reads[0].Quantity != 1 || reads[1].Quantity != 1 {
		t.Fatalf("older lot %+v", reads)
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
	reads = view.Positions[0].OpenReads
	if len(reads) != 2 || reads[0].Situation != nil || !reads[0].Newest {
		t.Fatalf("blank newest lot %+v", reads)
	}
	if reads[1].Situation == nil || reads[1].Situation.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("older lot stayed %+v", reads[1])
	}
}

func TestSituationChange_ListsEveryRemainingBuy(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "nasdaq|AAPL": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-lots", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-lots", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceDown},
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-lots", Symbol: "BTCUSDT", Side: "buy", Quantity: 0.4,
	}); err != nil {
		t.Fatal(err)
	}
	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
	}
	got, err := svc.SituationChange(ctx, "sit-lots", "binance", "BTCUSDT")
	if err != nil || !got.LiveOK || got.Entry == nil || got.Entry.PriceCVD.Kind != domain.SituationCVDPriceDown {
		t.Fatalf("entry %+v %v", got, err)
	}
	if len(got.OpenReads) != 2 || got.OpenReads[0].Quantity != 0.4 || !got.OpenReads[0].Newest {
		t.Fatalf("lots %+v", got.OpenReads)
	}
	if got.OpenReads[0].Comparison == nil || got.OpenReads[0].Comparison.Same || got.OpenReads[0].Comparison.Clauses[0].Entry != domain.SituationCVDPriceDown {
		t.Fatalf("newest cmp %+v", got.OpenReads[0].Comparison)
	}
	if got.OpenReads[1].Quantity != 1 || got.OpenReads[1].Comparison == nil || got.OpenReads[1].Comparison.Clauses[0].Entry != domain.SituationCVDPriceUp {
		t.Fatalf("older cmp %+v", got.OpenReads[1])
	}
	if got.Comparison.Clauses[0].Entry != domain.SituationCVDPriceDown {
		t.Fatalf("headline still newest %+v", got.Comparison)
	}
	src.err = errors.New("tape down")
	quiet, err := svc.SituationChange(ctx, "sit-lots", "binance", "BTCUSDT")
	if err != nil || quiet.LiveOK || len(quiet.OpenReads) != 2 || quiet.OpenReads[0].Comparison != nil || quiet.OpenReads[0].Situation == nil {
		t.Fatalf("live failed %+v %v", quiet, err)
	}
	eq, err := svc.SituationChange(ctx, "sit-lots", "nasdaq", "AAPL")
	if err != nil || eq.Entry != nil || len(eq.OpenReads) != 0 {
		t.Fatalf("equity %+v %v", eq, err)
	}
}

func TestSituationChange_NamesTheClauseThatLeftAndTheBlankOne(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	src := &fakeSituation{got: &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		Funding:  &domain.SituationFunding{Payer: "long"},
	}}
	svc.SetSituationSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "sit-left", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	buy, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-left", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	src.got = &domain.CoinSituation{
		Symbol:         "BTCUSDT",
		PriceCVD:       &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees},
		FundingMissing: []string{"binance", "okx"},
		Size:           &domain.SituationSize{Kind: domain.SituationSizeBothBuy},
	}
	sell, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "sit-left", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListTrades(ctx, "sit-left", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	var closed *domain.Trade
	for i := range list {
		if list[i].ID == sell.ID {
			closed = &list[i]
		}
	}
	if closed == nil || len(closed.Left) != 1 || closed.Left[0].SourceTradeID != buy.ID {
		t.Fatalf("sell left %+v", closed)
	}
	cmp := closed.Left[0].Comparison
	if cmp.Same || len(cmp.Clauses) != 2 {
		t.Fatalf("cmp %+v", cmp)
	}
	if cmp.Clauses[0].Clause != "priceCvd" || cmp.Clauses[0].Status != domain.SituationLeftChanged || cmp.Clauses[0].Live != domain.SituationCVDAgrees {
		t.Fatalf("price %+v", cmp.Clauses[0])
	}
	if cmp.Clauses[1].Clause != "funding" || cmp.Clauses[1].Status != domain.SituationLeftMissing || len(cmp.Clauses[1].Missing) != 2 {
		t.Fatalf("funding %+v", cmp.Clauses[1])
	}
	open, err := svc.SituationChange(ctx, "sit-left", "binance", "BTCUSDT")
	if err != nil || !open.LiveOK || open.Entry == nil || open.Entry.PriceCVD.Kind != domain.SituationCVDPriceUp {
		t.Fatalf("open %+v %v", open, err)
	}
	if open.Comparison.Same || len(open.Comparison.Clauses) != 2 || open.Comparison.Clauses[0].Status != domain.SituationLeftChanged {
		t.Fatalf("open cmp %+v", open.Comparison)
	}
	src.got = &domain.CoinSituation{
		Symbol:   "BTCUSDT",
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		Funding:  &domain.SituationFunding{Payer: "long"},
	}
	held, err := svc.SituationChange(ctx, "sit-left", "binance", "BTCUSDT")
	if err != nil || !held.Comparison.Same {
		t.Fatalf("still %+v %v", held, err)
	}
	eq, err := svc.SituationChange(ctx, "sit-left", "nasdaq", "AAPL")
	if err != nil || eq.LiveOK || eq.Entry != nil || eq.Comparison.Same {
		t.Fatalf("equity %+v %v", eq, err)
	}
	one, err := svc.TradeSituation(ctx, "sit-left", sell.ID)
	if err != nil || one == nil || len(one.Left) != 1 {
		t.Fatalf("one trade %+v %v", one, err)
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
