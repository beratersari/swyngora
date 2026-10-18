package pricealert

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/alertstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func newSvc(t *testing.T) (*Service, *alertstore.SQLite) {
	t.Helper()
	store, err := alertstore.Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	// Tests use httptest (loopback) and offline hostnames; production leaves this false.
	svc := New(store)
	svc.AllowPrivateWebhooks = true
	return svc, store
}

func TestService_SetWebhook_SSRFBlockedByDefault(t *testing.T) {
	store, err := alertstore.Open(filepath.Join(t.TempDir(), "ssrf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := New(store) // AllowPrivateWebhooks == false
	_, err = svc.SetWebhook(context.Background(), "ssrf-user", domain.WebhookSettings{
		URL: "http://127.0.0.1/hook", DeliveryMode: "immediate",
	})
	if err == nil {
		t.Fatal("expected SSRF rejection for loopback webhook")
	}
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestService_CreateListGetDelete(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "user-1", Exchange: "binance", Symbol: "btcusdt",
		Condition: "above", TargetPrice: 90000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Symbol != "BTCUSDT" || a.Status != domain.AlertStatusActive {
		t.Fatalf("%+v", a)
	}
	list, err := svc.List(ctx, "user-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	got, err := svc.Get(ctx, "user-1", a.ID)
	if err != nil || got.ID != a.ID {
		t.Fatalf("%+v %v", got, err)
	}
	if err := svc.Delete(ctx, "user-1", a.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get(ctx, "user-1", a.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestService_Validation(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, CreateInput{ClientID: "", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty client: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{ClientID: "c", Symbol: "BTCUSDT", Condition: "sideways", TargetPrice: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad cond: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{ClientID: "c", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 0})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad price: %v", err)
	}
}

func TestService_MaxAlerts(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	for i := 0; i < domain.MaxPriceAlertsPerClient; i++ {
		_, err := svc.Create(ctx, CreateInput{
			ClientID: "maxc", Symbol: "S" + strconv.Itoa(i) + "USDT",
			Condition: "above", TargetPrice: float64(i + 1),
		})
		if err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
	}
	_, err := svc.Create(ctx, CreateInput{
		ClientID: "maxc", Symbol: "OVERFLOWUSDT", Condition: "above", TargetPrice: 1,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected max error: %v", err)
	}
}

func TestService_PersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.db")
	ctx := context.Background()

	store1, err := alertstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	svc1 := New(store1)
	a, err := svc1.Create(ctx, CreateInput{
		ClientID: "restart-user", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2500,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = store1.Close()

	store2, err := alertstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	svc2 := New(store2)
	list, err := svc2.List(ctx, "restart-user")
	if err != nil || len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("after restart: %+v err=%v", list, err)
	}
}

type fakeTicker struct {
	mu     sync.Mutex
	prices map[string]string // exchange|symbol -> last
	calls  int
}

func (f *fakeTicker) GetTicker24h(_ context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	key := exchange + "|" + symbol
	p, ok := f.prices[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p}, nil
}

type closedAccounts map[string]bool

func (m closedAccounts) IsClosed(_ context.Context, clientID string) (bool, *domain.Account, error) {
	return m[clientID], nil, nil
}

func TestChecker_SkipsClosedAccount(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "closed-user", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	mkt := &fakeTicker{prices: map[string]string{"binance|BTCUSDT": "200"}}
	c := &Checker{Alerts: svc, Market: mkt, Accounts: closedAccounts{"closed-user": true}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "closed-user", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("closed tenant must not fire: %+v", got)
	}
	c.Accounts = closedAccounts{"closed-user": false}
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "closed-user", a.ID)
	if got.Status != domain.AlertStatusTriggered {
		t.Fatalf("reopened tenant should fire: %+v", got)
	}
}

func TestChecker_SituationLeavesTheEntryReadOnce(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-alert", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Condition != "leaves" || a.Kind != domain.AlertKindSituation {
		t.Fatalf("created %+v", a)
	}
	entry := &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp}}
	live := &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp}}
	c := &Checker{
		Alerts:    svc,
		Situation: fakeSituation{live: live},
		Entries:   fakeEntries{got: entry},
	}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "sit-alert", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("same read must stay active: %+v", got)
	}
	live.PriceCVD.Kind = domain.SituationCVDAgrees
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "sit-alert", a.ID)
	if got.Status != domain.AlertStatusTriggered {
		t.Fatalf("left read should fire: %+v", got)
	}
	if got.SituationFire == nil || len(got.SituationFire.Comparison.Clauses) != 1 || got.SituationFire.Comparison.Clauses[0].Clause != "priceCvd" || got.SituationFire.Comparison.Clauses[0].Entry != domain.SituationCVDPriceUp || got.SituationFire.Comparison.Clauses[0].Live != domain.SituationCVDAgrees {
		t.Fatalf("fire %+v", got.SituationFire)
	}
	live.PriceCVD.Kind = domain.SituationCVDPriceDown
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "sit-alert", a.ID)
	if got2.Status != domain.AlertStatusTriggered {
		t.Fatalf("one-time must not re-fire: %+v", got2)
	}
	if got2.SituationFire == nil || got2.SituationFire.Comparison.Clauses[0].Live != domain.SituationCVDAgrees {
		t.Fatalf("one-time replaced the fire: %+v", got2.SituationFire)
	}
}

