package dataimport

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/importstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeAlerts struct {
	list []domain.PriceAlert
}

func (f *fakeAlerts) Create(_ context.Context, a domain.PriceAlert) (*domain.PriceAlert, error) {
	for _, x := range f.list {
		if x.ID == a.ID {
			return nil, domain.ErrInvalidArgument
		}
	}
	cp := a
	f.list = append(f.list, cp)
	return &cp, nil
}
func (f *fakeAlerts) Get(context.Context, string, string) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) ListByClient(context.Context, string) ([]domain.PriceAlert, error) {
	return append([]domain.PriceAlert(nil), f.list...), nil
}
func (f *fakeAlerts) ListActive(context.Context) ([]domain.PriceAlert, error) { return nil, nil }
func (f *fakeAlerts) MarkTriggered(context.Context, string, float64, time.Time) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) RecordRepeatingTrigger(context.Context, string, float64, time.Time) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) SetArmed(context.Context, string, bool) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) Delete(_ context.Context, _, id string) error {
	next := f.list[:0]
	for _, a := range f.list {
		if a.ID != id {
			next = append(next, a)
		}
	}
	f.list = next
	return nil
}
func (f *fakeAlerts) CountByClient(context.Context, string) (int, error) { return len(f.list), nil }
func (f *fakeAlerts) GetWebhook(context.Context, string) (*domain.ClientWebhook, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) SetWebhook(context.Context, string, domain.WebhookSettings) (*domain.ClientWebhook, error) {
	return nil, nil
}
func (f *fakeAlerts) DeleteWebhook(context.Context, string) error { return nil }
func (f *fakeAlerts) EnqueueNotification(context.Context, domain.AlertNotification) (*domain.AlertNotification, error) {
	return nil, nil
}
func (f *fakeAlerts) ListDueNotifications(context.Context, time.Time, int) ([]domain.AlertNotification, error) {
	return nil, nil
}
func (f *fakeAlerts) MarkNotificationDelivered(context.Context, string, time.Time) error {
	return nil
}
func (f *fakeAlerts) ScheduleNotificationRetry(context.Context, string, int, time.Time, string) error {
	return nil
}
func (f *fakeAlerts) FailNotification(context.Context, string, string) error { return nil }
func (f *fakeAlerts) GetNotificationByAlertID(context.Context, string) (*domain.AlertNotification, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) AddDigestItem(context.Context, string, string, string, string, time.Time) (*domain.AlertDigest, error) {
	return nil, nil
}
func (f *fakeAlerts) SealOpenDigests(context.Context, time.Time) (int, error) { return 0, nil }
func (f *fakeAlerts) ListDueDigests(context.Context, time.Time, int) ([]domain.AlertDigest, error) {
	return nil, nil
}
func (f *fakeAlerts) GetDigest(context.Context, string) (*domain.AlertDigest, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAlerts) ListDigestItems(context.Context, string) ([]domain.AlertDigestItem, error) {
	return nil, nil
}
func (f *fakeAlerts) MarkDigestDelivered(context.Context, string, time.Time) error { return nil }
func (f *fakeAlerts) ScheduleDigestRetry(context.Context, string, int, time.Time, string) error {
	return nil
}
func (f *fakeAlerts) FailDigest(context.Context, string, string) error { return nil }
func (f *fakeAlerts) PurgeClient(context.Context, string) error {
	f.list = nil
	return nil
}

type fakeScanner struct {
	backtests []domain.ScannerBacktest
	signals   map[string][]domain.ScannerBacktestSignal
}

