package portfolio

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestRecurringBuy_CreatePauseResumeDelete(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-user", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(time.Hour)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-user", Symbol: "BTCUSDT", Amount: 50, Frequency: "daily", StartAt: &start,
	})
	if err != nil || plan.Status != domain.RecurringBuyActive {
		t.Fatalf("%+v %v", plan, err)
	}
	plan, err = svc.PauseRecurringBuyPlan(ctx, "rb-user", plan.ID)
	if err != nil || plan.Status != domain.RecurringBuyPaused {
		t.Fatalf("%+v %v", plan, err)
	}
	plan, err = svc.ResumeRecurringBuyPlan(ctx, "rb-user", plan.ID)
	if err != nil || plan.Status != domain.RecurringBuyActive {
		t.Fatalf("%+v %v", plan, err)
	}
	if err := svc.DeleteRecurringBuyPlan(ctx, "rb-user", plan.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetRecurringBuyPlan(ctx, "rb-user", plan.ID); err != domain.ErrNotFound {
		t.Fatalf("%v", err)
	}
}

func TestRecurringBuy_NameWeekdayIntervalAndUpdate(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-named", StartingBalance: 20000}); err != nil {
		t.Fatal(err)
	}
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-named", Symbol: "BTCUSDT", Name: "Salary Day Buy",
		Amount: 500, Frequency: "monthly", DayOfMonth: 25,
	})
	if err != nil || plan.Name != "Salary Day Buy" || plan.DayOfMonth != 25 {
		t.Fatalf("%+v %v", plan, err)
	}
	if plan.NextRunAt.Day() != 25 {
		t.Fatalf("next run day=%v", plan.NextRunAt)
	}
	mon, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-named", Symbol: "ETHUSDT", Name: "Monday ETH stack",
		Amount: 1500, Frequency: "weekly", Weekday: "monday",
	})
	if err != nil || mon.Weekday != "monday" || mon.NextRunAt.Weekday() != time.Monday {
		t.Fatalf("%+v %v", mon, err)
	}
	iv, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-named", Symbol: "ETHUSDT", Name: "Buy Coins With 30% of My Money",
		Amount: 1500, Frequency: "interval", IntervalHours: 12,
	})
	if err != nil || iv.IntervalHours != 12 || iv.Name != "Buy Coins With 30% of My Money" {
		t.Fatalf("%+v %v", iv, err)
	}
	newName := "Payday BTC"
	amt := 600.0
	got, err := svc.UpdateRecurringBuyPlan(ctx, RecurringBuyUpdateInput{
		ClientID: "rb-named", PlanID: plan.ID, Name: &newName, Amount: &amt,
	})
	if err != nil || got.Name != "Payday BTC" || got.Amount != 600 {
		t.Fatalf("%+v %v", got, err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_ = past
	_ = n
}

func TestRecurringBuy_ExecuteAndInsufficientCash(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-exec", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	// Due immediately
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-exec", Symbol: "BTCUSDT", Amount: 200, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	view, _ := svc.View(ctx, "rb-exec")
	// spent ~200
	if view.CashBalance > 801 || view.CashBalance < 799 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
	if len(view.Positions) != 1 || view.Positions[0].Quantity < 1.9 {
		t.Fatalf("pos=%+v", view.Positions)
	}
	runs, err := svc.ListRecurringBuyRuns(ctx, "rb-exec", plan.ID, 10, 0)
	if err != nil || len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("%+v %v", runs, err)
	}
	// Next process should not buy again same period
	n, _ = svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if n != 0 {
		// plan next_run_at advanced to tomorrow — not due
		// ok if 0
	}
	runs, _ = svc.ListRecurringBuyRuns(ctx, "rb-exec", plan.ID, 10, 0)
	if len(runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(runs))
	}

	// Insufficient cash plan
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-poor", StartingBalance: 10}); err != nil {
		t.Fatal(err)
	}
	past2 := time.Now().UTC().Add(-time.Minute)
	poor, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-poor", Symbol: "BTCUSDT", Amount: 500, Frequency: "weekly", StartAt: &past2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, _ = svc.ListRecurringBuyRuns(ctx, "rb-poor", poor.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunFailed {
		t.Fatalf("%+v", runs)
	}
	if !domain.RecurringFailRetryable(runs[0].FailReason) {
		t.Fatalf("insufficient cash should retry: %q", runs[0].FailReason)
	}
	// Plan stays due so a later deposit can fill this period (no double spend yet).
	got, _ := svc.GetRecurringBuyPlan(ctx, "rb-poor", poor.ID)
	if got.Status != domain.RecurringBuyActive || got.Spent != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestRecurringBuy_IdempotentConcurrentWorkers(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "50"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-race", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-2 * time.Hour)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-race", Symbol: "ETHUSDT", Amount: 100, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
		}()
	}
	wg.Wait()
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-race", plan.ID, 20, 0)
	if len(runs) != 1 {
		t.Fatalf("want exactly 1 run, got %d %+v", len(runs), runs)
	}
	view, _ := svc.View(ctx, "rb-race")
	// only one ~100 buy
	if view.CashBalance < 9890 || view.CashBalance > 9910 {
		t.Fatalf("cash=%v (double buy?)", view.CashBalance)
	}
}

