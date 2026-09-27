package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type recoveryBooks struct {
	snaps []domain.BookHistorySnapshot
}

func (recoveryBooks) SnapshotAt(context.Context, string, string, time.Time) (*domain.BookHistorySnapshot, error) {
	return nil, domain.ErrNotFound
}
func (recoveryBooks) List(context.Context, domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return nil, nil
}
func (r recoveryBooks) ListLadders(_ context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	if q.From.IsZero() || q.To.IsZero() {
		return nil, nil
	}
	return r.snaps, nil
}
func (recoveryBooks) Compare(context.Context, string, string, time.Time, time.Time) (*domain.BookHistoryDiff, error) {
	return nil, domain.ErrNotFound
}
func (recoveryBooks) Note(string, string) {}

func TestGetSweepRecovery_EmptyWhenNoSweep(t *testing.T) {
	svc := New(gapBookPort{}, &fakeSupply{}).WithBookHistory(recoveryBooks{})
	got, err := svc.GetSweepRecovery(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Note == "" || len(got.Rows) != 0 {
		t.Fatalf("%+v", got)
	}
}