func (f *fakeScanner) CreateRule(context.Context, domain.ScannerRule) (*domain.ScannerRule, error) {
	return nil, nil
}
func (f *fakeScanner) GetRule(context.Context, string, string) (*domain.ScannerRule, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeScanner) ListRules(context.Context, string) ([]domain.ScannerRule, error) {
	return nil, nil
}
func (f *fakeScanner) ListEnabledRules(context.Context) ([]domain.ScannerRule, error) {
	return nil, nil
}
func (f *fakeScanner) DeleteRule(context.Context, string, string) error { return nil }
func (f *fakeScanner) CountRules(context.Context, string) (int, error)  { return 0, nil }
func (f *fakeScanner) InsertResult(context.Context, domain.ScannerResult) (*domain.ScannerResult, bool, error) {
	return nil, false, nil
}
func (f *fakeScanner) ListResults(context.Context, string, int, int) ([]domain.ScannerResult, error) {
	return nil, nil
}
func (f *fakeScanner) CountResults(context.Context, string) (int, error) { return 0, nil }
func (f *fakeScanner) CreateBacktest(_ context.Context, b domain.ScannerBacktest) (*domain.ScannerBacktest, error) {
	for _, x := range f.backtests {
		if x.ID == b.ID {
			return nil, domain.ErrInvalidArgument
		}
	}
	cp := b
	f.backtests = append(f.backtests, cp)
	return &cp, nil
}
func (f *fakeScanner) GetBacktest(context.Context, string, string) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeScanner) FindActiveBacktest(context.Context, string, string, domain.Exchange, string, time.Time, time.Time) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeScanner) ListBacktests(_ context.Context, clientID string, limit, offset int) ([]domain.ScannerBacktest, error) {
	var out []domain.ScannerBacktest
	for _, b := range f.backtests {
		if b.ClientID == clientID {
			out = append(out, b)
		}
	}
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (f *fakeScanner) CountBacktests(_ context.Context, clientID string) (int, error) {
	n := 0
	for _, b := range f.backtests {
		if b.ClientID == clientID {
			n++
		}
	}
	return n, nil
}
func (f *fakeScanner) ListPendingBacktests(context.Context) ([]domain.ScannerBacktest, error) {
	return nil, nil
}
func (f *fakeScanner) ClaimBacktest(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func (f *fakeScanner) UpdateBacktestProgress(context.Context, string, int, int, int, float64) error {
	return nil
}
func (f *fakeScanner) FinishBacktest(_ context.Context, id string, status domain.ScannerBacktestStatus, signalCount int, errMsg string, _ time.Time) error {
	for i := range f.backtests {
		if f.backtests[i].ID == id {
			f.backtests[i].Status = status
			f.backtests[i].SignalCount = signalCount
			f.backtests[i].ErrorMessage = errMsg
		}
	}
	return nil
}
func (f *fakeScanner) CancelBacktest(context.Context, string, string, time.Time) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeScanner) GetBacktestStatus(context.Context, string) (domain.ScannerBacktestStatus, error) {
	return "", domain.ErrNotFound
}
func (f *fakeScanner) InsertBacktestSignal(_ context.Context, sig domain.ScannerBacktestSignal) error {
	if f.signals == nil {
		f.signals = map[string][]domain.ScannerBacktestSignal{}
	}
	f.signals[sig.BacktestID] = append(f.signals[sig.BacktestID], sig)
	return nil
}
func (f *fakeScanner) ListBacktestSignals(_ context.Context, backtestID string, limit, offset int) ([]domain.ScannerBacktestSignal, error) {
	sigs := f.signals[backtestID]
	if offset >= len(sigs) {
		return nil, nil
	}
	sigs = sigs[offset:]
	if limit > 0 && len(sigs) > limit {
		sigs = sigs[:limit]
	}
	return sigs, nil
}
func (f *fakeScanner) CountBacktestSignals(_ context.Context, backtestID string) (int, error) {
	return len(f.signals[backtestID]), nil
}
func (f *fakeScanner) DeleteBacktest(_ context.Context, clientID, id string) error {
	next := f.backtests[:0]
	found := false
	for _, b := range f.backtests {
		if b.ID == id && b.ClientID == clientID {
			found = true
			continue
		}
		next = append(next, b)
	}
	f.backtests = next
	if !found {
		return domain.ErrNotFound
	}
	delete(f.signals, id)
	return nil
}
func (f *fakeScanner) PurgeClient(context.Context, string) error {
	f.backtests = nil
	f.signals = map[string][]domain.ScannerBacktestSignal{}
	return nil
}

func sampleExportJSON() []byte {
	pl := map[string]any{
		"clientId": "other",
		"watchlist": map[string]any{
			"clientId": "other",
			"items": []map[string]any{
				{"exchange": "binance", "symbol": "BTCUSDT", "note": "n", "addedAt": time.Now().UTC().Format(time.RFC3339Nano)},
				{"exchange": "binance", "symbol": "ETHUSDT", "addedAt": time.Now().UTC().Format(time.RFC3339Nano)},
				{"exchange": "nope", "symbol": "X"}, // invalid
			},
		},
		"shares": []map[string]any{
			{"ownerClientId": "other", "granteeClientId": "friend1", "role": "viewer",
				"createdAt": time.Now().UTC().Format(time.RFC3339Nano), "updatedAt": time.Now().UTC().Format(time.RFC3339Nano)},
			{"ownerClientId": "other", "granteeClientId": "friend1", "role": "editor"}, // file dup
			{"ownerClientId": "other", "granteeClientId": "self", "role": "bad"},       // invalid role
		},
		"alerts": []map[string]any{
			{"id": "a1", "exchange": "binance", "symbol": "BTCUSDT", "condition": "above", "targetPrice": 100,
				"mode": "one_time", "status": "triggered", "createdAt": time.Now().UTC().Format(time.RFC3339Nano)},
		},
		"backtests": []map[string]any{
			{"id": "bt1", "ruleId": "r1", "exchange": "binance", "symbol": "BTCUSDT", "interval": "1h",
				"rangeStart": time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339Nano),
				"rangeEnd":   time.Now().UTC().Format(time.RFC3339Nano),
				"status": "completed", "signalCount": 1, "createdAt": time.Now().UTC().Format(time.RFC3339Nano),
				"signals": []map[string]any{
					{"id": "s1", "signalAt": time.Now().UTC().Format(time.RFC3339Nano), "closePrice": 50, "summary": "hit"},
				}},
		},
	}
	b, _ := json.Marshal(pl)
	return b
}

