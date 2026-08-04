package export

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/exportstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeAlerts struct {
	list []domain.PriceAlert
}

func (f *fakeAlerts) Create(context.Context, domain.PriceAlert) (*domain.PriceAlert, error) {
	return nil, nil
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
func (f *fakeAlerts) Delete(context.Context, string, string) error { return nil }
func (f *fakeAlerts) CountByClient(context.Context, string) (int, error) {
	return len(f.list), nil
}
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
func (f *fakeScanner) CreateBacktest(context.Context, domain.ScannerBacktest) (*domain.ScannerBacktest, error) {
	return nil, nil
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
func (f *fakeScanner) FinishBacktest(context.Context, string, domain.ScannerBacktestStatus, int, string, time.Time) error {
	return nil
}
func (f *fakeScanner) CancelBacktest(context.Context, string, string, time.Time) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeScanner) GetBacktestStatus(context.Context, string) (domain.ScannerBacktestStatus, error) {
	return "", domain.ErrNotFound
}
func (f *fakeScanner) InsertBacktestSignal(context.Context, domain.ScannerBacktestSignal) error {
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
func (f *fakeScanner) DeleteBacktest(context.Context, string, string) error {
	return domain.ErrNotFound
}

func newTestService(t *testing.T) (*Service, *watchliststore.Memory) {
	t.Helper()
	dir := t.TempDir()
	wl := watchliststore.NewMemory()
	ctx := context.Background()
	_, _ = wl.Add(ctx, "user1", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "n", AddedAt: time.Now().UTC(),
	}, domain.WatchlistUnconditionalVersion)
	_, _ = wl.CreateShare(ctx, domain.WatchlistShare{
		OwnerClientID: "user1", GranteeClientID: "friend", Role: domain.WatchlistRoleViewer,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	alerts := &fakeAlerts{list: []domain.PriceAlert{{
		ID: "a1", ClientID: "user1", Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT",
		Condition: domain.AlertAbove, TargetPrice: 100, Mode: domain.AlertModeOneTime,
		Status: domain.AlertStatusTriggered, CreatedAt: time.Now().UTC(),
	}}}
	scanner := &fakeScanner{
		backtests: []domain.ScannerBacktest{{
			ID: "bt1", ClientID: "user1", RuleID: "r1", Exchange: domain.ExchangeBinance,
			Symbol: "BTCUSDT", Interval: "1h", Status: domain.BacktestCompleted,
			RangeStart: time.Now().Add(-24 * time.Hour), RangeEnd: time.Now(), CreatedAt: time.Now().UTC(),
		}},
		signals: map[string][]domain.ScannerBacktestSignal{
			"bt1": {{ID: "s1", BacktestID: "bt1", SignalAt: time.Now().UTC(), ClosePrice: 50, Summary: "hit"}},
		},
	}
	svc, err := New(exportstore.NewMemory(), DataSources{
		Watchlist: wl, Alerts: alerts, Scanner: scanner,
	}, Options{FileDir: filepath.Join(dir, "files"), FileTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return svc, wl
}

func TestExport_JSONCompleteAndDownload(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	job, err := svc.Start(ctx, StartInput{ClientID: "user1", Format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != domain.ExportPending {
		t.Fatalf("%+v", job)
	}

	// Second concurrent start rejected
	_, err = svc.Start(ctx, StartInput{ClientID: "user1", Format: "csv"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}

	if n, err := svc.ProcessPending(ctx); err != nil || n != 1 {
		t.Fatalf("process n=%d err=%v", n, err)
	}
	got, err := svc.Get(ctx, "user1", job.ID)
	if err != nil || got.Status != domain.ExportCompleted {
		t.Fatalf("%+v %v", got, err)
	}
	if got.ProgressPct != 100 || got.ByteSize <= 0 || got.FileName == "" {
		t.Fatalf("%+v", got)
	}

	// Download only for owner
	dl, err := svc.OpenDownload(ctx, "user1", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dl.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	var pl map[string]any
	if err := json.Unmarshal(raw, &pl); err != nil {
		t.Fatal(err)
	}
	if pl["clientId"] != "user1" {
		t.Fatalf("%v", pl)
	}
	// Stranger cannot get job
	_, err = svc.Get(ctx, "stranger", job.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestExport_CSVAndCancel(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	job, err := svc.Start(ctx, StartInput{
		ClientID: "user1", Format: "csv", Sections: []string{"watchlist", "shares"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Cancel while pending
	canceled, err := svc.Cancel(ctx, "user1", job.ID)
	if err != nil || canceled.Status != domain.ExportCanceled {
		t.Fatalf("%+v %v", canceled, err)
	}
	// Can start again after cancel
	job2, err := svc.Start(ctx, StartInput{ClientID: "user1", Format: "csv", Sections: []string{"alerts"}})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := svc.ProcessPending(ctx); err != nil || n != 1 {
		t.Fatalf("process n=%d err=%v", n, err)
	}
	got, _ := svc.Get(ctx, "user1", job2.ID)
	if got.Status != domain.ExportCompleted {
		t.Fatalf("%+v", got)
	}
	raw, _ := os.ReadFile(got.FilePath)
	if !contains(string(raw), "section:alerts") {
		t.Fatalf("csv missing alerts section: %s", raw)
	}
}

func TestExport_CleanupExpired(t *testing.T) {
	svc, _ := newTestService(t)
	// short TTL via direct finish
	ctx := context.Background()
	job, err := svc.Start(ctx, StartInput{ClientID: "user1", Format: "json", Sections: []string{"watchlist"}})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := svc.ProcessPending(ctx); err != nil || n != 1 {
		t.Fatalf("process n=%d err=%v", n, err)
	}
	got, _ := svc.Get(ctx, "user1", job.ID)
	// Force expiry in past
	past := time.Now().UTC().Add(-time.Minute)
	_ = svc.store.Finish(ctx, job.ID, domain.ExportCompleted, got.FileName, got.FilePath, got.ByteSize, &past, "", time.Now().UTC())
	n, err := svc.CleanupExpired(ctx)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	_, err = svc.Get(ctx, "user1", job.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(got.FilePath); !os.IsNotExist(err) {
		t.Fatalf("file should be gone: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