func TestChecker_SituationFire_NamesWhatLeftAndKeepsIt(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	if _, err := svc.SetWebhook(ctx, "sit-fire", domain.WebhookSettings{
		URL: "http://127.0.0.1:9/hook", DeliveryMode: "immediate",
	}); err != nil {
		t.Fatal(err)
	}
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-fire", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation), Mode: "repeating",
	})
	if err != nil {
		t.Fatal(err)
	}
	since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	entry := &domain.CoinSituation{
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp, Since: since},
		Funding:  &domain.SituationFunding{Payer: "long", RatePct: 0.01, Since: since},
		Prints:   &domain.SituationPrints{Side: domain.SituationPrintLong, LongNotional: 10, ShortNotional: 1},
	}
	live := &domain.CoinSituation{
		PriceCVD:       &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp, Since: since},
		Funding:        &domain.SituationFunding{Payer: "long", RatePct: 0.02},
		FundingMissing: nil,
		Prints:         &domain.SituationPrints{Side: domain.SituationPrintLong, LongNotional: 80, ShortNotional: 1},
	}
	c := &Checker{Alerts: svc, Situation: fakeSituation{live: live}, Entries: fakeEntries{got: entry}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "sit-fire", a.ID)
	if got.Status != domain.AlertStatusActive || got.SituationFire != nil {
		t.Fatalf("same kinds must stay quiet: %+v fire=%+v", got.Status, got.SituationFire)
	}
	live.PriceCVD.Kind = domain.SituationCVDAgrees
	live.Funding = nil
	live.FundingMissing = []string{"okx"}
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "sit-fire", a.ID)
	if got.Status != domain.AlertStatusActive || got.Armed || got.SituationFire == nil {
		t.Fatalf("repeating fire: armed=%v status=%s fire=%+v", got.Armed, got.Status, got.SituationFire)
	}
	if got.SituationFire.Entry == nil || got.SituationFire.Entry.PriceCVD == nil || !got.SituationFire.Entry.PriceCVD.Since.IsZero() {
		t.Fatalf("fire kept the clock: %+v", got.SituationFire.Entry)
	}
	clauses := got.SituationFire.Comparison.Clauses
	if len(clauses) != 2 || clauses[0].Clause != "priceCvd" || clauses[0].Live != domain.SituationCVDAgrees || clauses[1].Clause != "funding" || clauses[1].Status != domain.SituationLeftMissing || len(clauses[1].Missing) != 1 || clauses[1].Missing[0] != "okx" {
		t.Fatalf("clauses %+v", clauses)
	}
	note, err := svc.GetNotificationByAlertID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(note.PayloadJSON), &body); err != nil {
		t.Fatal(err)
	}
	changed, _ := body["changed"].([]any)
	if len(changed) != 2 || changed[0] != "priceCvd" || changed[1] != "funding" {
		t.Fatalf("webhook changed %#v", body["changed"])
	}
	rawFire, _ := body["situationFire"].(map[string]any)
	rawCmp, _ := rawFire["comparison"].(map[string]any)
	rawClauses, _ := rawCmp["clauses"].([]any)
	if len(rawClauses) != 2 {
		t.Fatalf("webhook fire %#v", body["situationFire"])
	}
	c.RunOnce(ctx)
	held, _ := svc.Get(ctx, "sit-fire", a.ID)
	if held.Armed || held.SituationFire == nil || held.SituationFire.Comparison.Clauses[0].Live != domain.SituationCVDAgrees {
		t.Fatalf("still-left must keep the same fire: %+v", held.SituationFire)
	}
	live.PriceCVD.Kind = domain.SituationCVDPriceUp
	live.Funding = &domain.SituationFunding{Payer: "long", RatePct: 0.01}
	live.FundingMissing = nil
	c.RunOnce(ctx)
	rearmed, _ := svc.Get(ctx, "sit-fire", a.ID)
	if !rearmed.Armed || rearmed.SituationFire == nil || rearmed.SituationFire.Comparison.Clauses[0].Live != domain.SituationCVDAgrees {
		t.Fatalf("re-arm cleared the fire: armed=%v %+v", rearmed.Armed, rearmed.SituationFire)
	}
	live.PriceCVD.Kind = domain.SituationCVDPriceDown
	c.RunOnce(ctx)
	again, _ := svc.Get(ctx, "sit-fire", a.ID)
	if again.Armed || again.SituationFire == nil || again.SituationFire.Comparison.Clauses[0].Live != domain.SituationCVDPriceDown {
		t.Fatalf("next fire should replace: %+v", again.SituationFire)
	}
}

