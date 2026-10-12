package domain

import (
	"math"
	"testing"
)

func TestBuildSituationRecord_CountsACloseUnderEveryStoredKind(t *testing.T) {
	read := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
	}
	got := BuildSituationRecord([]SituationClose{
		{Exchange: ExchangeBinance, RealizedPnL: 20, Situation: read},
		{Exchange: ExchangeBinance, RealizedPnL: -10, Situation: read},
		{Exchange: ExchangeBinance, RealizedPnL: 0, Situation: read},
		{Exchange: ExchangeBybit, RealizedPnL: 4},
		{Exchange: ExchangeNasdaq, RealizedPnL: 50, Situation: read},
		{Exchange: ExchangeBinance, RealizedPnL: math.NaN(), Situation: read},
	})
	if got.Counted != 3 || got.Skipped != 1 {
		t.Fatalf("counted=%d skipped=%d", got.Counted, got.Skipped)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("rows %+v", got.Rows)
	}
	if got.Rows[0].Clause != "priceCvd" || got.Rows[0].Kind != SituationCVDPriceUp {
		t.Fatalf("first %+v", got.Rows[0])
	}
	if got.Rows[1].Clause != "funding" || got.Rows[1].Kind != "long" {
		t.Fatalf("second %+v", got.Rows[1])
	}
	for _, row := range got.Rows {
		if row.Count != 3 || row.Profitable != 1 || row.RealizedPnL != 10 {
			t.Fatalf("row %+v", row)
		}
	}
	if got.Note == "" {
		t.Fatal("note")
	}
}

func TestBuildSituationRecord_SortsClausesThenKinds(t *testing.T) {
	got := BuildSituationRecord([]SituationClose{
		{Exchange: ExchangeCoinbase, RealizedPnL: 1, Situation: &CoinSituation{
			Size:     &SituationSize{Kind: SituationSizeFlat},
			PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees},
		}},
		{Exchange: ExchangeCoinbase, RealizedPnL: -1, Situation: &CoinSituation{
			PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		}},
	})
	if len(got.Rows) != 3 {
		t.Fatalf("rows %+v", got.Rows)
	}
	want := []string{SituationCVDPriceUp, SituationCVDAgrees, SituationSizeFlat}
	for i, kind := range want {
		if got.Rows[i].Kind != kind {
			t.Fatalf("order %+v", got.Rows)
		}
	}
}

func TestBuildSituationRecord_EmptyBook(t *testing.T) {
	got := BuildSituationRecord(nil)
	if got.Counted != 0 || got.Skipped != 0 || len(got.Rows) != 0 || got.Note == "" {
		t.Fatalf("%+v", got)
	}
}
