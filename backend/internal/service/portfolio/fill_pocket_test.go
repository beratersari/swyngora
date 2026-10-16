package portfolio

import (
	"context"
	"errors"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakePocket struct {
	got domain.CloserPocket
	err error
	n   int
}

func (f *fakePocket) CloserPocket(context.Context, string) (domain.CloserPocket, error) {
	f.n++
	return f.got, f.err
}

func TestFillPocket_UsesTheNewestSpotFillNotTheFee(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	svc.WithPaperCosts(func(domain.Exchange) domain.TradingCost {
		return domain.TradingCost{FeeRate: 0.001}
	})
	src := &fakePocket{got: domain.CloserPocket{Covered: true, Side: domain.SituationLiqLong, Price: 99}}
	svc.SetPocketSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "pocket-spot", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "pocket-spot", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "110"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "pocket-spot", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.FillPocket(ctx, "pocket-spot", "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Open || got.FillKind != domain.FillPocketSpot || got.FillPrice != 110 {
		t.Fatalf("newest fill %+v", got)
	}
	view, err := svc.View(ctx, "pocket-spot")
	if err != nil {
		t.Fatal(err)
	}
	lot, ok := domain.LatestOpenLot(view.Positions[0].Lots)
	if !ok || lot.Price <= got.FillPrice {
		t.Fatalf("fee-inclusive lot %v should sit above the fill %v", lot.Price, got.FillPrice)
	}
	if got.PocketPrice != 99 || got.PocketFromFill != domain.FillPocketBelow || got.GapPct == nil || *got.GapPct != 10 {
		t.Fatalf("pocket %+v", got)
	}
	if src.n != 1 {
		t.Fatalf("reads %d", src.n)
	}
}

func TestFillPocket_NamesAMissingVenueAndSkipsAFlatBook(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	src := &fakePocket{got: domain.CloserPocket{Missing: []string{"bybit"}}}
	svc.SetPocketSource(src)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "pocket-miss", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	flat, err := svc.FillPocket(ctx, "pocket-miss", "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if flat.Open || src.n != 0 {
		t.Fatalf("flat book must not read the map: %+v reads %d", flat, src.n)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "pocket-miss", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.FillPocket(ctx, "pocket-miss", "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.PocketPrice != 0 || got.GapPct != nil || len(got.Missing) != 1 || got.Missing[0] != "bybit" {
		t.Fatalf("%+v", got)
	}
	if got.FillPrice != 100 {
		t.Fatalf("fill %v", got.FillPrice)
	}
}

func TestFillPocket_MarginEntryAndPaperLiquidation(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	svc.SetPocketSource(&fakePocket{got: domain.CloserPocket{Covered: true, Side: domain.SituationLiqLong, Price: 90}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "pocket-margin", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "pocket-margin", Symbol: "BTCUSDT", Side: "long", Type: "market", Quantity: 1, Leverage: 5,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.FillPocket(ctx, "pocket-margin", "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.FillKind != domain.FillPocketMargin || got.FillPrice != 100 || got.MarginLiqPrice != 80.5 {
		t.Fatalf("%+v", got)
	}
	if got.GapPct == nil || *got.GapPct != 10 || got.MarginGapPct == nil || *got.MarginGapPct != 10.6 {
		t.Fatalf("gaps fill %v margin %v", got.GapPct, got.MarginGapPct)
	}
}

func TestFillPocket_EquityAndBadVenue(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "pocket-eq", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	eq, err := svc.FillPocket(ctx, "pocket-eq", "nasdaq", "AAPL")
	if err != nil || !eq.Equity || eq.Open {
		t.Fatalf("%+v err %v", eq, err)
	}
	_, err = svc.FillPocket(ctx, "pocket-eq", "", "BTCUSDT")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("exchange %v", err)
	}
}
