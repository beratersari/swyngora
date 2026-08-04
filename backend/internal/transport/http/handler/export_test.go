package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/exportstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
)

type emptyAlerts struct{}

func (emptyAlerts) Create(context.Context, domain.PriceAlert) (*domain.PriceAlert, error) {
	return nil, nil
}
func (emptyAlerts) Get(context.Context, string, string) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) ListByClient(context.Context, string) ([]domain.PriceAlert, error) {
	return nil, nil
}
func (emptyAlerts) ListActive(context.Context) ([]domain.PriceAlert, error) { return nil, nil }
func (emptyAlerts) MarkTriggered(context.Context, string, float64, time.Time) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) RecordRepeatingTrigger(context.Context, string, float64, time.Time) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) SetArmed(context.Context, string, bool) (*domain.PriceAlert, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) Delete(context.Context, string, string) error       { return nil }
func (emptyAlerts) CountByClient(context.Context, string) (int, error) { return 0, nil }
func (emptyAlerts) GetWebhook(context.Context, string) (*domain.ClientWebhook, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) SetWebhook(context.Context, string, domain.WebhookSettings) (*domain.ClientWebhook, error) {
	return nil, nil
}
func (emptyAlerts) DeleteWebhook(context.Context, string) error { return nil }
func (emptyAlerts) EnqueueNotification(context.Context, domain.AlertNotification) (*domain.AlertNotification, error) {
	return nil, nil
}
func (emptyAlerts) ListDueNotifications(context.Context, time.Time, int) ([]domain.AlertNotification, error) {
	return nil, nil
}
func (emptyAlerts) MarkNotificationDelivered(context.Context, string, time.Time) error {
	return nil
}
func (emptyAlerts) ScheduleNotificationRetry(context.Context, string, int, time.Time, string) error {
	return nil
}
func (emptyAlerts) FailNotification(context.Context, string, string) error { return nil }
func (emptyAlerts) GetNotificationByAlertID(context.Context, string) (*domain.AlertNotification, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) AddDigestItem(context.Context, string, string, string, string, time.Time) (*domain.AlertDigest, error) {
	return nil, nil
}
func (emptyAlerts) SealOpenDigests(context.Context, time.Time) (int, error) { return 0, nil }
func (emptyAlerts) ListDueDigests(context.Context, time.Time, int) ([]domain.AlertDigest, error) {
	return nil, nil
}
func (emptyAlerts) GetDigest(context.Context, string) (*domain.AlertDigest, error) {
	return nil, domain.ErrNotFound
}
func (emptyAlerts) ListDigestItems(context.Context, string) ([]domain.AlertDigestItem, error) {
	return nil, nil
}
func (emptyAlerts) MarkDigestDelivered(context.Context, string, time.Time) error { return nil }
func (emptyAlerts) ScheduleDigestRetry(context.Context, string, int, time.Time, string) error {
	return nil
}
func (emptyAlerts) FailDigest(context.Context, string, string) error { return nil }

type emptyScanner struct{}

