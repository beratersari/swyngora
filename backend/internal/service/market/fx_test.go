package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type stubFx struct {
	rates map[string]float64
	err   error
	n     int
}

func (s *stubFx) LatestUSD(context.Context) (map[string]float64, time.Time, error) {
	s.n++
	if s.err != nil {
		return nil, time.Time{}, s.err
	}
	return s.rates, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), nil
}

func TestGetFxRates_CachesAndAliasesUSDT(t *testing.T) {
	src := &stubFx{rates: map[string]float64{domain.FxTRY: 40}}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithFx(src)
	a, err := svc.GetFxRates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a.Rates[domain.FxTRY] != 40 || a.Rates[domain.FxUSDT] != 1 {
		t.Fatalf("%+v", a.Rates)
	}
	_, _ = svc.GetFxRates(context.Background())
	if src.n != 1 {
		t.Fatalf("cache miss count=%d", src.n)
	}
}

func TestGetFxRates_FallbackWhenMissingSource(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	got, err := svc.GetFxRates(context.Background())
	if err != nil || got.Rates[domain.FxBaseUSD] != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}
