package domain

import (
	"testing"
	"time"
)

func TestParseSituationReason_RejectsUnknownAndDropsDuplicates(t *testing.T) {
	got, err := ParseSituationReason([]string{" funding ", "priceCvd", "funding"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Set || got.HashPart() != "priceCvd,funding" {
		t.Fatalf("%+v", got)
	}
	if _, err := ParseSituationReason([]string{"nope"}); err == nil {
		t.Fatal("expected invalid clause")
	}
	empty, err := ParseSituationReason(nil)
	if err != nil || !empty.Set || empty.HashPart() != "-" {
		t.Fatalf("empty %+v %v", empty, err)
	}
	if (SituationReason{}).HashPart() != "" {
		t.Fatal("unset reason has no hash part")
	}
}

func TestKeepSituationClauses_KeepsOnlyTheNamedSet(t *testing.T) {
	full := &CoinSituation{
		Symbol:       "BTCUSDT",
		PriceCVD:     &SituationPriceCVD{Kind: SituationCVDPriceUp, Since: time.Now().UTC()},
		OpenInterest: &SituationOpenInterest{Flow: SituationOIOpened},
		Funding:      &SituationFunding{Payer: "long"},
	}
	kept := KeepSituationClauses(full, SituationReason{Set: true, Clauses: []string{SituationClausePriceCVD}})
	if kept == nil || kept.PriceCVD == nil || kept.PriceCVD.Kind != SituationCVDPriceUp || !kept.PriceCVD.Since.IsZero() {
		t.Fatalf("kept %+v", kept)
	}
	if kept.OpenInterest != nil || kept.Funding != nil {
		t.Fatalf("extra %+v", kept)
	}
	if full.PriceCVD.Since.IsZero() {
		t.Fatal("filter must not clear the caller's clock")
	}
	unset := KeepSituationClauses(full, SituationReason{})
	if unset != full || unset.OpenInterest == nil || unset.Funding == nil {
		t.Fatalf("unset %+v", unset)
	}
	if KeepSituationClauses(full, SituationReason{Set: true}) != nil {
		t.Fatal("empty reason must store nothing")
	}
	if KeepSituationClauses(full, SituationReason{Set: true, Clauses: []string{SituationClauseCrowd}}) != nil {
		t.Fatal("a claimed clause that is blank stores nothing")
	}
}
