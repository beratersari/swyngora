package domain

import (
	"testing"
	"time"
)

func TestOpenLotReads_NewestFirstAndSkipsABlankNewestEntry(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	older := &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}}
	newer := &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceDown}}
	closed := t0.Add(3 * time.Hour)
	lots := []TaxLot{
		{ID: "old", Quantity: 1, OpenedAt: t0, SourceTradeID: "buy-old"},
		{ID: "gone", Quantity: 1, OpenedAt: t0.Add(2 * time.Hour), SourceTradeID: "buy-gone", ClosedAt: &closed},
		{ID: "flat", Quantity: 0, OpenedAt: t0.Add(4 * time.Hour), SourceTradeID: "buy-flat"},
		{ID: "new", Quantity: 0.4, OpenedAt: t0.Add(time.Hour), SourceTradeID: "buy-new"},
	}
	byTrade := map[string]Trade{
		"buy-old": {ID: "buy-old", Situation: older, CreatedAt: t0.Add(time.Second)},
		"buy-new": {ID: "buy-new", Situation: newer, CreatedAt: t0.Add(time.Hour + time.Second)},
	}
	got := OpenLotReads(ExchangeBinance, lots, byTrade)
	if len(got) != 2 || !got[0].Newest || got[0].SourceTradeID != "buy-new" || got[0].Quantity != 0.4 {
		t.Fatalf("newest %+v", got)
	}
	if got[0].Situation == nil || got[0].Situation.PriceCVD.Kind != SituationCVDPriceDown {
		t.Fatalf("newest read %+v", got[0].Situation)
	}
	if got[0].OpenedAt.Equal(t0.Add(time.Hour)) {
		t.Fatal("opened at the lot clock instead of the fill")
	}
	if got[1].Newest || got[1].SourceTradeID != "buy-old" || got[1].Situation.PriceCVD.Kind != SituationCVDPriceUp {
		t.Fatalf("older %+v", got[1])
	}
	entry, at := EntryFromOpenReads(got)
	if entry == nil || entry.PriceCVD.Kind != SituationCVDPriceDown || !at.Equal(got[0].OpenedAt) {
		t.Fatalf("entry %+v %v", entry, at)
	}

	blank := OpenLotReads(ExchangeBinance, []TaxLot{
		{ID: "old", Quantity: 1, OpenedAt: t0, SourceTradeID: "buy-old"},
		{ID: "new", Quantity: 1, OpenedAt: t0.Add(time.Hour), SourceTradeID: "buy-new"},
	}, map[string]Trade{
		"buy-old": {ID: "buy-old", Situation: older, CreatedAt: t0},
	})
	if len(blank) != 2 || blank[0].Situation != nil || !blank[0].Newest {
		t.Fatalf("blank newest %+v", blank)
	}
	entry, at = EntryFromOpenReads(blank)
	if entry != nil || !at.IsZero() {
		t.Fatalf("blank entry %+v %v", entry, at)
	}
	if blank[1].Situation == nil || blank[1].Situation.PriceCVD.Kind != SituationCVDPriceUp {
		t.Fatalf("older kept %+v", blank[1].Situation)
	}

	if OpenLotReads(ExchangeNasdaq, lots, byTrade) != nil {
		t.Fatal("equity listed lots")
	}
	if entry, at := EntryFromOpenReads(nil); entry != nil || !at.IsZero() {
		t.Fatal("empty entry")
	}
}

func TestWithOpenLotComparisons_NamesEachBuy(t *testing.T) {
	live := &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees}}
	reads := []OpenLotRead{
		{ID: "new", Newest: true, Situation: &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceDown}}},
		{ID: "old", Situation: &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}}},
		{ID: "blank"},
	}
	got := WithOpenLotComparisons(reads, live)
	if len(got) != 3 || got[0].Comparison == nil || got[0].Comparison.Same || got[0].Comparison.Clauses[0].Live != SituationCVDAgrees {
		t.Fatalf("newest %+v", got[0].Comparison)
	}
	if got[1].Comparison == nil || got[1].Comparison.Clauses[0].Entry != SituationCVDPriceUp {
		t.Fatalf("older %+v", got[1].Comparison)
	}
	if got[2].Comparison == nil || got[2].Comparison.Same || len(got[2].Comparison.Clauses) != 0 {
		t.Fatalf("blank %+v", got[2].Comparison)
	}
	plain := OpenLotChanges(reads)
	if len(plain) != 3 || plain[0].Comparison != nil || plain[0].ID != "new" {
		t.Fatalf("plain %+v", plain)
	}
}
