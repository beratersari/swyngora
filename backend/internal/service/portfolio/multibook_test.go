package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestMultiBook_CreateListSelectIsolate(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	ctx := context.Background()

	main, err := svc.Create(ctx, CreateInput{ClientID: "mb1", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if main.ID != "mb1" || main.Name != domain.DefaultPortfolioName {
		t.Fatalf("first book legacy id/name: %+v", main)
	}
	risky, err := svc.Create(ctx, CreateInput{ClientID: "mb1", Name: "Risky", StartingBalance: 2000})
	if err != nil {
		t.Fatal(err)
	}
	if risky.ID == "" || risky.ID == main.ID || risky.Name != "Risky" {
		t.Fatalf("second book: %+v", risky)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mb1", Name: "risky", StartingBalance: 100}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("dup name: %v", err)
	}

	list, err := svc.List(ctx, "mb1")
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	if _, err := svc.View(ctx, "mb1"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("view without id with 2 books: %v", err)
	}
	vMain, err := svc.View(ctx, "mb1", main.ID)
	if err != nil || math.Abs(vMain.CashBalance-10000) > 1e-9 || vMain.Name != "Main" {
		t.Fatalf("main view %+v %v", vMain, err)
	}
	vRisk, err := svc.View(ctx, "mb1", "Risky")
	if err != nil || math.Abs(vRisk.CashBalance-2000) > 1e-9 || vRisk.ID != risky.ID {
		t.Fatalf("risky by name %+v %v", vRisk, err)
	}

	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "mb1", PortfolioID: risky.ID, Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	vRisk, _ = svc.View(ctx, "mb1", risky.ID)
	if math.Abs(vRisk.CashBalance-1900) > 1e-6 || len(vRisk.Positions) != 1 {
		t.Fatalf("risky after buy %+v", vRisk)
	}
	vMain, _ = svc.View(ctx, "mb1", main.ID)
	if math.Abs(vMain.CashBalance-10000) > 1e-9 || len(vMain.Positions) != 0 {
		t.Fatalf("main must be isolated %+v", vMain)
	}

	renamed, err := svc.Rename(ctx, "mb1", risky.ID, "Moon bag")
	if err != nil || renamed.Name != "Moon bag" {
		t.Fatalf("rename %+v %v", renamed, err)
	}
	if _, err := svc.View(ctx, "mb1", "Risky"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("old name: %v", err)
	}

	if err := svc.Delete(ctx, "mb1", risky.ID); err != nil {
		t.Fatal(err)
	}
	list, err = svc.List(ctx, "mb1")
	if err != nil || len(list) != 1 || list[0].ID != main.ID {
		t.Fatalf("after delete %+v %v", list, err)
	}
	// Sole book again — portfolioId optional.
	v, err := svc.View(ctx, "mb1")
	if err != nil || v.ID != main.ID {
		t.Fatalf("sole view %+v %v", v, err)
	}
}