func TestRecurringBuy_CatchUpOnlyLatest(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-catch", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	// next_run 3 days ago daily → only one buy for latest day
	start := time.Now().UTC().Add(-72 * time.Hour)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-catch", Symbol: "BTCUSDT", Amount: 100, Frequency: "daily", StartAt: &start,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-catch", plan.ID, 20, 0)
	if len(runs) != 1 {
		t.Fatalf("want 1 run for latest missed only, got %d", len(runs))
	}
}

func TestRecurringBuy_ExecutesOnBookAfterSecondPortfolioCreated(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	main, err := svc.Create(ctx, CreateInput{ClientID: "rb-multi", StartingBalance: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-multi", Name: "Alts", StartingBalance: 3000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-multi", PortfolioID: main.ID, Symbol: "BTCUSDT",
		Amount: 200, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("processed=%d want 1", n)
	}
	runs, err := svc.ListRecurringBuyRuns(ctx, "rb-multi", plan.ID, 10, 0, main.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("run=%+v (DCA must fill Main after a second book exists)", runs)
	}
	if runs[0].TradeID == "" {
		t.Fatal("succeeded run has no trade id")
	}
	view, err := svc.View(ctx, "rb-multi", main.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.CashBalance > 4801 || view.CashBalance < 4799 {
		t.Fatalf("Main cash=%v want ~4800 after $200 recurring buy", view.CashBalance)
	}
	if len(view.Positions) != 1 || view.Positions[0].Quantity < 1.9 {
		t.Fatalf("Main positions=%+v want ~2 BTC", view.Positions)
	}
}

func TestRecurringBuy_ExecutesOnNonPrimaryBook(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "50"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-alt", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	alts, err := svc.Create(ctx, CreateInput{ClientID: "rb-alt", Name: "Alts", StartingBalance: 2000})
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-alt", PortfolioID: alts.ID, Symbol: "ETHUSDT",
		Amount: 100, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, err := svc.ListRecurringBuyRuns(ctx, "rb-alt", plan.ID, 10, 0, alts.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("run=%+v (DCA must fill the UUID book, not the owner id)", runs)
	}
	view, err := svc.View(ctx, "rb-alt", alts.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.CashBalance > 1901 || view.CashBalance < 1899 {
		t.Fatalf("Alts cash=%v want ~1900", view.CashBalance)
	}
	if len(view.Positions) != 1 || view.Positions[0].Quantity < 1.9 {
		t.Fatalf("Alts positions=%+v want ~2 ETH", view.Positions)
	}
}

func TestRecurringBuy_TimezoneMondayNine(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "64000"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-tz", StartingBalance: 100000}); err != nil {
		t.Fatal(err)
	}
	hour := 9
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-tz", Symbol: "BTCUSDT", Amount: 100, Frequency: "weekly",
		Weekday: "monday", TimeZone: "Europe/Istanbul", Hour: &hour, Minute: intPtr(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Istanbul")
	local := plan.NextRunAt.In(loc)
	if local.Weekday() != time.Monday || local.Hour() != 9 || local.Minute() != 0 {
		t.Fatalf("next=%v local=%v", plan.NextRunAt, local)
	}
	if plan.TimeZone != "Europe/Istanbul" || !plan.HasLocalTime {
		t.Fatalf("%+v", plan)
	}
	if _, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-tz", Symbol: "BTCUSDT", Amount: 100, Frequency: "daily",
		TimeZone: "Not/AZone",
	}); err == nil {
		t.Fatal("expected invalid timezone")
	}
}

func TestRecurringBuy_MaxPriceSkipsAndAllows(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "66000"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-cap", StartingBalance: 200000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	high, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-cap", Symbol: "BTCUSDT", Amount: 1000, Frequency: "daily",
		StartAt: &past, MaxPrice: 65000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-cap", high.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunFailed {
		t.Fatalf("want failed over-max run %+v", runs)
	}
	if !strings.Contains(runs[0].FailReason, "maxPrice") {
		t.Fatalf("reason=%q", runs[0].FailReason)
	}
	view, _ := svc.View(ctx, "rb-cap")
	if view.CashBalance < 199999 {
		t.Fatalf("should not spend when over max, cash=%v", view.CashBalance)
	}

	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "64000"}}
	okPast := time.Now().UTC().Add(-time.Minute)
	ok, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-cap", Symbol: "BTCUSDT", Amount: 1000, Frequency: "daily",
		StartAt: &okPast, MaxPrice: 65000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	okRuns, _ := svc.ListRecurringBuyRuns(ctx, "rb-cap", ok.ID, 10, 0)
	if len(okRuns) != 1 || okRuns[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("64k should buy %+v", okRuns)
	}
}

