package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type gridMem struct {
	snaps []domain.GridSnapshot
	cover []domain.LiquidationCoverage
}

func (m gridMem) ListGridSnapshots(context.Context, domain.GridRead) ([]domain.GridSnapshot, error) {
	return m.snaps, nil
}
func (m gridMem) ListGridTakerSums(context.Context, domain.GridRead) ([]domain.GridTakerSum, error) {
	return nil, nil
}
func (m gridMem) ListGridLiquidations(context.Context, domain.GridRead) ([]domain.GridLiquidationSum, error) {
	return nil, nil
}
func (m gridMem) ListLiquidationCoverage(context.Context) ([]domain.LiquidationCoverage, error) {
	return m.cover, nil
}

func TestGetDerivativesGrid_NamesTheMissingVenue(t *testing.T) {
	svc := New(nil, nil)
	now := time.Now().UTC()
	col := now.Truncate(time.Hour).Add(-time.Hour)
	at := col.Add(10 * time.Minute)
	svc.SetDerivativesGrid(gridMem{snaps: []domain.GridSnapshot{
		{Metric: domain.GridMetricFunding, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0001},
		{Metric: domain.GridMetricFunding, Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0001},
	}})
	got, err := svc.GetDerivativesGrid(context.Background(), "all", "funding", "24h", 5)
	if err != nil {
		t.Fatal(err)
	}
	var cell *domain.GridCell
	for i := range got.Rows {
		if got.Rows[i].Symbol != "BTCUSDT" {
			continue
		}
		for j := range got.Rows[i].Cells {
			if got.Rows[i].Cells[j].Time.Equal(col) {
				cell = &got.Rows[i].Cells[j]
			}
		}
	}
	if cell == nil || cell.Definite || len(cell.Missing) != 1 || cell.Missing[0] != "okx" {
		t.Fatalf("%+v rows %d", cell, len(got.Rows))
	}
}

func TestGetDerivativesGrid_RejectsUnknownMetric(t *testing.T) {
	svc := New(nil, nil)
	svc.SetDerivativesGrid(gridMem{})
	if _, err := svc.GetDerivativesGrid(context.Background(), "all", "price", "24h", 5); err == nil {
		t.Fatal("expected invalid metric")
	}
}
