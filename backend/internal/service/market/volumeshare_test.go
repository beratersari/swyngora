package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeFutVol struct {
	bars []domain.FuturesVolumeBar
	err  error
}

func (f *fakeFutVol) ListFuturesVolumeBars(_ context.Context, _ string) ([]domain.FuturesVolumeBar, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]domain.FuturesVolumeBar(nil), f.bars...), nil
}

func TestGetVolumeShare_MixAndChange(t *testing.T) {
	now := time.Now().UTC().Truncate(5 * time.Minute)
	bn := volumeShareBars(now, 55, 50)
	by := volumeShareBars(now, 30, 35)
	ok := volumeShareBars(now, 15, 15)
	svc := New(&fakeMarket{}, &fakeSupply{}).WithFuturesVolume(map[domain.Exchange]domain.FuturesVolumePort{
		domain.ExchangeBinance: &fakeFutVol{bars: bn},
		domain.ExchangeBybit:   &fakeFutVol{bars: by},
		domain.ExchangeOKX:     &fakeFutVol{bars: ok},
	})
	got, err := svc.GetVolumeShare(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var h1 domain.VolumeShareWindow
	for _, w := range got.Windows {
		if w.Window == domain.VolumeShareWindow1h {
			h1 = w
		}
	}
	if !h1.Complete || len(h1.Venues) != 3 || h1.Venues[0].Exchange != domain.ExchangeBinance {
		t.Fatalf("1h %+v", h1)
	}
	if h1.Venues[0].Kind != domain.VolumeShareGained {
		t.Fatalf("binance kind=%s %+v", h1.Venues[0].Kind, h1.Venues[0])
	}
}

func TestGetVolumeShare_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetVolumeShare(context.Background(), "all", ""); err == nil {
		t.Fatal("expected error")
	}
}

func volumeShareBars(now time.Time, cur, prior float64) []domain.FuturesVolumeBar {
	out := make([]domain.FuturesVolumeBar, 0, 24)
	for i := 0; i < 12; i++ {
		out = append(out, domain.FuturesVolumeBar{
			Time: now.Add(-time.Duration(i+1) * 5 * time.Minute), QuoteVolume: cur,
		})
	}
	for i := 0; i < 12; i++ {
		out = append(out, domain.FuturesVolumeBar{
			Time: now.Add(-time.Hour - time.Duration(i+1)*5*time.Minute), QuoteVolume: prior,
		})
	}
	return out
}