func newTestSvc(t *testing.T) (*Service, *watchliststore.Memory, *fakeAlerts, *fakeScanner) {
	t.Helper()
	wl := watchliststore.NewMemory()
	alerts := &fakeAlerts{}
	sc := &fakeScanner{signals: map[string][]domain.ScannerBacktestSignal{}}
	svc, err := New(importstore.NewMemory(), DataSources{Watchlist: wl, Alerts: alerts, Scanner: sc}, Options{
		FileDir: filepath.Join(t.TempDir(), "imp"), FileTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, wl, alerts, sc
}

func TestImport_PreviewMergeConfirm(t *testing.T) {
	svc, wl, alerts, sc := newTestSvc(t)
	ctx := context.Background()
	// Existing BTC so merge will skip one watchlist item
	_, _ = wl.Add(ctx, "user1", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", AddedAt: time.Now().UTC(),
	}, domain.WatchlistUnconditionalVersion)

	job, err := svc.Preview(ctx, PreviewInput{
		ClientID: "user1", FileName: "export.json", FileBytes: sampleExportJSON(), FormatHint: "json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != domain.ImportPreviewed {
		t.Fatalf("%+v", job)
	}
	wlSec := job.SectionCounts[domain.ExportSectionWatchlist]
	if wlSec.Valid != 2 || wlSec.Invalid != 1 || wlSec.WillAdd != 1 {
		t.Fatalf("watchlist counts %+v", wlSec)
	}
	if job.Totals.Invalid < 1 {
		t.Fatalf("totals %+v", job.Totals)
	}

	// Confirm merge
	job, err = svc.Confirm(ctx, "user1", job.ID, "merge")
	if err != nil || job.Status != domain.ImportPending {
		t.Fatalf("%+v %v", job, err)
	}
	// Second confirm blocked by active
	_, err = svc.Preview(ctx, PreviewInput{ClientID: "user1", FileName: "x.json", FileBytes: sampleExportJSON()})
	if err != nil {
		t.Fatal(err)
	}
	// Process
	if n, err := svc.ProcessPending(ctx); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	got, err := svc.Get(ctx, "user1", job.ID)
	if err != nil || got.Status != domain.ImportCompleted {
		t.Fatalf("%+v %v", got, err)
	}
	// ETH added
	list, _ := wl.Get(ctx, "user1")
	if len(list.Items) != 2 {
		t.Fatalf("items=%d", len(list.Items))
	}
	if len(alerts.list) != 1 || len(sc.backtests) != 1 {
		t.Fatalf("alerts=%d backtests=%d", len(alerts.list), len(sc.backtests))
	}
	if len(sc.signals["bt1"]) != 1 {
		t.Fatalf("signals=%+v", sc.signals)
	}
}

func TestImport_ReplaceAndNoDoubleShare(t *testing.T) {
	svc, wl, _, _ := newTestSvc(t)
	ctx := context.Background()
	_, _ = wl.Add(ctx, "user1", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance, Symbol: "SOLUSDT", AddedAt: time.Now().UTC(),
	}, domain.WatchlistUnconditionalVersion)
	_, _ = wl.CreateShare(ctx, domain.WatchlistShare{
		OwnerClientID: "user1", GranteeClientID: "oldfriend", Role: domain.WatchlistRoleViewer,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})

	job, err := svc.Preview(ctx, PreviewInput{ClientID: "user1", FileBytes: sampleExportJSON(), FormatHint: "json"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = svc.Confirm(ctx, "user1", job.ID, "replace")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessPending(ctx); err != nil {
		t.Fatal(err)
	}
	list, _ := wl.Get(ctx, "user1")
	// replace = only imported symbols (BTC, ETH)
	if len(list.Items) != 2 {
		t.Fatalf("items=%+v", list.Items)
	}
	shares, _ := wl.ListSharesByOwner(ctx, "user1")
	if len(shares) != 1 || shares[0].GranteeClientID != "friend1" {
		t.Fatalf("shares=%+v", shares)
	}
}

func TestImport_CancelPreview(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	ctx := context.Background()
	job, err := svc.Preview(ctx, PreviewInput{ClientID: "user1", FileBytes: sampleExportJSON()})
	if err != nil {
		t.Fatal(err)
	}
	canceled, err := svc.Cancel(ctx, "user1", job.ID)
	if err != nil || canceled.Status != domain.ImportCanceled {
		t.Fatalf("%+v %v", canceled, err)
	}
}

func TestImport_ConfirmConflictActive(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	ctx := context.Background()
	j1, err := svc.Preview(ctx, PreviewInput{ClientID: "user1", FileBytes: sampleExportJSON()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Confirm(ctx, "user1", j1.ID, "merge"); err != nil {
		t.Fatal(err)
	}
	j2, err := svc.Preview(ctx, PreviewInput{ClientID: "user1", FileBytes: sampleExportJSON()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Confirm(ctx, "user1", j2.ID, "merge")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("%v", err)
	}
}
