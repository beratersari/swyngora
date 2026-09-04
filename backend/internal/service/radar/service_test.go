package radar

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/radarstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeMkt struct {
	ticker  string
	candles []domain.Candle
	scan    *domain.ForensicsScanReport
}

func (f *fakeMkt) ScanForensics(context.Context, string, string, string, string, string, float64, int, string, string, string, string, string, string) (*domain.ForensicsScanReport, error) {
	return f.scan, nil
}
func (f *fakeMkt) GetTicker24h(context.Context, string, string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{LastPrice: f.ticker}, nil
}
func (f *fakeMkt) GetCandles(context.Context, string, string, string, int, *time.Time, *time.Time) ([]domain.Candle, error) {
	return f.candles, nil
}

func TestRecordHitsAndReview(t *testing.T) {
	store, err := radarstore.Open(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	mkt := &fakeMkt{ticker: "100"}
	svc := New(store, mkt)
	svc.now = func() time.Time { return now }
	rep := &domain.ForensicsScanReport{
		Exchange: "binance", Lookback: "3d", Window: "1h", AsOf: now,
		Hits: []domain.ForensicsScanHit{
			{Symbol: "BTCUSDT", Exchange: "binance", Similarity: 82, Lean: domain.CVDDirUp, WhySimilar: "OI matched"},
		},
	}
	n, err := svc.RecordHits(context.Background(), rep)
	if err != nil || n != 1 {
		t.Fatalf("insert %d %v", n, err)
	}
	n, err = svc.RecordHits(context.Background(), rep)
	if err != nil || n != 0 {
		t.Fatalf("dedupe %d %v", n, err)
	}
	rev, err := svc.Review(context.Background(), "binance", "", "7d", 20)
	if err != nil || len(rev.Cases) != 1 || rev.Cases[0].FoundPrice != 100 {
		t.Fatalf("%+v %v", rev, err)
	}
	if rev.Summary == "" || rev.Note == "" {
		t.Fatalf("summary %+v", rev)
	}
}

func TestFillDue_WritesReturn(t *testing.T) {
	store, err := radarstore.Open(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	found := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	due := found.Add(15 * time.Minute)
	c := domain.NewRadarCase(domain.ForensicsScanHit{Symbol: "ETHUSDT", Exchange: "binance", Lean: domain.CVDDirUp}, found, 50, "3d", "1h")
	c.ID = "eth1"
	if _, _, err := store.InsertIfQuiet(context.Background(), c, time.Minute); err != nil {
		t.Fatal(err)
	}
	mkt := &fakeMkt{ticker: "50", candles: []domain.Candle{
		{OpenTime: found, CloseTime: due, Close: "51"},
	}}
	svc := New(store, mkt)
	svc.now = func() time.Time { return due.Add(time.Minute) }
	n, err := svc.FillDue(context.Background(), due.Add(time.Minute), 10)
	if err != nil || n < 1 {
		t.Fatalf("fill %d %v", n, err)
	}
	rev, err := svc.Review(context.Background(), "", "ETHUSDT", "7d", 10)
	if err != nil || len(rev.Cases) != 1 {
		t.Fatal(err)
	}
	a15 := rev.Cases[0].After[0]
	if a15.FilledAt == nil || a15.Direction != domain.CVDDirUp || a15.Useful == nil || !*a15.Useful {
		t.Fatalf("after %+v", rev.Cases[0].After)
	}
}
