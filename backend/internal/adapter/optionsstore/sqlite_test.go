package optionsstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLiteInsertListPurge(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "options.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	rec := domain.OptionsSample{
		Asset: "BTC", SampledAt: now, OpenInterest: 10, OpenInterestUSD: 1000,
		HasPutCallOI: true, PutCallOI: 1.1, HasDVOL: true, DVOL: 49,
		HasGamma: true, GammaNetUSD: -8e8, GammaRegime: domain.GammaRegimeShort,
		HasGammaFlip: true, GammaFlip: 97200, GammaFlipSide: domain.GammaFlipAbove,
		HasIndex: true, IndexPrice: 100000,
	}
	ok, err := s.InsertSample(context.Background(), rec)
	if err != nil || !ok {
		t.Fatalf("insert %v %v", ok, err)
	}
	ok, err = s.InsertSample(context.Background(), rec)
	if err != nil || ok {
		t.Fatalf("dup should update not insert %v %v", ok, err)
	}
	got, err := s.ListSamples(context.Background(), "BTCUSDT", now.Add(-time.Hour), now.Add(time.Hour), 10)
	if err != nil || len(got) != 1 || got[0].PutCallOI != 1.1 {
		t.Fatalf("list %+v %v", got, err)
	}
	if !got[0].HasGamma || got[0].GammaNetUSD != -8e8 || got[0].GammaFlip != 97200 {
		t.Fatalf("gamma list %+v", got[0])
	}
	n, err := s.PurgeOlderThan(context.Background(), now.Add(time.Minute))
	if err != nil || n != 1 {
		t.Fatalf("purge %d %v", n, err)
	}
}

func TestMemoryInsertList(t *testing.T) {
	t.Parallel()
	m := &Memory{}
	now := time.Now().UTC().Truncate(time.Minute)
	_, _ = m.InsertSample(context.Background(), domain.OptionsSample{Asset: "ETH", SampledAt: now, OpenInterest: 3})
	got, err := m.ListSamples(context.Background(), "ETH", now.Add(-time.Hour), now.Add(time.Hour), 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}
