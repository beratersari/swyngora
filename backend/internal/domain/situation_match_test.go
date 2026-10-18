package domain

import "testing"

func TestNormalizeSituationMatch_RequiresAKnownKind(t *testing.T) {
	if _, err := NormalizeSituationMatch(nil); err == nil {
		t.Fatal("nil match")
	}
	if _, err := NormalizeSituationMatch(&SituationMatch{}); err == nil {
		t.Fatal("empty match")
	}
	if _, err := NormalizeSituationMatch(&SituationMatch{PriceCVD: "sideways"}); err == nil {
		t.Fatal("unknown kind")
	}
	got, err := NormalizeSituationMatch(&SituationMatch{
		PriceCVD: "  " + SituationCVDPriceUp + " ",
		Funding:  "long",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.PriceCVD != SituationCVDPriceUp || got.Funding != "long" || got.Prints != "" {
		t.Fatalf("%+v", got)
	}
}

func TestSituationMatched_SelectedKindsOnly(t *testing.T) {
	want := &SituationMatch{PriceCVD: SituationCVDPriceUp, Funding: "long", Prints: SituationPrintLong}
	live := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long", RatePct: 0.02},
		Prints:   &SituationPrints{Side: SituationPrintLong, LongNotional: 90, ShortNotional: 4},
		Crowd:    &SituationCrowd{Lean: LongShortGapLeanTopShort},
		PrintAnchors: []PrintAnchor{{
			Anchor: PrintAnchorDay, Side: SituationPrintShort, LongNotional: 1, ShortNotional: 9, Complete: true,
		}},
	}
	met, cmp := SituationMatched(want, live)
	if !met || !cmp.Same || len(cmp.Clauses) != 3 {
		t.Fatalf("met=%v %+v", met, cmp)
	}
	if cmp.Clauses[0].Status != SituationMatchMet || cmp.Clauses[0].Clause != "priceCvd" || cmp.Clauses[1].Clause != "prints" || cmp.Clauses[2].Clause != "funding" {
		t.Fatalf("%+v", cmp.Clauses)
	}
	live.Prints.LongNotional = 1
	live.Prints.ShortNotional = 80
	live.Prints.Side = SituationPrintShort
	if met, _ := SituationMatched(want, live); met {
		t.Fatal("print side left")
	}
	live.Prints.Side = SituationPrintLong
	live.Funding = nil
	live.FundingMissing = []string{"okx"}
	if met, cmp := SituationMatched(want, live); met || len(cmp.Clauses) != 0 {
		t.Fatalf("blank funding counted: met=%v %+v", met, cmp)
	}
	if met, _ := SituationMatched(nil, live); met {
		t.Fatal("nil selection")
	}
	onlyCVD := &SituationMatch{PriceCVD: SituationCVDPriceUp}
	live.PriceCVD.Kind = SituationCVDAgrees
	if met, _ := SituationMatched(onlyCVD, live); met {
		t.Fatal("different kind")
	}
}