func TestService_SituationMatch_RejectsATradeAndAnEmptyRead(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-match", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
		Condition: "matches",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{
		ClientID: "sit-match", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
		Condition: "matches", Situation: &domain.SituationMatch{Funding: "sideways"},
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("unknown kind: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{
		ClientID: "sit-match", Exchange: "nasdaq", Symbol: "AAPL", Kind: string(domain.AlertKindSituation),
		Condition: "matches", Situation: &domain.SituationMatch{PriceCVD: domain.SituationCVDPriceUp},
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("equity: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{
		ClientID: "sit-match", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
		Situation: &domain.SituationMatch{PriceCVD: domain.SituationCVDPriceUp},
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("leaves must not store a selection: %v", err)
	}
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-match", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
		Condition: "matches", Mode: "repeating",
		Situation: &domain.SituationMatch{PriceCVD: domain.SituationCVDPriceUp, Funding: "long"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Condition != domain.AlertConditionMatches || a.SituationMatch == nil || a.SituationMatch.Funding != "long" || !a.Armed {
		t.Fatalf("%+v", a)
	}
}

func TestChecker_SituationMatchDoesNotNeedATrade(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	if _, err := svc.SetWebhook(ctx, "sit-match", domain.WebhookSettings{
		URL: "http://127.0.0.1:9/hook", DeliveryMode: "immediate",
	}); err != nil {
		t.Fatal(err)
	}
	repeating, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-match", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
		Condition: "matches", Mode: "repeating",
		Situation: &domain.SituationMatch{PriceCVD: domain.SituationCVDPriceUp, Prints: domain.SituationPrintLong},
	})
	if err != nil {
		t.Fatal(err)
	}
	once, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-once", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
		Condition: "matches",
		Situation: &domain.SituationMatch{Funding: "short"},
	})
	if err != nil {
		t.Fatal(err)
	}
	live := &domain.CoinSituation{
		PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp},
		Prints:   &domain.SituationPrints{Side: domain.SituationPrintShort, LongNotional: 1, ShortNotional: 9},
		Funding:  &domain.SituationFunding{Payer: "long", RatePct: 0.01},
	}
	c := &Checker{Alerts: svc, Situation: fakeSituation{live: live}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if got.Status != domain.AlertStatusActive || !got.Armed || got.SituationFire != nil {
		t.Fatalf("short prints must stay quiet: %+v", got)
	}
	quiet, _ := svc.Get(ctx, "sit-once", once.ID)
	if quiet.Status != domain.AlertStatusActive || quiet.SituationFire != nil {
		t.Fatalf("longs paying must stay quiet: %+v", quiet)
	}
	live.Prints.LongNotional = 400
	live.Prints.ShortNotional = 1
	c.RunOnce(ctx)
	dollars, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if dollars.SituationFire != nil || !dollars.Armed {
		t.Fatalf("the same print side must stay quiet: %+v", dollars.SituationFire)
	}
	live.Prints.Side = domain.SituationPrintLong
	c.RunOnce(ctx)
	fired, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if fired.Armed || fired.SituationFire == nil || !fired.SituationFire.Comparison.Same || len(fired.SituationFire.Comparison.Clauses) != 2 {
		t.Fatalf("repeating match: %+v", fired.SituationFire)
	}
	if fired.SituationFire.Comparison.Clauses[0].Status != domain.SituationMatchMet || fired.SituationFire.Live == nil || fired.SituationFire.Live.Prints == nil || fired.SituationFire.Live.Prints.LongNotional != 400 {
		t.Fatalf("fire %+v", fired.SituationFire)
	}
	note, err := svc.GetNotificationByAlertID(ctx, repeating.ID)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(note.PayloadJSON), &body); err != nil {
		t.Fatal(err)
	}
	matched, _ := body["matched"].([]any)
	if body["type"] != "situation_alert.triggered" || body["condition"] != "matches" || len(matched) != 2 || matched[0] != "priceCvd" || matched[1] != "prints" {
		t.Fatalf("webhook %#v", body)
	}
	live.Funding.Payer = "short"
	live.Prints.LongNotional = 12
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if still.Armed || still.SituationFire == nil || still.SituationFire.Live.Prints.LongNotional != 400 {
		t.Fatalf("an unselected funding change must keep the same fire: %+v", still.SituationFire)
	}
	onceGot, _ := svc.Get(ctx, "sit-once", once.ID)
	if onceGot.Status != domain.AlertStatusTriggered || onceGot.SituationFire == nil || onceGot.SituationFire.Comparison.Clauses[0].Clause != "funding" {
		t.Fatalf("one-time match: %+v", onceGot.SituationFire)
	}
	live.Prints = nil
	c.RunOnce(ctx)
	rearmed, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if !rearmed.Armed || rearmed.SituationFire == nil || rearmed.SituationFire.Comparison.Clauses[1].Live != domain.SituationPrintLong {
		t.Fatalf("a blank clause must re-arm and keep the fire: %+v", rearmed.SituationFire)
	}
	live.Prints = &domain.SituationPrints{Side: domain.SituationPrintLong, LongNotional: 3, ShortNotional: 1}
	c.RunOnce(ctx)
	again, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if again.Armed || again.SituationFire == nil || again.SituationFire.Live.Prints.LongNotional != 3 {
		t.Fatalf("the next match replaces the fire: %+v", again.SituationFire)
	}
	c.Situation = failingSituation{err: errors.New("tape down")}
	live.PriceCVD.Kind = domain.SituationCVDAgrees
	c.RunOnce(ctx)
	failed, _ := svc.Get(ctx, "sit-match", repeating.ID)
	if failed.Armed || failed.SituationFire == nil || failed.SituationFire.Live.Prints.LongNotional != 3 {
		t.Fatalf("a failed read must not re-arm or replace the fire: %+v", failed.SituationFire)
	}
}

type failingSituation struct {
	err error
}

func (f failingSituation) GetCoinSituation(context.Context, string) (*domain.CoinSituation, error) {
	return nil, f.err
}

type fakeWatch struct {
	status domain.SituationWatchStatus
	err    error
	n      int
}

func (f *fakeWatch) SituationAlertWatch(context.Context, string, string, string) (domain.SituationWatch, error) {
	f.n++
	if f.err != nil {
		return domain.SituationWatch{}, f.err
	}
	return domain.SituationWatch{Status: f.status}, nil
}