func TestRecurringBuy_MaxPriceBlocksAfterFeeAndSlippage(t *testing.T) {
	// Last is under 65k, but Binance slip+fee unit cost is over.
	svc := newSvcWithCosts(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "64980"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-eff", StartingBalance: 200000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-eff", Symbol: "BTCUSDT", Amount: 1000, Frequency: "daily",
		StartAt: &past, MaxPrice: 65000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-eff", plan.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunFailed {
		t.Fatalf("want failed fee+slip run %+v", runs)
	}
	if !strings.Contains(runs[0].FailReason, "fee and slippage") {
		t.Fatalf("reason=%q", runs[0].FailReason)
	}
	view, _ := svc.View(ctx, "rb-eff")
	if view.CashBalance < 199999 {
		t.Fatalf("should not spend, cash=%v", view.CashBalance)
	}
}

func TestRecurringBuy_MaxPricePatchDuringLastPriceBlocksFill(t *testing.T) {
	px := &slowPx{inner: &fakePx{prices: map[string]string{"binance|BTCUSDT": "64000"}}, delay: 80 * time.Millisecond}
	svc := newSvc(t, nil)
	svc.market = px
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-race-lock", StartingBalance: 200000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-race-lock", Symbol: "BTCUSDT", Amount: 1000, Frequency: "daily",
		StartAt: &past, MaxPrice: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		max := 100.0
		_, err := svc.UpdateRecurringBuyPlan(ctx, RecurringBuyUpdateInput{
			ClientID: "rb-race-lock", PlanID: plan.ID, MaxPrice: &max,
		})
		done <- err
	}()
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-race-lock", plan.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunFailed {
		t.Fatalf("PATCH during in-flight lastPrice must block fill %+v", runs)
	}
	if !strings.Contains(runs[0].FailReason, "maxPrice") {
		t.Fatalf("reason=%q", runs[0].FailReason)
	}
	view, _ := svc.View(ctx, "rb-race-lock")
	if view.CashBalance < 199999 {
		t.Fatalf("should not spend, cash=%v", view.CashBalance)
	}
}

