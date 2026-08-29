package scanner

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeCandles struct {
	byKey     map[string][]domain.Candle
	lastLimit int
}

func (f *fakeCandles) GetCandles(_ context.Context, exchange, symbol, interval string, limit int, _, _ *time.Time) ([]domain.Candle, error) {
	f.lastLimit = limit
	k := exchange + "|" + symbol + "|" + interval
	c := f.byKey[k]
	if len(c) > limit {
		return c[len(c)-limit:], nil
	}
	return c, nil
}

type fakeWatch struct {
	wl *domain.Watchlist
}

func (f *fakeWatch) Get(_ context.Context, actorClientID, ownerClientID string) (*domain.WatchlistAccess, error) {
	owner := ownerClientID
	if owner == "" {
		owner = actorClientID
	}
	if f.wl == nil || f.wl.ClientID != owner {
		return nil, domain.ErrNotFound
	}
	return &domain.WatchlistAccess{Watchlist: *f.wl, OwnerClientID: owner, Role: domain.WatchlistRoleOwner}, nil
}

func TestScanner_CreateRunDedupe(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Volume spike candles
	candles := make([]domain.Candle, 25)
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		vol := "100"
		if i == 24 {
			vol = "400"
		}
		candles[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    "10",
			Volume:   vol,
		}
	}
	market := &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|BTCUSDT|1h": candles,
	}}
	watch := &fakeWatch{wl: &domain.Watchlist{
		ClientID: "scan-user",
		Items: []domain.WatchlistItem{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"},
		},
	}}
	svc := New(st, market, watch)
	ctx := context.Background()

	rule, err := svc.Create(ctx, CreateInput{
		ClientID: "scan-user", Type: "volume_increase", Interval: "1h",
		VolumeLookback: 20, VolumeMinRatio: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.RunOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("first run n=%d err=%v", n, err)
	}
	// Same market data — no new result
	n, err = svc.RunOnce(ctx)
	if err != nil || n != 0 {
		t.Fatalf("dedupe n=%d err=%v", n, err)
	}
	list, total, err := svc.ListResults(ctx, "scan-user", 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("results total=%d list=%d err=%v", total, len(list), err)
	}
	if list[0].RuleID != rule.ID || list[0].Symbol != "BTCUSDT" {
		t.Fatalf("%+v", list[0])
	}
	page, err := svc.ListResultsPage(ctx, "scan-user", 10, 0)
	if err != nil || page == nil || page.Total != 1 || len(page.Results) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	if page.Hits24h < 0 {
		t.Fatalf("hits24h=%d", page.Hits24h)
	}

	// Same condition still true on the next bar — no new result
	candles2 := append([]domain.Candle{}, candles...)
	next := candles[24]
	next.OpenTime = candles[24].OpenTime.Add(time.Hour)
	next.Volume = "500"
	candles2 = append(candles2, next)
	market.byKey["binance|BTCUSDT|1h"] = candles2
	n, err = svc.RunOnce(ctx)
	if err != nil || n != 0 {
		t.Fatalf("still-true bar n=%d err=%v", n, err)
	}
	// Condition goes false, then true again — new result
	quiet := candles2[len(candles2)-1]
	quiet.OpenTime = quiet.OpenTime.Add(time.Hour)
	quiet.Volume = "100"
	candles2 = append(candles2, quiet)
	market.byKey["binance|BTCUSDT|1h"] = candles2
	n, err = svc.RunOnce(ctx)
	if err != nil || n != 0 {
		t.Fatalf("false bar n=%d err=%v", n, err)
	}
	again := quiet
	again.OpenTime = quiet.OpenTime.Add(time.Hour)
	again.Volume = "600"
	candles2 = append(candles2, again)
	market.byKey["binance|BTCUSDT|1h"] = candles2
	n, err = svc.RunOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("re-trigger n=%d err=%v", n, err)
	}
}

type closedAccounts map[string]bool

func (m closedAccounts) IsClosed(_ context.Context, clientID string) (bool, *domain.Account, error) {
	return m[clientID], nil, nil
}