func TestList_SituationWatch_NamesABlankEntry(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	watch := &fakeWatch{status: domain.SituationWatchBlank}
	svc.SetSituationWatch(watch)
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-watch", Symbol: "BTCUSDT", Kind: string(domain.AlertKindSituation),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-watch", Symbol: "ETHUSDT", Kind: string(domain.AlertKindSituation),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-watch", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 1,
	}); err != nil {
		t.Fatal(err)
	}
	watch.n = 0
	list, err := svc.List(ctx, "sit-watch")
	if err != nil {
		t.Fatal(err)
	}
	blank := 0
	for _, a := range list {
		if domain.IsSituationAlert(a.Kind) {
			if a.SituationWatch != domain.SituationWatchBlank {
				t.Fatalf("watch %+v", a)
			}
			blank++
			continue
		}
		if a.SituationWatch != "" {
			t.Fatalf("price alert watch %q", a.SituationWatch)
		}
	}
	if blank != 2 || watch.n != 2 {
		t.Fatalf("blank=%d lookups=%d", blank, watch.n)
	}
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "sit-watch", Symbol: "SOLUSDT", Kind: string(domain.AlertKindSituation),
		Condition: "matches", Situation: &domain.SituationMatch{Crowd: domain.LongShortGapLeanTopLong},
	}); err != nil {
		t.Fatal(err)
	}
	watch.n = 0
	list, err = svc.List(ctx, "sit-watch")
	if err != nil {
		t.Fatal(err)
	}
	selected := 0
	for _, a := range list {
		if domain.IsSituationMatchAlert(a) {
			if a.SituationWatch != domain.SituationWatchSelected || a.SituationMatch == nil || a.SituationMatch.Crowd != domain.LongShortGapLeanTopLong {
				t.Fatalf("match watch %+v", a)
			}
			selected++
		}
	}
	if selected != 1 || watch.n != 2 {
		t.Fatalf("selected=%d lookups=%d", selected, watch.n)
	}
	watch.err = errors.New("book down")
	list, err = svc.List(ctx, "sit-watch")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if domain.IsSituationMatchAlert(a) {
			if a.SituationWatch != domain.SituationWatchSelected {
				t.Fatalf("match watch follows the book: %+v", a.SituationWatch)
			}
			continue
		}
		if a.SituationWatch != "" {
			t.Fatalf("lookup error should leave the watch off: %+v", a.SituationWatch)
		}
	}
}

type fakeSituation struct {
	live *domain.CoinSituation
}

func (f fakeSituation) GetCoinSituation(context.Context, string) (*domain.CoinSituation, error) {
	return f.live, nil
}

type fakeEntries struct {
	got *domain.CoinSituation
}

func (f fakeEntries) EntrySituation(context.Context, string, string, string) (*domain.CoinSituation, error) {
	return f.got, nil
}

func TestChecker_TriggersOnce(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "chk", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Below target — should not fire
	mkt := &fakeTicker{prices: map[string]string{"binance|BTCUSDT": "99"}}
	c := &Checker{Alerts: svc, Market: mkt, Logger: nil}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "chk", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("should still active: %+v", got)
	}

	// Meet target
	mkt.prices["binance|BTCUSDT"] = "100.5"
	c.RunOnce(ctx)
	got, err = svc.Get(ctx, "chk", a.ID)
	if err != nil || got.Status != domain.AlertStatusTriggered {
		t.Fatalf("%+v %v", got, err)
	}
	if got.TriggeredPrice != 100.5 {
		t.Fatalf("triggered price=%v", got.TriggeredPrice)
	}
	// Run again with higher price — must not change triggered price / re-fire
	mkt.prices["binance|BTCUSDT"] = "200"
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "chk", a.ID)
	if got2.TriggeredPrice != 100.5 || got2.Status != domain.AlertStatusTriggered {
		t.Fatalf("re-fire mutated alert: %+v", got2)
	}
}

func TestChecker_BelowAndDedupeFetch(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "a", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2000})
	_, _ = svc.Create(ctx, CreateInput{ClientID: "b", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2100})
	mkt := &fakeTicker{prices: map[string]string{"binance|ETHUSDT": "1999"}}
	c := &Checker{Alerts: svc, Market: mkt}
	c.RunOnce(ctx)
	if mkt.calls != 1 {
		t.Fatalf("want 1 ticker fetch for same pair, got %d", mkt.calls)
	}
	listA, _ := svc.List(ctx, "a")
	listB, _ := svc.List(ctx, "b")
	if listA[0].Status != domain.AlertStatusTriggered || listB[0].Status != domain.AlertStatusTriggered {
		t.Fatalf("a=%+v b=%+v", listA, listB)
	}
}

func TestChecker_DoesNotTriggerOnStaleWithoutRefresh(t *testing.T) {
	// Sanity: missing ticker does not mark triggered.
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, CreateInput{ClientID: "x", Symbol: "SOLUSDT", Condition: "above", TargetPrice: 1})
	c := &Checker{Alerts: svc, Market: &fakeTicker{prices: map[string]string{}}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "x", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("%+v", got)
	}
}

func TestChecker_RepeatingCrossAndRearm(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "rep", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100, Mode: "repeating",
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Mode != domain.AlertModeRepeating || a.Armed {
		t.Fatalf("defaults: %+v", a)
	}
	mkt := &fakeTicker{prices: map[string]string{"binance|BTCUSDT": "105"}}
	c := &Checker{Alerts: svc, Market: mkt}
	// Start already above, disarmed — must not fire
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "rep", a.ID)
	if got.Status != domain.AlertStatusActive || got.TriggeredPrice != 0 {
		t.Fatalf("no fire while already hot: %+v", got)
	}
	// Safe side re-arms
	mkt.prices["binance|BTCUSDT"] = "90"
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "rep", a.ID)
	if !got.Armed {
		t.Fatalf("expected armed after safe side: %+v", got)
	}
	// Cross up — fire, stay active, disarmed
	mkt.prices["binance|BTCUSDT"] = "101"
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "rep", a.ID)
	if got.Status != domain.AlertStatusActive || got.TriggeredPrice != 101 || got.Armed {
		t.Fatalf("after first cross: %+v", got)
	}
	// Stay above — no re-fire (triggered price unchanged)
	mkt.prices["binance|BTCUSDT"] = "120"
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "rep", a.ID)
	if got2.TriggeredPrice != 101 {
		t.Fatalf("re-fired while staying hot: %+v", got2)
	}
	// Back safe then cross again
	mkt.prices["binance|BTCUSDT"] = "95"
	c.RunOnce(ctx)
	mkt.prices["binance|BTCUSDT"] = "102"
	c.RunOnce(ctx)
	got3, _ := svc.Get(ctx, "rep", a.ID)
	if got3.TriggeredPrice != 102 || got3.Status != domain.AlertStatusActive {
		t.Fatalf("second cross: %+v", got3)
	}
}