func (emptyScanner) CreateRule(context.Context, domain.ScannerRule) (*domain.ScannerRule, error) {
	return nil, nil
}
func (emptyScanner) GetRule(context.Context, string, string) (*domain.ScannerRule, error) {
	return nil, domain.ErrNotFound
}
func (emptyScanner) ListRules(context.Context, string) ([]domain.ScannerRule, error) {
	return nil, nil
}
func (emptyScanner) ListEnabledRules(context.Context) ([]domain.ScannerRule, error) {
	return nil, nil
}
func (emptyScanner) DeleteRule(context.Context, string, string) error { return nil }
func (emptyScanner) CountRules(context.Context, string) (int, error)  { return 0, nil }
func (emptyScanner) InsertResult(context.Context, domain.ScannerResult) (*domain.ScannerResult, bool, error) {
	return nil, false, nil
}
func (emptyScanner) ListResults(context.Context, string, int, int) ([]domain.ScannerResult, error) {
	return nil, nil
}
func (emptyScanner) CountResults(context.Context, string) (int, error) { return 0, nil }
func (emptyScanner) CreateBacktest(context.Context, domain.ScannerBacktest) (*domain.ScannerBacktest, error) {
	return nil, nil
}
func (emptyScanner) GetBacktest(context.Context, string, string) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (emptyScanner) FindActiveBacktest(context.Context, string, string, domain.Exchange, string, time.Time, time.Time) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (emptyScanner) ListBacktests(context.Context, string, int, int) ([]domain.ScannerBacktest, error) {
	return nil, nil
}
func (emptyScanner) CountBacktests(context.Context, string) (int, error) { return 0, nil }
func (emptyScanner) ListPendingBacktests(context.Context) ([]domain.ScannerBacktest, error) {
	return nil, nil
}
func (emptyScanner) ClaimBacktest(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func (emptyScanner) UpdateBacktestProgress(context.Context, string, int, int, int, float64) error {
	return nil
}
func (emptyScanner) FinishBacktest(context.Context, string, domain.ScannerBacktestStatus, int, string, time.Time) error {
	return nil
}
func (emptyScanner) CancelBacktest(context.Context, string, string, time.Time) (*domain.ScannerBacktest, error) {
	return nil, domain.ErrNotFound
}
func (emptyScanner) GetBacktestStatus(context.Context, string) (domain.ScannerBacktestStatus, error) {
	return "", domain.ErrNotFound
}
func (emptyScanner) InsertBacktestSignal(context.Context, domain.ScannerBacktestSignal) error {
	return nil
}
func (emptyScanner) ListBacktestSignals(context.Context, string, int, int) ([]domain.ScannerBacktestSignal, error) {
	return nil, nil
}
func (emptyScanner) CountBacktestSignals(context.Context, string) (int, error) { return 0, nil }
func (emptyScanner) DeleteBacktest(context.Context, string, string) error {
	return domain.ErrNotFound
}

func newExportFixture(t *testing.T) (*ExportHandler, *exportsvc.Service) {
	t.Helper()
	wl := watchliststore.NewMemory()
	_, _ = wl.Add(context.Background(), "export-user", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", AddedAt: time.Now().UTC(),
	}, domain.WatchlistUnconditionalVersion)
	svc, err := exportsvc.New(exportstore.NewMemory(), exportsvc.DataSources{
		Watchlist: wl, Alerts: emptyAlerts{}, Scanner: emptyScanner{},
	}, exportsvc.Options{FileDir: filepath.Join(t.TempDir(), "ex"), FileTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return NewExportHandler(svc), svc
}

func TestExportHTTP_StartConflictProgressDownload(t *testing.T) {
	h, svc := newExportFixture(t)

	body, _ := json.Marshal(map[string]any{
		"clientId": "export-user", "format": "json", "sections": []string{"watchlist"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Start(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("start %d %s", rr.Code, rr.Body.String())
	}
	var job exportJobDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &job)
	if job.ID == "" || job.Status != "pending" {
		t.Fatalf("%+v", job)
	}

	// Second concurrent export → 409
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Start(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Process in background
	if n, err := svc.ProcessPending(context.Background()); err != nil || n != 1 {
		t.Fatalf("process n=%d err=%v", n, err)
	}

	// Get progress
	req = httptest.NewRequest(http.MethodGet, "/api/v1/export/"+job.ID+"?clientId=export-user", nil)
	req.SetPathValue("id", job.ID)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &job)
	if job.Status != "completed" || job.ProgressPct != 100 || job.DownloadURL == "" {
		t.Fatalf("%+v", job)
	}

	// Owner download
	req = httptest.NewRequest(http.MethodGet, "/api/v1/export/"+job.ID+"/download?clientId=export-user", nil)
	req.SetPathValue("id", job.ID)
	rr = httptest.NewRecorder()
	h.Download(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("download %d %s", rr.Code, rr.Body.String())
	}
	raw, _ := io.ReadAll(rr.Body)
	if !bytes.Contains(raw, []byte("export-user")) {
		t.Fatalf("body=%s", raw)
	}

	// Other user cannot download
	req = httptest.NewRequest(http.MethodGet, "/api/v1/export/"+job.ID+"/download?clientId=other-user", nil)
	req.SetPathValue("id", job.ID)
	rr = httptest.NewRecorder()
	h.Download(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("stranger download status=%d", rr.Code)
	}
}

func TestExportHTTP_Cancel(t *testing.T) {
	h, _ := newExportFixture(t)
	body, _ := json.Marshal(map[string]any{"clientId": "export-user", "format": "csv"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Start(rr, req)
	var job exportJobDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &job)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/export/"+job.ID+"/cancel?clientId=export-user", nil)
	req.SetPathValue("id", job.ID)
	rr = httptest.NewRecorder()
	h.Cancel(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &job)
	if job.Status != "canceled" {
		t.Fatalf("%+v", job)
	}
}