func TestScanner_RunOnceSkipsClosedAccount(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	candles := make([]domain.Candle, 25)
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		vol := "100"
		if i == 24 {
			vol = "400"
		}
		candles[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    "10",
			Volume:   vol,
		}
	}
	svc := New(st, &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|BTCUSDT|1h": candles,
	}}, &fakeWatch{wl: &domain.Watchlist{
		ClientID: "scan-closed",
		Items:    []domain.WatchlistItem{{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"}},
	}})
	svc.SetAccountChecker(closedAccounts{"scan-closed": true})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "scan-closed", Type: "volume_increase", Interval: "1h",
		VolumeLookback: 20, VolumeMinRatio: 2,
	}); err != nil {
		t.Fatal(err)
	}
	n, err := svc.RunOnce(ctx)
	if err != nil || n != 0 {
		t.Fatalf("closed tenant must not insert results n=%d err=%v", n, err)
	}
}

func TestScanner_CreateValidation(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st, &fakeCandles{}, &fakeWatch{})
	ctx := context.Background()
	_, err = svc.Create(ctx, CreateInput{ClientID: "u", Type: "nope"})
	if err == nil {
		t.Fatal("want invalid type")
	}
	_, err = svc.Create(ctx, CreateInput{
		ClientID: "u", Type: "rsi", RSICondition: "below", RSIThreshold: 30, RSIPeriod: 14,
	})
	if err != nil {
		t.Fatal(err)
	}
	combo, err := svc.Create(ctx, CreateInput{
		ClientID:     "u",
		Conditions:   []string{"rsi", "volume_increase"},
		MatchMode:    "any",
		RSICondition: "below", RSIThreshold: 30,
		VolumeLookback: 20, VolumeMinRatio: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if combo.Type != domain.ScannerRuleCombo || combo.MatchMode != domain.ScannerMatchAny || len(combo.Conditions) != 2 {
		t.Fatalf("%+v", combo)
	}
}

func TestScanner_ComboLookbackUsesLongestCondition(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	market := &fakeCandles{byKey: map[string][]domain.Candle{
		"binance|BTCUSDT|1h": {{OpenTime: time.Now().UTC(), Close: "10", Volume: "1"}},
	}}
	svc := New(st, market, &fakeWatch{wl: &domain.Watchlist{
		ClientID: "look-user",
		Items:    []domain.WatchlistItem{{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"}},
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "look-user", Interval: "1h",
		Conditions:   []string{"rsi", "ma_crossover"},
		RSICondition: "below", RSIThreshold: 30, RSIPeriod: 14,
		MAFastPeriod: 12, MASlowPeriod: 200, MADirection: "golden_cross",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if market.lastLimit < 230 {
		t.Fatalf("combo lookback should follow MA(200), got %d", market.lastLimit)
	}
}

func TestScanner_UpdateEnableAndParams(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st, &fakeCandles{}, &fakeWatch{})
	ctx := context.Background()
	rule, err := svc.Create(ctx, CreateInput{
		ClientID: "u", Type: "rsi", RSICondition: "below", RSIThreshold: 30, RSIPeriod: 14,
	})
	if err != nil {
		t.Fatal(err)
	}
	off := false
	th := 22.0
	period := 21
	got, err := svc.Update(ctx, UpdateInput{
		ClientID: "u", ID: rule.ID, Enabled: &off, RSIThreshold: &th, RSIPeriod: &period,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled || got.RSIThreshold != 22 || got.RSIPeriod != 21 {
		t.Fatalf("%+v", got)
	}
	en, err := svc.ListEnabledRules(ctx)
	if err != nil || len(en) != 0 {
		t.Fatalf("disabled rule still listed %+v %v", en, err)
	}
	on := true
	got, err = svc.Update(ctx, UpdateInput{ClientID: "u", ID: rule.ID, Enabled: &on})
	if err != nil || !got.Enabled {
		t.Fatalf("re-enable %+v %v", got, err)
	}
}
