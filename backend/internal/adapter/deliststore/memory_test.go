package deliststore

import (
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestMemoryReplaceAndLookup(t *testing.T) {
	m := NewMemory()
	t1 := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	m.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Exchange: domain.ExchangeBinance, Symbol: "acxusdt", DelistTime: t1},
		{Exchange: domain.ExchangeBinance, Symbol: "HFTUSDT", DelistTime: t1},
	})
	got, ok := m.DelistTime(domain.ExchangeBinance, "ACXUSDT")
	if !ok || !got.Equal(t1) {
		t.Fatalf("lookup ACXUSDT: ok=%v t=%v", ok, got)
	}
	if _, ok := m.DelistTime(domain.ExchangeBinance, "BTCUSDT"); ok {
		t.Fatal("BTC should not be scheduled")
	}
	list := m.List(domain.ExchangeBinance)
	if len(list) != 2 {
		t.Fatalf("list len=%d", len(list))
	}
}

func TestMemoryKeepsEarliestDelist(t *testing.T) {
	m := NewMemory()
	early := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	m.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "FOOUSDT", DelistTime: late},
		{Symbol: "FOOUSDT", DelistTime: early},
	})
	got, ok := m.DelistTime(domain.ExchangeBinance, "FOOUSDT")
	if !ok || !got.Equal(early) {
		t.Fatalf("want earliest %v got %v ok=%v", early, got, ok)
	}
}

func TestMemoryKeepsAnnouncementWhenMerging(t *testing.T) {
	m := NewMemory()
	halt := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	ann := time.Date(2026, 8, 20, 6, 0, 0, 0, time.UTC)
	m.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "ICXUSDT", DelistTime: halt},
		{Symbol: "ICXUSDT", DelistTime: halt.Add(time.Hour), AnnouncedAt: ann},
	})
	e, ok := m.Get(domain.ExchangeBinance, "ICXUSDT")
	if !ok || !e.DelistTime.Equal(halt) || !e.AnnouncedAt.Equal(ann) {
		t.Fatalf("entry=%+v ok=%v", e, ok)
	}
}
