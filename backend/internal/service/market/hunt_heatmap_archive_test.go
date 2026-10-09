package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// archiveHeatFut returns only the recent page unless the caller asks for a
// bucket. That is the newest-2,500 crowding the heatmap used to hit.
type archiveHeatFut struct {
	recent    []domain.FuturesSnapshot
	old       []domain.FuturesSnapshot
	sawBucket time.Duration
}

func (h *archiveHeatFut) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	if q.Metric == "liquidations" {
		return []domain.LiquidationEvent{}, nil
	}
	if q.Metric != domain.FuturesMetricOpenInterest {
		return []domain.FuturesSnapshot{}, nil
	}
	if q.Bucket > 0 {
		h.sawBucket = q.Bucket
		out := make([]domain.FuturesSnapshot, 0, len(h.old)+len(h.recent))
		out = append(out, h.old...)
		out = append(out, h.recent...)
		return out, nil
	}
	return append([]domain.FuturesSnapshot{}, h.recent...), nil
}

func TestGetLiquidationHuntHeatmap_LongWindowKeepsHourlyArchive(t *testing.T) {
	now := time.Now().UTC()
	var candles []domain.Candle
	for i := 0; i <= 31*6; i++ {
		ts := now.Add(-time.Duration(31*6-i) * 4 * time.Hour)
		candles = append(candles, domain.Candle{
			CloseTime: ts, Close: "100000", High: "100100", Low: "99900",
		})
	}
	var recent []domain.FuturesSnapshot
	for i := 0; i < 8; i++ {
		recent = append(recent, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance,
			Symbol: "BTCUSDT", SampledAt: now.Add(-time.Duration(i) * 6 * time.Hour), Value: 5_000_000,
		})
	}
	oldAt := now.Add(-20 * 24 * time.Hour)
	old := []domain.FuturesSnapshot{{
		Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance,
		Symbol: "BTCUSDT", SampledAt: oldAt, Value: 4_000_000,
	}}
	fut := &archiveHeatFut{recent: recent, old: old}
	svc := New(&fakeMarket{candles: candles, ticker: &domain.Ticker24h{LastPrice: "100000"}}, &fakeSupply{})
	svc.SetFuturesHistory(fut)
	got, err := svc.GetLiquidationHuntHeatmap(context.Background(), "binance", "BTCUSDT", "30d")
	if err != nil {
		t.Fatal(err)
	}
	if fut.sawBucket != 2*time.Hour {
		t.Fatalf("bucket %v", fut.sawBucket)
	}
	var earliest time.Time
	var oldColumn bool
	var hole bool
	for i, ok := range got.Binance.HasData {
		if i >= len(got.Times) {
			break
		}
		ts := got.Times[i]
		if ok && (earliest.IsZero() || ts.Before(earliest)) {
			earliest = ts
		}
		if ok && ts.Sub(oldAt) < 12*time.Hour && oldAt.Sub(ts) < 12*time.Hour {
			oldColumn = true
		}
		if !ok && ts.Sub(now.Add(-28*24*time.Hour)) < 6*time.Hour && now.Add(-28*24*time.Hour).Sub(ts) < 6*time.Hour {
			hole = true
		}
	}
	if !oldColumn || earliest.After(now.Add(-10*24*time.Hour)) {
		t.Fatalf("archive column missing earliest=%s old=%v cols=%d", earliest, oldColumn, got.Binance.ColumnsWithOI)
	}
	if !hole {
		t.Fatalf("a column with no sample should stay empty")
	}
}