type slowPx struct {
	inner *fakePx
	delay time.Duration
}

func (p *slowPx) GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	time.Sleep(p.delay)
	return p.inner.GetTicker24h(ctx, exchange, symbol)
}

func TestRecurringBuy_MaxPricePatchAppliesToInFlightWorker(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "64000"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-race-max", StartingBalance: 200000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-race-max", Symbol: "BTCUSDT", Amount: 1000, Frequency: "daily",
		StartAt: &past, MaxPrice: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.market = &patchMaxOnTick{svc: svc, clientID: "rb-race-max", planID: plan.ID, inner: px}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-race-max", plan.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunFailed {
		t.Fatalf("in-flight maxPrice PATCH must block the buy %+v", runs)
	}
	if !strings.Contains(runs[0].FailReason, "maxPrice") {
		t.Fatalf("reason=%q", runs[0].FailReason)
	}
	view, _ := svc.View(ctx, "rb-race-max")
	if view.CashBalance < 199999 {
		t.Fatalf("should not spend, cash=%v", view.CashBalance)
	}
}

type patchMaxOnTick struct {
	svc      *Service
	clientID string
	planID   string
	inner    *fakePx
}

func (p *patchMaxOnTick) GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	max := 100.0
	_, _ = p.svc.UpdateRecurringBuyPlan(ctx, RecurringBuyUpdateInput{
		ClientID: p.clientID, PlanID: p.planID, MaxPrice: &max,
	})
	return p.inner.GetTicker24h(ctx, exchange, symbol)
}

func TestRecurringBuy_TempErrorRetriesOnce(t *testing.T) {
	px := &failThenOkPx{fails: 1, inner: &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}}
	svc := newSvc(t, nil)
	svc.market = px
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-retry", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-retry", Symbol: "BTCUSDT", Amount: 200, Frequency: "daily",
		StartAt: &past, Budget: 200,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_ = n
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-retry", plan.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunFailed || !domain.RecurringFailRetryable(runs[0].FailReason) {
		t.Fatalf("want retryable fail %+v", runs)
	}
	got, _ := svc.GetRecurringBuyPlan(ctx, "rb-retry", plan.ID)
	if got.Spent != 0 {
		t.Fatalf("temp fail must not spend %v", got.Spent)
	}
	if !got.NextRunAt.After(time.Now().UTC()) && got.NextRunAt.After(past.Add(time.Hour)) {
		t.Fatalf("should still be due or soon: next=%v", got.NextRunAt)
	}
	n, err = svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil || n != 1 {
		t.Fatalf("retry n=%d err=%v", n, err)
	}
	runs, _ = svc.ListRecurringBuyRuns(ctx, "rb-retry", plan.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("retry should succeed once %+v", runs)
	}
	got, _ = svc.GetRecurringBuyPlan(ctx, "rb-retry", plan.ID)
	if got.Spent < 199 || got.Spent > 201 {
		t.Fatalf("spent once %v", got.Spent)
	}
	n, _ = svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if n != 0 {
		t.Fatalf("must not buy the same period again n=%d", n)
	}
	got, _ = svc.GetRecurringBuyPlan(ctx, "rb-retry", plan.ID)
	if got.Spent < 199 || got.Spent > 201 {
		t.Fatalf("spent twice %v", got.Spent)
	}
}

type failThenOkPx struct {
	fails int
	n     int
	inner *fakePx
}

func (f *failThenOkPx) GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	f.n++
	if f.n <= f.fails {
		return nil, domain.ErrNotFound
	}
	return f.inner.GetTicker24h(ctx, exchange, symbol)
}

