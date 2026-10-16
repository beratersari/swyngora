package domain

import (
	"testing"
	"time"
)

func TestReasonHistory_KeepsTheSelectedLiveKind(t *testing.T) {
	entry := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
		Prints:   &SituationPrints{Side: SituationPrintLong, LongNotional: 10, Window: "24h"},
	}
	same := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
		Prints:   &SituationPrints{Side: SituationPrintLong, LongNotional: 80, Window: "24h"},
	}
	leftPrice := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceDown},
		Funding:  &SituationFunding{Payer: "long"},
		Prints:   &SituationPrints{Side: SituationPrintLong, LongNotional: 5, Window: "24h"},
	}
	early := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	later := early.Add(time.Minute)
	leave := BuildSituationLeave([]SituationClose{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", ClosedAt: early, RealizedPnL: 20, Situation: entry, Exit: same},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", ClosedAt: later, RealizedPnL: -10, Situation: entry, Exit: leftPrice},
	})
	live := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
		Prints:   &SituationPrints{Side: SituationPrintLong, LongNotional: 1, Window: "24h"},
	}
	got := ReasonHistory(leave, SituationReason{
		Set:     true,
		Clauses: []string{SituationClauseFunding, SituationClausePriceCVD, SituationClausePrints},
	}, live)
	if len(got) != 3 {
		t.Fatalf("rows %+v", got)
	}
	if got[0].Clause != SituationClausePriceCVD || got[0].Kind != SituationCVDPriceUp {
		t.Fatalf("price %+v", got[0])
	}
	if got[0].Still.Count != 1 || got[0].Still.Profitable != 1 || got[0].Still.RealizedPnL != 20 {
		t.Fatalf("price still %+v", got[0].Still)
	}
	if got[0].Left.Count != 1 || got[0].Left.RealizedPnL != -10 || got[0].Left.Closes[0].Symbol != "ETHUSDT" {
		t.Fatalf("price left %+v", got[0].Left)
	}
	if got[1].Clause != SituationClausePrints || got[1].Kind != SituationPrintLong {
		t.Fatalf("prints %+v", got[1])
	}
	if got[1].Still.Count != 2 || got[1].Left.Count != 0 || got[1].Still.RealizedPnL != 10 {
		t.Fatalf("print dollars stay still %+v", got[1])
	}
	if got[2].Clause != SituationClauseFunding || got[2].Still.Count != 2 || got[2].Left.Count != 0 {
		t.Fatalf("funding %+v", got[2])
	}
}

func TestReasonHistory_NamesALiveKindWithNoPastClose(t *testing.T) {
	leave := BuildSituationLeave([]SituationClose{{
		Exchange:    ExchangeBinance,
		Symbol:      "BTCUSDT",
		RealizedPnL: 4,
		Situation:   &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}},
		Exit:        &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}},
	}})
	live := &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees}}
	got := ReasonHistory(leave, SituationReason{Set: true, Clauses: []string{SituationClausePriceCVD}}, live)
	if len(got) != 1 || got[0].Clause != SituationClausePriceCVD || got[0].Kind != SituationCVDAgrees {
		t.Fatalf("%+v", got)
	}
	if got[0].Still.Count != 0 || got[0].Left.Count != 0 || got[0].Still.Closes == nil || got[0].Left.Closes == nil {
		t.Fatalf("empty groups %+v %+v", got[0].Still, got[0].Left)
	}
}

func TestReasonHistory_SkipsAClauseTheLiveReadLeftBlank(t *testing.T) {
	live := &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}}
	got := ReasonHistory(SituationLeave{}, SituationReason{
		Set:     true,
		Clauses: []string{SituationClausePriceCVD, SituationClauseFunding},
	}, live)
	if len(got) != 1 || got[0].Clause != SituationClausePriceCVD || got[0].Still.Count != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestReasonHistory_QuietWithoutAReason(t *testing.T) {
	live := &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}}
	if ReasonHistory(SituationLeave{}, SituationReason{}, live) != nil {
		t.Fatal("unset")
	}
	if ReasonHistory(SituationLeave{}, SituationReason{Set: true}, live) != nil {
		t.Fatal("empty")
	}
	if ReasonHistory(SituationLeave{}, SituationReason{Set: true, Clauses: []string{SituationClausePriceCVD}}, nil) != nil {
		t.Fatal("no live read")
	}
}
