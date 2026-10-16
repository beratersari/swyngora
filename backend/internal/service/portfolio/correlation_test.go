package portfolio

import (
	"context"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type corrCandles struct {
	bars map[string][]domain.Candle
}

func (c corrCandles) GetCandles(_ context.Context, exchange, symbol, _ string, _ int, _, _ *time.Time) ([]domain.Candle, error) {
	rows, ok := c.bars[exchange+"|"+symbol]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return rows, nil
}

func corrPath(start float64, n int, same bool) []domain.Candle {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	px := start
	out := make([]domain.Candle, 0, n)
	for i := 0; i < n; i++ {
		if same {
			if i%2 == 0 {
				px *= 1.02
			} else {
				px *= 0.98
			}
		} else if i%2 == 0 {
			px *= 0.98
		} else {
			px *= 1.02
		}
		out = append(out, domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * 5 * time.Minute),
			Close:    formatPx(px),
		})
	}
	return out
}

func formatPx(v float64) string {
	return strconv.FormatFloat(v, 'f', 8, 64)
}

func TestBookCorrelation_CoinsThatMoveTogether(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"binance|ETHUSDT": "50",
		"binance|SOLUSDT": "10",
	}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "corr", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "corr", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "corr", Exchange: "binance", Symbol: "ETHUSDT", Side: "buy", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	n := domain.PortfolioCorrSpec().Bars
	if n > 40 {
		n = 40
	}
	svc.SetCandleSource(corrCandles{bars: map[string][]domain.Candle{
		"binance|BTCUSDT": corrPath(100, n, true),
		"binance|ETHUSDT": corrPath(50, n, true),
		"binance|SOLUSDT": corrPath(10, n, false),
	}})
	got, err := svc.BookCorrelation(ctx, "corr", "", "", PortfolioCorrRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Together == nil || *got.Together < 0.99 {
		t.Fatalf("together %+v", got.Together)
	}
	if got.SwingPct == nil || got.ApartPct == nil || *got.SwingPct <= *got.ApartPct {
		t.Fatalf("swing %v apart %v", got.SwingPct, got.ApartPct)
	}
	added, err := svc.BookCorrelation(ctx, "corr", "", "", PortfolioCorrRequest{
		Exchange: domain.ExchangeBinance, Symbol: "SOLUSDT", Side: "long", Notional: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if added.Add == nil || !added.Add.Complete || added.Add.SwingPct == nil {
		t.Fatalf("add %+v", added.Add)
	}
	if *added.Add.SwingPct >= *got.SwingPct {
		t.Fatalf("an opposite coin should lower the swing %v -> %v", *got.SwingPct, *added.Add.SwingPct)
	}
}

func TestBookCorrelation_MissingCoinIsNotZero(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "miss", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "miss", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "miss", Exchange: "binance", Symbol: "ETHUSDT", Side: "buy", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	n := 40
	svc.SetCandleSource(corrCandles{bars: map[string][]domain.Candle{
		"binance|BTCUSDT": corrPath(100, n, true),
	}})
	got, err := svc.BookCorrelation(ctx, "miss", "", "", PortfolioCorrRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Together != nil {
		t.Fatal("a missing coin became a pair")
	}
	found := false
	for _, name := range got.Missing {
		if name == "ETH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing %+v", got.Missing)
	}
	if got.SwingPct == nil {
		t.Fatal("the coin that has minutes should still have a swing")
	}
}

func TestPreviewOrder_OppositeCoinLowersSwing(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"binance|ETHUSDT": "50",
		"binance|SOLUSDT": "10",
	}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-corr", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "prev-corr", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "prev-corr", Exchange: "binance", Symbol: "ETHUSDT", Side: "buy", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	n := 40
	svc.SetCandleSource(corrCandles{bars: map[string][]domain.Candle{
		"binance|BTCUSDT": corrPath(100, n, true),
		"binance|ETHUSDT": corrPath(50, n, true),
		"binance|SOLUSDT": corrPath(10, n, false),
	}})
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-corr", Exchange: "binance", Symbol: "SOLUSDT", Side: "buy", Quantity: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Correlation == nil || got.Correlation.SwingPct == nil || got.Correlation.Add == nil {
		t.Fatalf("preview correlation %+v", got.Correlation)
	}
	if !got.Correlation.Add.Complete || got.Correlation.Add.SwingPct == nil {
		t.Fatalf("add %+v", got.Correlation.Add)
	}
	if *got.Correlation.Add.SwingPct >= *got.Correlation.SwingPct {
		t.Fatalf("opening the opposite coin should lower the swing %v -> %v", *got.Correlation.SwingPct, *got.Correlation.Add.SwingPct)
	}
	view, err := svc.View(ctx, "prev-corr")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 2 {
		t.Fatalf("preview wrote a fill: %d positions", len(view.Positions))
	}
}
