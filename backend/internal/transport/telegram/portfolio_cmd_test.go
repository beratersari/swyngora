package telegram

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

func bypassRate(r *Router) {
	r.mu.Lock()
	r.lastAt = map[int64]time.Time{}
	r.mu.Unlock()
}

func newPaperRouter(t *testing.T) *Router {
	t.Helper()
	fm := &fakeMarket{}
	ms := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  fm,
		domain.ExchangeCoinbase: fm,
		domain.ExchangeBybit:    fm,
	}, fakeSupply{})
	ws := watchlist.New(watchliststore.NewMemory())
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "tg-pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := portfolio.New(st, ms).WithPaperCosts(domain.ZeroTradingCosts)
	return NewRouter(ms, ws, Options{
		DefaultExchange: "binance", LowMcapLimit: 10, AllowAll: true, Portfolio: ps,
	})
}

func TestPortfolio_CreateAndView(t *testing.T) {
	r := newPaperRouter(t)
	ctx := context.Background()
	out := r.Handle(ctx, 1, 77, "/portfolio")
	if !strings.Contains(out, "create") {
		t.Fatalf("want create hint: %s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 1, 77, "/portfolio create 5000")
	if !strings.Contains(out, "created") && !strings.Contains(out, "5000") {
		t.Fatalf("%s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 1, 77, "/pf")
	if !(strings.Contains(out, "Paper") && (strings.Contains(out, "5000") || strings.Contains(out, "5,000"))) {
		t.Fatalf("%s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 1, 77, "/portfolio create 2000 Risky")
	if !strings.Contains(out, "Risky") {
		t.Fatalf("named create: %s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 1, 77, "/portfolio list")
	if !strings.Contains(out, "Main") || !strings.Contains(out, "Risky") {
		t.Fatalf("list: %s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 1, 77, "/portfolio use Main")
	if !strings.Contains(out, "Main") {
		t.Fatalf("use: %s", out)
	}
}

func TestBuy_RequiresConfirmThenFills(t *testing.T) {
	r := newPaperRouter(t)
	ctx := context.Background()
	bypassRate(r)
	r.Handle(ctx, 5, 88, "/portfolio create 10000")
	bypassRate(r)
	prev := r.HandleMessage(ctx, 5, 88, "/buy BTCUSDT 2")
	if !strings.Contains(prev.Text, "Confirm") || !strings.Contains(prev.Text, "BTCUSDT") {
		t.Fatalf("preview: %s", prev.Text)
	}
	if !strings.Contains(prev.Text, "200") { // 2 * 100
		t.Fatalf("want total 200: %s", prev.Text)
	}
	if len(prev.Keyboard) == 0 || len(prev.Keyboard[0]) < 2 {
		t.Fatalf("want confirm/cancel buttons: %+v", prev.Keyboard)
	}
	confirm := prev.Keyboard[0][0].CallbackData
	if !strings.HasPrefix(confirm, "pf:c:") {
		t.Fatalf("callback=%s", confirm)
	}
	// Trade must not have executed yet.
	view, err := r.portfolio.View(ctx, "tg-88")
	if err != nil || view.CashBalance != 10000 {
		t.Fatalf("pre-confirm cash=%v err=%v", view, err)
	}

	done := r.HandleCallback(ctx, 5, 88, confirm)
	if !strings.Contains(done.Text, "filled") && !strings.Contains(done.Text, "Filled") {
		t.Fatalf("fill: %s", done.Text)
	}
	if !done.ClearKeyboard {
		t.Fatal("want buttons cleared after fill")
	}
	view, err = r.portfolio.View(ctx, "tg-88")
	if err != nil || view.CashBalance != 9800 {
		t.Fatalf("post cash=%v err=%v", view.CashBalance, err)
	}

	again := r.HandleCallback(ctx, 5, 88, confirm)
	if !strings.Contains(strings.ToLower(again.Text), "expired") && !strings.Contains(strings.ToLower(again.Text), "already") {
		t.Fatalf("second confirm: %s", again.Text)
	}
}

func TestBuy_CancelDoesNotTrade(t *testing.T) {
	r := newPaperRouter(t)
	ctx := context.Background()
	bypassRate(r)
	r.Handle(ctx, 6, 89, "/portfolio create 10000")
	bypassRate(r)
	prev := r.HandleMessage(ctx, 6, 89, "/buy BTCUSDT 1")
	cancel := prev.Keyboard[0][1].CallbackData
	out := r.HandleCallback(ctx, 6, 89, cancel)
	if !strings.Contains(strings.ToLower(out.Text), "cancel") {
		t.Fatalf("%s", out.Text)
	}
	view, _ := r.portfolio.View(ctx, "tg-89")
	if view.CashBalance != 10000 {
		t.Fatalf("cash after cancel=%v", view.CashBalance)
	}
}

func TestBuy_WrongUserRejected(t *testing.T) {
	r := newPaperRouter(t)
	ctx := context.Background()
	bypassRate(r)
	r.Handle(ctx, 7, 90, "/portfolio create 10000")
	bypassRate(r)
	prev := r.HandleMessage(ctx, 7, 90, "/buy BTCUSDT 1")
	data := prev.Keyboard[0][0].CallbackData
	out := r.HandleCallback(ctx, 7, 91, data)
	if !strings.Contains(strings.ToLower(out.Text), "another") {
		t.Fatalf("%s", out.Text)
	}
	view, _ := r.portfolio.View(ctx, "tg-90")
	if view.CashBalance != 10000 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
}

func TestSell_PreviewAndFill(t *testing.T) {
	r := newPaperRouter(t)
	ctx := context.Background()
	bypassRate(r)
	r.Handle(ctx, 8, 92, "/portfolio create 10000")
	bypassRate(r)
	buy := r.HandleMessage(ctx, 8, 92, "/buy BTCUSDT 1")
	r.HandleCallback(ctx, 8, 92, buy.Keyboard[0][0].CallbackData)
	bypassRate(r)
	prev := r.HandleMessage(ctx, 8, 92, "/sell BTCUSDT 1")
	if !strings.Contains(prev.Text, "SELL") || !strings.Contains(prev.Text, "BTCUSDT") {
		t.Fatalf("%s", prev.Text)
	}
	out := r.HandleCallback(ctx, 8, 92, prev.Keyboard[0][0].CallbackData)
	if !strings.Contains(strings.ToLower(out.Text), "filled") {
		t.Fatalf("%s", out.Text)
	}
	view, _ := r.portfolio.View(ctx, "tg-92")
	if view.CashBalance < 9999 {
		t.Fatalf("cash after roundtrip=%v", view.CashBalance)
	}
}

func TestTelegram_DepositWithdrawHistory(t *testing.T) {
	r := newPaperRouter(t)
	ctx := context.Background()
	bypassRate(r)
	r.Handle(ctx, 10, 94, "/portfolio create 10000")
	bypassRate(r)
	out := r.Handle(ctx, 10, 94, "/deposit 500 salary")
	if !strings.Contains(out, "Deposit") || !strings.Contains(out, "500") {
		t.Fatalf("%s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 10, 94, "/withdraw 100")
	if !strings.Contains(out, "Withdrawal") {
		t.Fatalf("%s", out)
	}
	bypassRate(r)
	out = r.Handle(ctx, 10, 94, "/cash")
	if !strings.Contains(out, "Cash history") || !strings.Contains(out, "Opening") {
		t.Fatalf("%s", out)
	}
	view, _ := r.portfolio.View(ctx, "tg-94")
	if view.CashBalance != 10400 || view.TotalPnL != 0 {
		t.Fatalf("cash=%v pnl=%v", view.CashBalance, view.TotalPnL)
	}
}

func TestBuy_NoPortfolio(t *testing.T) {
	r := newPaperRouter(t)
	out := r.Handle(context.Background(), 9, 93, "/buy BTCUSDT 1")
	if !strings.Contains(out, "create") {
		t.Fatalf("%s", out)
	}
}

func TestEncodeInlineKeyboard(t *testing.T) {
	s := encodeInlineKeyboard(confirmCancelKeyboard("abc123"))
	if !strings.Contains(s, "Confirm") || !strings.Contains(s, "pf:c:abc123") || !strings.Contains(s, "pf:x:abc123") {
		t.Fatalf("%s", s)
	}
}
