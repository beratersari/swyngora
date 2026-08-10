package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestIdempotency_MarketPendingMargin(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "idemp", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}

	tr1, view1, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp", Symbol: "BTCUSDT", Side: "buy", Quantity: 1, IdempotencyKey: "buy-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tr2, view2, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp", Symbol: "BTCUSDT", Side: "buy", Quantity: 1, IdempotencyKey: "buy-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr1.ID != tr2.ID {
		t.Fatalf("retry created a second trade: %s vs %s", tr1.ID, tr2.ID)
	}
	if math.Abs(view1.CashBalance-view2.CashBalance) > 1e-9 {
		t.Fatalf("cash changed on retry %v -> %v", view1.CashBalance, view2.CashBalance)
	}
	_, total, err := svc.ListTrades(ctx, "idemp", 20, 0)
	if err != nil || total != 1 {
		t.Fatalf("trades total=%d err=%v", total, err)
	}

	_, _, err = svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp", Symbol: "BTCUSDT", Side: "buy", Quantity: 2, IdempotencyKey: "buy-1",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict on different request, got %v", err)
	}

	o1, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "idemp", Symbol: "ETHUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 40, IdempotencyKey: "pend-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	o2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "idemp", Symbol: "ETHUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 40, IdempotencyKey: "pend-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if o1.ID != o2.ID {
		t.Fatalf("pending retry new id %s vs %s", o1.ID, o2.ID)
	}

	pos1, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "idemp", Symbol: "BTCUSDT", Side: "long", Type: "market", Quantity: 0.1, Leverage: 2, IdempotencyKey: "m-open",
	})
	if err != nil || pos1 == nil {
		t.Fatalf("margin open %+v %v", pos1, err)
	}
	pos2, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "idemp", Symbol: "BTCUSDT", Side: "long", Type: "market", Quantity: 0.1, Leverage: 2, IdempotencyKey: "m-open",
	})
	if err != nil || pos2 == nil {
		t.Fatalf("margin retry %+v %v", pos2, err)
	}
	if pos1.ID != pos2.ID {
		t.Fatalf("margin retry new position %s vs %s", pos1.ID, pos2.ID)
	}

	cl1, trc1, err := svc.CloseMarginPosition(ctx, MarginCloseInput{
		ClientID: "idemp", PositionID: pos1.ID, IdempotencyKey: "m-close",
	})
	if err != nil {
		t.Fatal(err)
	}
	cl2, trc2, err := svc.CloseMarginPosition(ctx, MarginCloseInput{
		ClientID: "idemp", PositionID: pos1.ID, IdempotencyKey: "m-close",
	})
	if err != nil {
		t.Fatal(err)
	}
	if trc1.ID != trc2.ID {
		t.Fatalf("close retry new trade %s vs %s", trc1.ID, trc2.ID)
	}
	if cl1.Status != domain.MarginPositionClosed || cl2.Status != domain.MarginPositionClosed {
		t.Fatalf("close status %s %s", cl1.Status, cl2.Status)
	}
}

func TestIdempotency_InvalidKeyAndNoKey(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "idemp2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp2", Symbol: "BTCUSDT", Side: "buy", Quantity: 1, IdempotencyKey: "has space",
	}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want invalid key, got %v", err)
	}
	tr1, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp2", Symbol: "BTCUSDT", Side: "buy", Quantity: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	tr2, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp2", Symbol: "BTCUSDT", Side: "buy", Quantity: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr1.ID == tr2.ID {
		t.Fatal("missing key must not replay")
	}
	// Failed first attempt must not consume the key.
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp2", Symbol: "BTCUSDT", Side: "buy", Quantity: 1e8, IdempotencyKey: "later-ok",
	}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want insufficient cash, got %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "idemp2", Symbol: "BTCUSDT", Side: "buy", Quantity: 0.1, IdempotencyKey: "later-ok",
	}); err != nil {
		t.Fatalf("retry after failed first should succeed: %v", err)
	}
}