func TestCreate_DefaultModeOneTime(t *testing.T) {
	svc, _ := newSvc(t)
	a, err := svc.Create(context.Background(), CreateInput{
		ClientID: "m", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 1,
	})
	if err != nil || a.Mode != domain.AlertModeOneTime {
		t.Fatalf("%+v %v", a, err)
	}
}

func TestCreate_InvalidMode(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.Create(context.Background(), CreateInput{
		ClientID: "m", Symbol: "ETHUSDT", Condition: "above", TargetPrice: 1, Mode: "daily",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestCreate_OrderBookImbalanceAndWall(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	imb, err := svc.Create(ctx, CreateInput{
		ClientID: "ob", Symbol: "BTCUSDT", Kind: "imbalance", Condition: "above", TargetPrice: 0.2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if imb.Kind != domain.AlertKindImbalance || imb.Mode != domain.AlertModeRepeating || !imb.Armed || imb.RangePct != 2 {
		t.Fatalf("%+v", imb)
	}
	wall, err := svc.Create(ctx, CreateInput{
		ClientID: "ob", Symbol: "ETHUSDT", Kind: "wall", Condition: "bid", TargetPrice: 0.15, RangePct: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if wall.Kind != domain.AlertKindWall || wall.RangePct != 5 || !wall.Armed {
		t.Fatalf("%+v", wall)
	}
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "ob", Symbol: "BTCUSDT", Kind: "imbalance", Condition: "above", TargetPrice: 0.01,
	}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("tiny threshold: %v", err)
	}
}

type prepTape struct {
	calls []struct {
		ex, sym string
		look    time.Duration
	}
}

func (p *prepTape) Prepare(_ context.Context, exchange, symbol string, lookback time.Duration) {
	p.calls = append(p.calls, struct {
		ex, sym string
		look    time.Duration
	}{exchange, symbol, lookback})
}

func TestCreate_LiquidationFeedAndCascade(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	feed, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_feed", Exchange: "bybit", TargetPrice: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if feed.Kind != domain.AlertKindLiqFeed || feed.Symbol != domain.LiqAlertSymbolAll || !feed.Armed || feed.Mode != domain.AlertModeRepeating {
		t.Fatalf("%+v", feed)
	}
	cas, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_cascade", Exchange: "all", Symbol: "BTCUSDT", Condition: "cascade",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cas.Kind != domain.AlertKindLiqCascade || cas.Symbol != "BTCUSDT" || string(cas.Exchange) != "all" || !cas.Armed {
		t.Fatalf("%+v", cas)
	}
	mkt, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_cascade", Symbol: "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mkt.Symbol != domain.LiqAlertSymbolAll {
		t.Fatalf("%+v", mkt)
	}
	not, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_notional", Exchange: "bybit", Symbol: "BTCUSDT",
		Condition: "long", TargetPrice: 20e6, Window: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if not.Kind != domain.AlertKindLiqNotional || not.RangePct != 5 || not.Condition != "long" || !not.Armed {
		t.Fatalf("%+v", not)
	}
}

func TestCreate_LiquidationNotionalPreparesTape(t *testing.T) {
	svc, _ := newSvc(t)
	tape := &prepTape{}
	svc.Tape = tape
	_, err := svc.Create(context.Background(), CreateInput{
		ClientID: "prep", Kind: "liquidation_notional", Exchange: "bybit", Symbol: "PEPEUSDT",
		Condition: "both", TargetPrice: 1e6, Window: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tape.calls) != 1 || tape.calls[0].sym != "PEPEUSDT" || tape.calls[0].ex != "bybit" || tape.calls[0].look != 5*time.Minute {
		t.Fatalf("prepare %+v", tape.calls)
	}
	_, err = svc.Create(context.Background(), CreateInput{
		ClientID: "prep", Kind: "liquidation_feed", Exchange: "bybit", TargetPrice: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tape.calls) != 1 {
		t.Fatalf("feed should not prepare a coin %+v", tape.calls)
	}
}

type fakeBook struct {
	mu    sync.Mutex
	books map[string]*domain.OrderBook
	calls int
}

func (f *fakeBook) GetSpotOrderBook(_ context.Context, exchange, symbol, _ string, _ int, rangePct float64) (*domain.OrderBook, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	key := exchange + "|" + symbol
	b, ok := f.books[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *b
	cp.Analysis.RangePct = rangePct
	return &cp, nil
}

func TestChecker_OrderBookImbalanceNoRetrigger(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "bk", Symbol: "BTCUSDT", Kind: "imbalance", Condition: "above", TargetPrice: 0.2,
	})
	if err != nil {
		t.Fatal(err)
	}
	books := &fakeBook{books: map[string]*domain.OrderBook{
		"binance|BTCUSDT": {Analysis: domain.OrderBookAnalysis{Imbalance: 0.3, Pressure: "buy"}},
	}}
	c := &Checker{Alerts: svc, Books: books}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "bk", a.ID)
	if got.TriggeredPrice != 0.3 || got.Armed || got.Status != domain.AlertStatusActive {
		t.Fatalf("first fire %+v", got)
	}
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "bk", a.ID)
	if got2.TriggeredPrice != 0.3 {
		t.Fatalf("re-fired while still imbalanced: %+v", got2)
	}
	books.mu.Lock()
	books.books["binance|BTCUSDT"] = &domain.OrderBook{Analysis: domain.OrderBookAnalysis{Imbalance: 0.01}}
	books.mu.Unlock()
	c.RunOnce(ctx)
	got3, _ := svc.Get(ctx, "bk", a.ID)
	if !got3.Armed {
		t.Fatalf("re-arm after clear: %+v", got3)
	}
	books.mu.Lock()
	books.books["binance|BTCUSDT"] = &domain.OrderBook{Analysis: domain.OrderBookAnalysis{Imbalance: 0.25}}
	books.mu.Unlock()
	c.RunOnce(ctx)
	got4, _ := svc.Get(ctx, "bk", a.ID)
	if got4.TriggeredPrice != 0.25 {
		t.Fatalf("second appearance: %+v", got4)
	}
}

func TestChecker_OrderBookWallAppearAndClear(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "w", Symbol: "ETHUSDT", Kind: "wall", Condition: "ask", TargetPrice: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	books := &fakeBook{books: map[string]*domain.OrderBook{
		"binance|ETHUSDT": {Analysis: domain.OrderBookAnalysis{Walls: []domain.OrderBookWall{{Side: "ask", Share: 0.4}}}},
	}}
	c := &Checker{Alerts: svc, Books: books}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "w", a.ID)
	if got.TriggeredPrice != 0.4 || got.Armed {
		t.Fatalf("wall fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "w", a.ID)
	if still.TriggeredPrice != 0.4 {
		t.Fatalf("still present re-fire %+v", still)
	}
	books.mu.Lock()
	books.books["binance|ETHUSDT"] = &domain.OrderBook{Analysis: domain.OrderBookAnalysis{}}
	books.mu.Unlock()
	c.RunOnce(ctx)
	cleared, _ := svc.Get(ctx, "w", a.ID)
	if !cleared.Armed {
		t.Fatalf("re-arm after wall gone %+v", cleared)
	}
}

type fakeLiq struct {
	feed   domain.LiquidationFeed
	rep    *domain.CascadeReport
	scan   *domain.CascadeScan
	events []domain.LiquidationEvent
	levels *domain.LiquidationLevelsReport
}

func (f *fakeLiq) GetLiquidationFeed(string) domain.LiquidationFeed { return f.feed }
func (f *fakeLiq) GetLiquidationCascade(context.Context, string, string) (*domain.CascadeReport, error) {
	return f.rep, nil
}
func (f *fakeLiq) ScanLiquidationCascades(context.Context, string) (*domain.CascadeScan, error) {
	return f.scan, nil
}
func (f *fakeLiq) ListLiquidationEvents(string, string, time.Time) []domain.LiquidationEvent {
	return f.events
}
func (f *fakeLiq) GetLiquidationLevels(context.Context, string, string, string) (*domain.LiquidationLevelsReport, error) {
	return f.levels, nil
}

func TestCreate_LiquidationLevel(t *testing.T) {
	svc, _ := newSvc(t)
	got, err := svc.Create(context.Background(), CreateInput{
		ClientID: "lv", Kind: "liquidation_level", Exchange: "binance", Symbol: "BTCUSDT",
		Condition: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.AlertKindLiqLevel || got.Condition != "high" || !got.Armed {
		t.Fatalf("%+v", got)
	}
}

func TestChecker_LevelAlertSkipsLowConfidence(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	high, err := svc.Create(ctx, CreateInput{
		ClientID: "lv", Kind: "liquidation_level", Exchange: "binance", Symbol: "BTCUSDT",
		Condition: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 14, 15, 0, 0, 0, time.UTC)
	src := &fakeLiq{
		events: []domain.LiquidationEvent{{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort,
			Price: 100.6, Notional: 20_000, Time: now.Add(-time.Minute),
		}},
		levels: &domain.LiquidationLevelsReport{
			LastPrice: "100",
			Review: &domain.LiquidationLevelsReview{
				Combined: domain.LiquidationLevelsReviewVenue{
					ByLeverage: []domain.LiquidationLevelsReviewBucket{
						{Leverage: 100, Side: domain.LiquidationSideShort, Validated: 20, HitRate: 0.6, Reliability: domain.LiquidationLevelsReliabilityHigh, Confidence: 1},
						{Leverage: 10, Side: domain.LiquidationSideShort, Validated: 20, HitRate: 0, Reliability: domain.LiquidationLevelsReliabilityLow, Confidence: 0.15},
					},
				},
			},
		},
	}
	c := &Checker{Alerts: svc, Liquidations: src, now: func() time.Time { return now }}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "lv", high.ID)
	if got.Armed {
		t.Fatalf("high-confidence 100x at last 100.6 should fire %+v", got)
	}
}

func TestChecker_LiquidationFeedNoRetrigger(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "ff", Kind: "liquidation_feed", Exchange: "bybit", TargetPrice: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	src := &fakeLiq{feed: domain.LiquidationFeed{Venues: []domain.LiquidationVenueHealth{{
		Exchange: "bybit", Live: false, LastSeenAt: now.Add(-3 * time.Minute),
	}}}}
	c := &Checker{Alerts: svc, Liquidations: src, now: func() time.Time { return now }}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "ff", a.ID)
	if got.Armed || got.TriggeredPrice < 170 {
		t.Fatalf("first fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "ff", a.ID)
	if still.TriggeredAt == nil || got.TriggeredAt == nil || !still.TriggeredAt.Equal(*got.TriggeredAt) {
		t.Fatalf("re-fired while down %+v vs %+v", still, got)
	}
	src.feed.Venues[0].Live = true
	src.feed.Venues[0].LastSeenAt = now
	c.RunOnce(ctx)
	ok, _ := svc.Get(ctx, "ff", a.ID)
	if !ok.Armed {
		t.Fatalf("re-arm when live %+v", ok)
	}
}

func TestChecker_LiquidationCascadeNamesCoin(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "cc", Kind: "liquidation_cascade", Exchange: "all", Symbol: "BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeLiq{rep: &domain.CascadeReport{
		Symbol: "BTCUSDT",
		Venues: []domain.CascadeVenue{{
			Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT",
			Grade: domain.CascadeGradeCascade, Score: 5, Side: "long", Summary: "bybit long cascade",
		}},
	}}
	c := &Checker{Alerts: svc, Liquidations: src, now: time.Now}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "cc", a.ID)
	if got.Armed || got.TriggeredPrice != 5 {
		t.Fatalf("cascade fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "cc", a.ID)
	if still.TriggeredPrice != 5 {
		t.Fatalf("re-fired %+v", still)
	}
}

func TestCreate_SetupRadarDefaultsRepeatingArmed(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "sr", Kind: "setup_radar", Exchange: "all", Symbol: "BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Kind != domain.AlertKindSetupRadar || a.Symbol != "BTCUSDT" || string(a.Exchange) != "all" {
		t.Fatalf("%+v", a)
	}
	if a.Mode != domain.AlertModeRepeating || !a.Armed {
		t.Fatalf("want repeating armed %+v", a)
	}
	all, err := svc.Create(ctx, CreateInput{
		ClientID: "sr", Kind: "setup_radar", Symbol: "all", Condition: "up", TargetPrice: 70,
	})
	if err != nil {
		t.Fatal(err)
	}
	if all.Symbol != domain.LiqAlertSymbolAll || all.Condition != "up" || all.TargetPrice != 70 {
		t.Fatalf("%+v", all)
	}
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "sr", Kind: "setup_radar", Condition: "above", TargetPrice: 60,
	}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad lean: %v", err)
	}
}

type fakeRadar struct {
	mu        sync.Mutex
	coin      *domain.AroundSimilarReport
	scan      *domain.ForensicsScanReport
	coinCalls int
	scanCalls int
}

func (f *fakeRadar) RadarCoin(context.Context, string, string) (*domain.AroundSimilarReport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.coinCalls++
	return f.coin, nil
}

func (f *fakeRadar) RadarScan(context.Context, string) (*domain.ForensicsScanReport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scanCalls++
	return f.scan, nil
}

func radarSimilar(symbol string, sim float64, lean string) *domain.AroundSimilarReport {
	up, down := 3, 0
	if lean == domain.CVDDirDown {
		up, down = 0, 3
	}
	return &domain.AroundSimilarReport{
		Symbol: symbol, Exchange: "binance", Events: 3,
		UpAfter: up, DownAfter: down,
		Matches: []domain.AroundSimilarHit{{
			Similarity: sim, Coverage: 90, AfterDirection: lean, AfterReturnPct: 2,
			Move: domain.AroundMove{At: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)},
		}},
		AfterHorizons: []domain.AroundSimilarHorizonStat{
			{Horizon: "1h", Up: up, Down: down, Sample: 3, Events: 3, MedianPct: 2},
		},
	}
}

func TestChecker_SetupRadarCoinNoRetrigger(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "rd", Kind: "setup_radar", Exchange: "binance", Symbol: "BTCUSDT",
		Condition: "any", TargetPrice: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeRadar{coin: radarSimilar("BTCUSDT", 78, domain.CVDDirUp)}
	c := &Checker{Alerts: svc, Radar: src, RadarTTL: -1, now: time.Now}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "rd", a.ID)
	if got.Armed || got.TriggeredPrice != 78 {
		t.Fatalf("first fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "rd", a.ID)
	if still.TriggeredAt == nil || got.TriggeredAt == nil || !still.TriggeredAt.Equal(*got.TriggeredAt) {
		t.Fatalf("re-fired while still similar %+v vs %+v", still, got)
	}
	src.mu.Lock()
	src.coin = &domain.AroundSimilarReport{Symbol: "BTCUSDT"}
	src.mu.Unlock()
	c.RunOnce(ctx)
	armed, _ := svc.Get(ctx, "rd", a.ID)
	if !armed.Armed {
		t.Fatalf("re-arm after clear %+v", armed)
	}
	src.mu.Lock()
	src.coin = radarSimilar("BTCUSDT", 81, domain.CVDDirUp)
	src.mu.Unlock()
	c.RunOnce(ctx)
	again, _ := svc.Get(ctx, "rd", a.ID)
	if again.Armed || again.TriggeredPrice != 81 {
		t.Fatalf("second appearance %+v", again)
	}
}

func TestChecker_SetupRadarScanStaysQuietWhileAnyHit(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "rs", Kind: "setup_radar", Exchange: "all", Symbol: "all",
		Condition: "any", TargetPrice: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeRadar{scan: &domain.ForensicsScanReport{
		Exchange: "all",
		Hits: []domain.ForensicsScanHit{
			{Symbol: "BTCUSDT", Exchange: "binance", Similarity: 82, Lean: domain.CVDDirUp, AfterStory: "rose"},
		},
	}}
	c := &Checker{Alerts: svc, Radar: src, RadarTTL: -1, now: time.Now}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "rs", a.ID)
	if got.Armed || got.TriggeredPrice != 82 {
		t.Fatalf("first fire %+v", got)
	}
	src.mu.Lock()
	src.scan.Hits = append(src.scan.Hits, domain.ForensicsScanHit{
		Symbol: "ETHUSDT", Exchange: "binance", Similarity: 70, Lean: domain.CVDDirDown, AfterStory: "fell",
	})
	src.mu.Unlock()
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "rs", a.ID)
	if still.TriggeredAt == nil || got.TriggeredAt == nil || !still.TriggeredAt.Equal(*got.TriggeredAt) {
		t.Fatalf("new coin while hot re-fired %+v vs %+v", still, got)
	}
	src.mu.Lock()
	src.scan.Hits = nil
	src.mu.Unlock()
	c.RunOnce(ctx)
	ok, _ := svc.Get(ctx, "rs", a.ID)
	if !ok.Armed {
		t.Fatalf("re-arm when no hits %+v", ok)
	}
}

