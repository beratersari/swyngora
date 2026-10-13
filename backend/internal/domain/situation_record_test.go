package domain

import (
	"math"
	"testing"
	"time"
)

func TestBuildSituationRecord_CountsACloseUnderEveryStoredKind(t *testing.T) {
	read := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
	}
	early := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	later := early.Add(time.Minute)
	got := BuildSituationRecord([]SituationClose{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", ClosedAt: early, RealizedPnL: 20, Situation: read},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", ClosedAt: later, RealizedPnL: -10, Situation: read},
		{Exchange: ExchangeBybit, Symbol: "SOLUSDT", RealizedPnL: 0, Situation: read},
		{Exchange: ExchangeBybit, Symbol: "XRPUSDT", RealizedPnL: 4},
		{Exchange: ExchangeNasdaq, Symbol: "AAPL", ClosedAt: later, RealizedPnL: 50, Situation: read},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", ClosedAt: later, RealizedPnL: math.NaN(), Situation: read},
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
		if len(row.Closes) != 3 {
			t.Fatalf("closes %+v", row.Closes)
		}
		if row.Closes[0].Symbol != "ETHUSDT" || !row.Closes[0].ClosedAt.Equal(later) || row.Closes[0].RealizedPnL != -10 {
			t.Fatalf("newest %+v", row.Closes[0])
		}
		if row.Closes[1].Symbol != "BTCUSDT" || !row.Closes[1].ClosedAt.Equal(early) || row.Closes[1].RealizedPnL != 20 {
			t.Fatalf("older %+v", row.Closes[1])
		}
		if row.Closes[2].Symbol != "SOLUSDT" || !row.Closes[2].ClosedAt.IsZero() || row.Closes[2].Exchange != ExchangeBybit {
			t.Fatalf("untimed %+v", row.Closes[2])
		}
	}
	if got.Rows[0].Closes[0].Symbol != got.Rows[1].Closes[0].Symbol || got.Rows[0].Closes[1].RealizedPnL != got.Rows[1].Closes[1].RealizedPnL {
		t.Fatalf("rows should name the same sells: %+v %+v", got.Rows[0].Closes, got.Rows[1].Closes)
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