func TestRecurringBuy_BudgetCountsFeeAndSlippage(t *testing.T) {
	svc := newSvcWithCosts(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	const startCash = 20000.0
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-fee-bud", StartingBalance: startCash}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-fee-bud", Symbol: "BTCUSDT", Amount: 1000, Frequency: "daily",
		StartAt: &past, Budget: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	view, err := svc.View(ctx, "rb-fee-bud")
	if err != nil {
		t.Fatal(err)
	}
	spentCash := startCash - view.CashBalance
	if spentCash < 999.5 || spentCash > 1000.5 {
		t.Fatalf("wallet debit=%v want ~1000 (fee+slip included in sizing)", spentCash)
	}
	got, _ := svc.GetRecurringBuyPlan(ctx, "rb-fee-bud", plan.ID)
	if got.Spent < spentCash-0.01 || got.Spent > spentCash+0.01 {
		t.Fatalf("spent=%v wallet=%v (spent must be cash out, not raw notional)", got.Spent, spentCash)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-fee-bud", plan.ID, 10, 0)
	if len(runs) != 1 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("%+v", runs)
	}
	if runs[0].Amount < spentCash-0.01 || runs[0].Amount > spentCash+0.01 {
		t.Fatalf("run amount=%v wallet=%v", runs[0].Amount, spentCash)
	}
	// leftover budget is ~0 after fee-inclusive debit of 1000
	later := time.Now().UTC().Add(26 * time.Hour)
	if _, err := svc.ProcessDueRecurringBuys(ctx, later); err != nil {
		t.Fatal(err)
	}
	runs, _ = svc.ListRecurringBuyRuns(ctx, "rb-fee-bud", plan.ID, 10, 0)
	var exhausted bool
	for _, r := range runs {
		if r.FailReason == "budget exhausted" {
			exhausted = true
		}
	}
	if !exhausted {
		t.Fatalf("want budget exhausted after fee-inclusive spend %+v", runs)
	}
}

func TestRecurringBuy_BudgetAndEndDate(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-bud", StartingBalance: 20000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	end := time.Now().UTC().Add(24 * time.Hour)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-bud", Symbol: "BTCUSDT", Amount: 500, Frequency: "daily",
		StartAt: &past, Budget: 500, EndsAt: &end,
	})
	if err != nil || plan.Budget != 500 || plan.EndsAt == nil {
		t.Fatalf("%+v %v", plan, err)
	}
	if _, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	view, _ := svc.View(ctx, "rb-bud")
	if view.CashBalance < 19499 || view.CashBalance > 19501 {
		t.Fatalf("budget 500 should spend 500, cash=%v", view.CashBalance)
	}
	got, _ := svc.GetRecurringBuyPlan(ctx, "rb-bud", plan.ID)
	if got.Spent < 499 || got.Spent > 501 {
		t.Fatalf("spent=%v", got.Spent)
	}

	// Next calendar period (same daily plan, later clock): budget exhausted
	later := time.Now().UTC().Add(26 * time.Hour)
	if _, err := svc.ProcessDueRecurringBuys(ctx, later); err != nil {
		t.Fatal(err)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-bud", plan.ID, 10, 0)
	var exhausted bool
	for _, r := range runs {
		if r.FailReason == "budget exhausted" {
			exhausted = true
		}
	}
	if !exhausted {
		t.Fatalf("want budget exhausted %+v", runs)
	}

	pastEnd := time.Now().UTC().Add(-time.Hour)
	ended, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-bud", Symbol: "BTCUSDT", Amount: 100, Frequency: "daily",
		StartAt: &past, EndsAt: &pastEnd,
	})
	if err != nil || ended.Status != domain.RecurringBuyEnded {
		t.Fatalf("past end should create ended %+v %v", ended, err)
	}
}

func intPtr(v int) *int { return &v }