func TestChecker_SetupRadarScanCached(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "rc", Kind: "setup_radar", Symbol: "all",
	}); err != nil {
		t.Fatal(err)
	}
	src := &fakeRadar{scan: &domain.ForensicsScanReport{Hits: []domain.ForensicsScanHit{}}}
	c := &Checker{Alerts: svc, Radar: src, RadarTTL: time.Hour, now: time.Now}
	c.RunOnce(ctx)
	c.RunOnce(ctx)
	src.mu.Lock()
	calls := src.scanCalls
	src.mu.Unlock()
	if calls != 1 {
		t.Fatalf("cached scan calls=%d", calls)
	}
}

func TestChecker_LiquidationNotionalNoRetriggerUntilNewWave(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "nw", Kind: "liquidation_notional", Exchange: "all", Symbol: "BTCUSDT",
		Condition: "both", TargetPrice: 20e6, Window: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	src := &fakeLiq{events: []domain.LiquidationEvent{
		{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Notional: 12e6, Time: now.Add(-2 * time.Minute)},
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort, Notional: 10e6, Time: now.Add(-1 * time.Minute)},
	}}
	c := &Checker{Alerts: svc, Liquidations: src, now: func() time.Time { return now }}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "nw", a.ID)
	if got.Armed || got.TriggeredPrice < 20e6 {
		t.Fatalf("first wave %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "nw", a.ID)
	if still.TriggeredAt == nil || got.TriggeredAt == nil || !still.TriggeredAt.Equal(*got.TriggeredAt) {
		t.Fatalf("same wave re-fired %+v vs %+v", still, got)
	}
	src.events = nil
	c.RunOnce(ctx)
	armed, _ := svc.Get(ctx, "nw", a.ID)
	if !armed.Armed {
		t.Fatalf("re-arm after wave ends %+v", armed)
	}
	src.events = []domain.LiquidationEvent{
		{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Notional: 25e6, Time: now.Add(-30 * time.Second)},
	}
	c.RunOnce(ctx)
	again, _ := svc.Get(ctx, "nw", a.ID)
	if again.Armed || again.TriggeredPrice != 25e6 {
		t.Fatalf("new wave %+v", again)
	}
}

type fakeTape struct {
	fund *domain.FundingSnapshot
}

func (f *fakeTape) GetFundingRate(context.Context, string, string, int) (*domain.FundingSnapshot, error) {
	return f.fund, nil
}
func (f *fakeTape) GetOpenInterest(context.Context, string, string) (*domain.OpenInterestSnapshot, error) {
	return nil, nil
}
func (f *fakeTape) GetTakerFlow(context.Context, string, string) (*domain.TakerFlowReport, error) {
	return nil, nil
}
func (f *fakeTape) GetFearGreed(context.Context) (*domain.FearGreedReport, error) {
	return nil, nil
}

func TestChecker_FundingAllFiresOnAnyVenue(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "fd", Kind: "funding", Exchange: "all", Symbol: "BTCUSDT",
		Condition: "above", TargetPrice: 0.05, Mode: "one_time",
	})
	if err != nil {
		t.Fatal(err)
	}
	quiet := &fakeTape{fund: &domain.FundingSnapshot{
		Exchange: "all",
		Venues: []domain.FundingVenueSnap{
			{Exchange: "binance", Current: domain.FundingCurrent{RatePct: "0.0100"}},
			{Exchange: "bybit", Current: domain.FundingCurrent{RatePct: "0.0200"}},
		},
	}}
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	c := &Checker{Alerts: svc, Tape: quiet, now: func() time.Time { return now }}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "fd", a.ID)
	if still.Status != domain.AlertStatusActive {
		t.Fatalf("under 0.05 on every venue should stay active %+v", still)
	}
	c.Tape = &fakeTape{fund: &domain.FundingSnapshot{
		Exchange: "all",
		Venues: []domain.FundingVenueSnap{
			{Exchange: "binance", Current: domain.FundingCurrent{RatePct: "0.0100"}},
			{Exchange: "bybit", Current: domain.FundingCurrent{RatePct: "0.0800"}},
			{Exchange: "okx", Current: domain.FundingCurrent{RatePct: "0.0300"}},
		},
	}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "fd", a.ID)
	if got.Status != domain.AlertStatusTriggered || got.TriggeredVenue != "bybit" || got.TriggeredPrice < 0.079 {
		t.Fatalf("bybit 0.08 should fire the all-exchange alert %+v", got)
	}
}
