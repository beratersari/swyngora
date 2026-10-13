package domain

import (
	"math"
	"testing"
	"time"
)

func TestBuildSituationLeave_SplitsStillAndLeft(t *testing.T) {
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
	got := BuildSituationLeave([]SituationClose{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", ClosedAt: early, RealizedPnL: 20, Situation: entry, Exit: same},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", ClosedAt: later, RealizedPnL: -10, Situation: entry, Exit: leftPrice},
		{Exchange: ExchangeBybit, Symbol: "XRPUSDT", RealizedPnL: 4, Situation: entry},
		{Exchange: ExchangeBybit, Symbol: "SOLUSDT", RealizedPnL: 3, Exit: same},
		{Exchange: ExchangeNasdaq, Symbol: "AAPL", ClosedAt: later, RealizedPnL: 50, Situation: entry, Exit: same},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", ClosedAt: later, RealizedPnL: math.NaN(), Situation: entry, Exit: same},
	})
	if got.Counted != 2 || got.Skipped != 1 || got.Unreadable != 1 {
		t.Fatalf("counted=%d skipped=%d unreadable=%d", got.Counted, got.Skipped, got.Unreadable)
	}
	if len(got.Rows) != 3 {
		t.Fatalf("rows %+v", got.Rows)
	}
	if got.Rows[0].Clause != "priceCvd" || got.Rows[0].Kind != SituationCVDPriceUp {
		t.Fatalf("price %+v", got.Rows[0])
	}
	if got.Rows[0].Still.Count != 1 || got.Rows[0].Still.Profitable != 1 || got.Rows[0].Still.RealizedPnL != 20 {
		t.Fatalf("price still %+v", got.Rows[0].Still)
	}
	if got.Rows[0].Left.Count != 1 || got.Rows[0].Left.Profitable != 0 || got.Rows[0].Left.RealizedPnL != -10 {
		t.Fatalf("price left %+v", got.Rows[0].Left)
	}
	if got.Rows[0].Still.Closes[0].Symbol != "BTCUSDT" || got.Rows[0].Left.Closes[0].Symbol != "ETHUSDT" {
		t.Fatalf("price closes %+v %+v", got.Rows[0].Still.Closes, got.Rows[0].Left.Closes)
	}
	if got.Rows[1].Clause != "prints" || got.Rows[1].Kind != SituationPrintLong {
		t.Fatalf("prints %+v", got.Rows[1])
	}
	if got.Rows[1].Still.Count != 2 || got.Rows[1].Left.Count != 0 || got.Rows[1].Still.RealizedPnL != 10 {
		t.Fatalf("print dollars stay still %+v", got.Rows[1])
	}
	if len(got.Rows[1].Left.Closes) != 0 {
		t.Fatalf("empty left %+v", got.Rows[1].Left.Closes)
	}
	if got.Rows[2].Clause != "funding" || got.Rows[2].Still.Count != 2 || got.Rows[2].Left.Count != 0 {
		t.Fatalf("funding %+v", got.Rows[2])
	}
	if got.Rows[1].Still.Closes[0].Symbol != "ETHUSDT" || !got.Rows[1].Still.Closes[0].ClosedAt.Equal(later) {
		t.Fatalf("newest %+v", got.Rows[1].Still.Closes)
	}
	if got.Note == "" {
		t.Fatal("note")
	}
}

func TestBuildSituationLeave_BlankSellClauseIsLeft(t *testing.T) {
	got := BuildSituationLeave([]SituationClose{{
		Exchange:    ExchangeCoinbase,
		Symbol:      "BTCUSDT",
		RealizedPnL: 0,
		Situation: &CoinSituation{
			PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees},
			Funding:  &SituationFunding{Payer: "short"},
		},
		Exit: &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees}},
	}})
	if got.Counted != 1 || got.Unreadable != 0 || len(got.Rows) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Rows[0].Clause != "priceCvd" || got.Rows[0].Still.Count != 1 || got.Rows[0].Still.Profitable != 0 {
		t.Fatalf("still %+v", got.Rows[0])
	}
	if got.Rows[1].Clause != "funding" || got.Rows[1].Left.Count != 1 || got.Rows[1].Still.Count != 0 {
		t.Fatalf("missing funding %+v", got.Rows[1])
	}
}

func TestBuildSituationLeave_SellOnlyClauseStaysQuiet(t *testing.T) {
	got := BuildSituationLeave([]SituationClose{{
		Exchange:    ExchangeBinance,
		RealizedPnL: 2,
		Situation:   &CoinSituation{PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp}},
		Exit: &CoinSituation{
			PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
			Funding:  &SituationFunding{Payer: "long"},
		},
	}})
	if got.Counted != 1 || len(got.Rows) != 1 || got.Rows[0].Clause != "priceCvd" || got.Rows[0].Still.Count != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestBuildSituationLeave_EmptySellReadIsUnreadable(t *testing.T) {
	entry := &CoinSituation{Size: &SituationSize{Kind: SituationSizeFlat}}
	got := BuildSituationLeave([]SituationClose{
		{Exchange: ExchangeBinance, RealizedPnL: 1, Situation: entry, Exit: &CoinSituation{}},
		{Exchange: ExchangeBinance, RealizedPnL: 1, Situation: entry, Exit: nil},
	})
	if got.Counted != 0 || got.Unreadable != 2 || got.Skipped != 0 || len(got.Rows) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestBuildSituationLeave_EmptyBook(t *testing.T) {
	got := BuildSituationLeave(nil)
	if got.Counted != 0 || got.Skipped != 0 || got.Unreadable != 0 || len(got.Rows) != 0 || got.Note == "" {
		t.Fatalf("%+v", got)
	}
}
