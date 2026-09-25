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

func TestSQLiteStrikeBookReplaceAndLookup(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "options.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	early := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	late := early.Add(4 * time.Hour)
	points := []domain.OptionsStrikeOIPoint{
		{Expiry: exp, Strike: 100000, CallOI: 10, PutOI: 4},
		{Expiry: exp, Strike: 90000, CallOI: 1, PutOI: 8},
	}
	if err := s.ReplaceStrikeBook(context.Background(), "BTC", early, points); err != nil {
		t.Fatal(err)
	}
	points[0].CallOI = 25
	if err := s.ReplaceStrikeBook(context.Background(), "BTC", early, points[:1]); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceStrikeBook(context.Background(), "BTC", late, []domain.OptionsStrikeOIPoint{
		{Expiry: exp, Strike: 100000, CallOI: 30, PutOI: 4},
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.StrikeBookAt(context.Background(), "BTCUSDT", early.Add(time.Hour))
	if err != nil || !ok || !got.SampledAt.Equal(early) || len(got.Points) != 1 || got.Points[0].CallOI != 25 {
		t.Fatalf("at early+1h ok=%v %+v %v", ok, got, err)
	}
	later, ok, err := s.StrikeBookAt(context.Background(), "BTC", late)
	if err != nil || !ok || len(later.Points) != 1 || later.Points[0].CallOI != 30 {
		t.Fatalf("late %+v %v", later, err)
	}
	n, err := s.PurgeOlderThan(context.Background(), late.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("purge counted strike rows as desk samples: %d", n)
	}
	_, ok, err = s.StrikeBookAt(context.Background(), "BTC", late)
	if err != nil || ok {
		t.Fatalf("strike rows should be gone ok=%v %v", ok, err)
	}
}

func TestMemoryStrikeBook(t *testing.T) {
	t.Parallel()
	m := &Memory{}
	exp := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	at := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	if err := m.ReplaceStrikeBook(context.Background(), "ETH", at, []domain.OptionsStrikeOIPoint{
		{Expiry: exp, Strike: 2000, CallOI: 3, PutOI: 1},
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := m.StrikeBookAt(context.Background(), "ETH", at.Add(-time.Minute))
	if err != nil || ok {
		t.Fatalf("future-only book should miss ok=%v %v", ok, err)
	}
	got, ok, err = m.StrikeBookAt(context.Background(), "ETH", at.Add(time.Minute))
	if err != nil || !ok || len(got.Points) != 1 || got.Points[0].PutOI != 1 {
		t.Fatalf("%+v %v", got, err)
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
